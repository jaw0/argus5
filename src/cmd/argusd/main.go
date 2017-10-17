// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Aug-28 21:47 (EDT)
// Function: argus main

package main

import (
	"expvar"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	_ "argus/agent"
	"argus/api"
	"argus/argus"
	"argus/clock"
	"argus/config"
	"argus/construct"
	"argus/darp"
	"argus/diag"
	"argus/graph/graphd"
	"argus/monel"
	_ "argus/monitor"
	"argus/monitor/ping"
	"argus/notify"
	"argus/resolv"
	"argus/sched"
	"argus/service"
	"argus/testport"
	"argus/web"
)

var dl = diag.Logger("main")
var shutdown = make(chan int)
var sigchan = make(chan os.Signal, 5)
var exitvalue = 0
var starttime = clock.Unix()
var status = "starting"

func init() {
	api.Add(true, "hup", apiHup)
	api.Add(true, "reload", apiHup)
	api.Add(true, "shutdown", apiStop)
	api.Add(true, "status", apiStatus)
	api.Add(true, "self", apiExpvar)
}

func main() {
	var configfile string
	var foreground bool

	flag.StringVar(&configfile, "c", "/dev/null", "config file")
	flag.BoolVar(&foreground, "f", false, "run in foreground")
	flag.Parse()

	if !foreground {
		daemonize()
	}

	diag.Init("argusd")

	// load small base config
	config.Load(configfile)
	cf := config.Cf()

	argus.Loggit("", "Argus starting")
	diag.Verbose("starting....")

	// init sighandlers
	signal.Notify(sigchan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGHUP, syscall.SIGUSR2)
	go sigHandle()
	// set up env, etal
	raiseFileLimit()
	os.Setenv("ARGUS_PID", fmt.Sprintf("%d", os.Getpid()))
	os.Setenv("ARGUS_VER", argus.Version)

	ping.Init()
	sched.Init()
	resolv.Init()
	notify.Init()
	graphd.Init()

	// start, http, test servers
	web.Init()
	testport.Start(cf.Port_test)

	// change user/group

	// init stats dir, etal
	createStatsDirs()
	createGdataDirs()

	// read large config
	if cf.Monitor_config != "" {
		construct.ReadConfig(cf.Monitor_config)
	}

	// prepare web serving
	web.Configured()
	api.Init()           // start local api server
	darp.Init(&MakeIt{}) // after config is loaded
	go statsCollector()

	argus.Loggit("", "Argus running")
	status = "running"
	// block + wait
	sched.Wait()

	// finish
	diag.Verbose("shutting down...")
	status = "shutting down"
	notify.Stop()
	monel.Stop()
	diag.Verbose("stopped")
	argus.Loggit("", "Argus exiting")
	os.Exit(exitvalue)
}

func raiseFileLimit() {

	cf := config.Cf()
	// attempt to increase to max possible
	limit := syscall.Rlimit{syscall.RLIM_INFINITY, syscall.RLIM_INFINITY}
	syscall.Setrlimit(syscall.RLIMIT_NOFILE, &limit)
	// how many do we actually have?
	syscall.Getrlimit(syscall.RLIMIT_NOFILE, &limit)

	warn := 100
	if cf.Mon_maxrun+10 > warn {
		warn = cf.Mon_maxrun + 10
	}

	if int(limit.Cur) < warn {
		dl.Verbose("open file limit is only %d. reducing maxrun to compensate.", limit.Cur)
		cf.Mon_maxrun = int(limit.Cur) - 10
	}
}

func sigHandle() {

	for {
		select {
		case n := <-sigchan:
			if n == syscall.SIGUSR2 {
				diag.Bug("usr2")
				continue
			}
			exitvalue = 1
			sched.Stop()
		}
	}
}

func createStatsDirs() {
	createDirs("stats")
}

func createGdataDirs() {
	createDirs("gdata")
}

func createDirs(dir string) {

	cf := config.Cf()

	if cf.Datadir == "" {
		return
	}

	fdir := cf.Datadir + "/" + dir

	os.Mkdir(fdir, 0777)

	for a := 'A'; a <= 'Z'; a++ {
		dir := fmt.Sprintf("%s/%c", fdir, a)
		os.Mkdir(dir, 0777)
		for b := 'A'; b <= 'Z'; b++ {
			dir := fmt.Sprintf("%s/%c/%c", fdir, a, b)
			os.Mkdir(dir, 0777)
		}
	}
}

func apiHup(ctx *api.Context) {

	sigchan <- syscall.SIGHUP
	ctx.SendOKFinal()
}

func apiStop(ctx *api.Context) {

	ctx.SendOKFinal()
	exitvalue = 0
	sched.Stop()
}

func apiStatus(ctx *api.Context) {

	ctx.SendOK()
	ctx.SendKVP("status", status)
	ctx.SendKVP("version", argus.Version)
	ctx.SendKVP("OS", runtime.GOOS)
	ctx.SendKVP("objects", monel.NMonel.String())
	ctx.SendKVP("services", service.NService.String())
	ctx.SendKVP("alerts", notify.NActive.String())
	ctx.SendKVP("uptime", argus.Elapsed(clock.Unix()-starttime))
	ctx.SendKVP("monrate", fmt.Sprintf("%.2f per second", monrate.Value()))
	ctx.SendKVP("idle", fmt.Sprintf("%.2f%%", 100*cpurate.Value()))

	// RSN - darp info

	ctx.SendFinal()
}

func apiExpvar(ctx *api.Context) {

	ctx.SendOK()

	expvar.Do(func(kv expvar.KeyValue) {
		if kv.Key == "memstats" || kv.Key == "cmdline" {
			return
		}
		ctx.SendKVP(kv.Key, kv.Value.String())
	})

	ctx.SendFinal()
}
