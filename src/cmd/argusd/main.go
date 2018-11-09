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
	"os/user"
	"runtime"
	"strconv"
	"syscall"
	"time"

	"github.com/jaw0/acgo/daemon"
	"github.com/jaw0/acgo/diag"

	_ "argus/agent"
	"argus/api"
	"argus/argus"
	"argus/clock"
	"argus/config"
	"argus/construct"
	"argus/darp"
	"argus/graph/graphd"
	"argus/monel"
	_ "argus/monitor"
	"argus/monitor/ping"
	"argus/notify"
	"argus/resolv"
	"argus/sched"
	"argus/sec"
	"argus/service"
	"argus/testport"
	"argus/web"
)

const WEEK = 7 * 24 * 3600 * time.Second

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
	var rootcert string
	var controlsock string
	var agentport int

	flag.StringVar(&configfile, "c", "", "config file")
	flag.BoolVar(&foreground, "f", false, "run in foreground")
	flag.StringVar(&rootcert, "A", "", "run in agent mode using the specified root cert")
	flag.IntVar(&agentport, "p", 0, "tcp port for agent mode")
	flag.StringVar(&controlsock, "s", "", "control socket")
	flag.Parse()

	if !foreground {
		daemon.Ize()
	}

	diag.Init("argusd")

	// load small base config
	if configfile != "" {
		config.Load(configfile)
	}
	cf := config.Cf()

	if controlsock != "" {
		cf.Control_Socket = controlsock
	}
	if rootcert != "" {
		cf.Agent_Mode = true
		if configfile == "" {
			diag.SetConfig(&diag.Config{Facility: "daemon"})
		}
	}

	argus.Loggit("", "Argus starting")
	diag.Verbose("starting....")

	// init sighandlers
	signal.Notify(sigchan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGHUP, syscall.SIGUSR2)
	go sigHandle()
	// set up env, etal
	raiseFileLimit()
	if runtime.GOMAXPROCS(0) < 128 {
		runtime.GOMAXPROCS(128)
	}
	os.Setenv("ARGUS_PID", fmt.Sprintf("%d", os.Getpid()))
	os.Setenv("ARGUS_VER", argus.Version)

	sec.Init(rootcert)
	ping.Init()
	sched.Init()
	resolv.Init()
	notify.Init()
	graphd.Init()

	// start, http, test servers
	web.Init()
	testport.Start(cf.Port_test)

	changeUser()

	// init stats dir, etal
	createStatsDirs()
	createGdataDirs()
	createNotifyDirs()
	initCleanDirs()

	// read large config
	if cf.Monitor_config != "" {
		files := construct.ReadConfig(cf.Monitor_config)

		if cf.Auto_Reload && !foreground {
			go watchFiles(files)
		}
	}

	if cf.DevMode {
		api.Add(true, "trace", apiTrace)
	}

	// prepare web serving
	web.Configured()
	api.Init()           // start local api server
	darp.Init(&MakeIt{}) // after config is loaded

	if cf.Agent_Mode {
		darp.Agent(agentport)
	}

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

func changeUser() {

	cf := config.Cf()

	if cf.User != "" {
		usr, err := user.Lookup(cf.User)
		if err != nil {
			dl.Fatal("invalid user '%s': %v", cf.User, err)
		}
		uid, _ := strconv.Atoi(usr.Uid)

		err = syscall.Setreuid(uid, uid)
		if err != nil {
			dl.Fatal("cannot change user: %v", err)
		}
	}

	if cf.Group != "" {
		grp, err := user.LookupGroup(cf.Group)
		if err != nil {
			dl.Fatal("invalid group '%s': %v", cf.Group, err)
		}
		gid, _ := strconv.Atoi(grp.Gid)

		err = syscall.Setregid(gid, gid)
		if err != nil {
			dl.Fatal("cannot change group: %v", err)
		}
	}
}

func raiseFileLimit() {

	cf := config.Cf()
	// attempt to increase to max possible
	limit := syscall.Rlimit{0xFFFFFFFF, 0xFFFFFFFF} // RLIM_INFINITY is broken on linux
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
			switch n {
			case syscall.SIGUSR2:
				diag.Bug("usr2")
				continue
			case syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT:
				exitvalue = daemon.ExitFinished
			case syscall.SIGHUP:
				exitvalue = daemon.ExitRestart
			default:
				exitvalue = daemon.ExitRestart
			}
			sched.Stop()
		}
	}
}

func watchFiles(files []string) {

	start := time.Now()

	for {
		for _, f := range files {
			info, err := os.Stat(f)
			if err != nil {
				continue
			}

			t := info.ModTime()

			if t.After(start) {
				dl.Verbose("config file '%s' changed - restarting", f)
				sigchan <- syscall.SIGHUP
				return
			}
		}

		time.Sleep(30 * time.Second)
	}
}

func createStatsDirs() {
	createDirs("stats")
}

func createGdataDirs() {
	createDirs("gdata")
}

func createNotifyDirs() {

	cf := config.Cf()

	if cf.Datadir == "" {
		return
	}

	fdir := cf.Datadir + "/notify"

	err := mkdir(fdir)
	if err != nil {
		dl.Fatal("cannot create '%s': %v", fdir, err)
	}
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
		err := mkdir(dir)
		if err != nil {
			dl.Fatal("cannot create '%s': %v", dir, err)
		}
		for b := 'A'; b <= 'Z'; b++ {
			dir := fmt.Sprintf("%s/%c/%c", fdir, a, b)
			err := mkdir(dir)
			if err != nil {
				dl.Fatal("cannot create '%s': %v", dir, err)
			}
		}
	}
}

func initCleanDirs() {

	sched.NewFunc(&sched.Conf{
		Freq: 6 * 3600,
		Text: "file cleanup",
		Auto: true,
	}, func() {
		cleanDirs("stats", 2*WEEK)
		cleanDirs("gdata", 2*WEEK)
	})
}

// remove old orphaned files
func cleanDirs(dir string, age time.Duration) {

	cf := config.Cf()

	if cf.Datadir == "" {
		return
	}

	now := time.Now()
	limit := now.Add(-age)

	fdir := cf.Datadir + "/" + dir

	for a := 'A'; a <= 'Z'; a++ {
		for b := 'A'; b <= 'Z'; b++ {
			dir := fmt.Sprintf("%s/%c/%c", fdir, a, b)

			f, err := os.Open(dir)
			if err != nil {
				continue
			}

			files, _ := f.Readdirnames(-1)
			f.Close()

			for _, file := range files {
				f := dir + "/" + file
				info, err := os.Stat(f)
				if err != nil {
					continue
				}

				if info.ModTime().Before(limit) {
					os.Remove(f)
				}
			}
		}
	}
}

func mkdir(dir string) error {

	info, err := os.Stat(dir)
	if err == nil && info.IsDir() {
		// already exists. great
		return nil
	}

	err = os.Mkdir(dir, 0777)
	return err
}

func apiHup(ctx *api.Context) {

	sigchan <- syscall.SIGHUP
	ctx.SendOKFinal()
}

func apiTrace(ctx *api.Context) {
	dl.Bug("trace request")
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
	ctx.SendKVP("system", runtime.Version()+"/"+runtime.GOOS+"/"+runtime.GOARCH)
	if darp.MyDarp != nil {
		ctx.SendKVP("darp", darp.MyDarp.Name+" "+darp.MyDarp.Type)
	}
	ctx.SendKVP("objects", monel.NMonel.String())
	ctx.SendKVP("services", service.NService.String())
	ctx.SendKVP("alerts", notify.NActive.String())
	ctx.SendKVP("uptime", argus.Elapsed(clock.Unix()-starttime))
	ctx.SendKVP("monrate", fmt.Sprintf("%.2f %.2f %.2f per second", monrate[0].Value(), monrate[1].Value(), monrate[2].Value()))
	ctx.SendKVP("idle", fmt.Sprintf("%.2f%% %.2f%% %.2f%%", 100*cpurate[0].Value(), 100*cpurate[1].Value(), 100*cpurate[2].Value()))

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
