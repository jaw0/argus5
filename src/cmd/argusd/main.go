// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Aug-28 21:47 (EDT)
// Function: argus main

package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

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

	web.Add(web.PUBLIC, "/hello", func(c *web.Context) { c.W.Write([]byte("hello, web!\n")) })
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
	construct.ReadConfig(cf.Monitor_config)

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
	ctx.SendKVP("objects", monel.NMonel.String())
	ctx.SendKVP("services", service.NService.String())
	ctx.SendKVP("alerts", notify.NActive.String())
	ctx.SendKVP("uptime", argus.Elapsed(clock.Unix()-starttime))
	ctx.SendKVP("monrate", fmt.Sprintf("%.2f per second", monrate.Value()))
	ctx.SendKVP("idle", fmt.Sprintf("%.2f%%", 100*idlerate.Value()))

	// RSN - darp info

	ctx.SendFinal()
}
