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
	"argus/config"
	"argus/construct"
	"argus/darp"
	"argus/diag"
	"argus/monel"
	_ "argus/monitor"
	"argus/monitor/ping"
	"argus/notify"
	"argus/resolv"
	"argus/sched"
	"argus/web"
)

var dl = diag.Logger("main")
var shutdown = make(chan int)
var sigchan = make(chan os.Signal, 5)
var exitvalue = 0

func init() {
	api.Add(false, "/hello", func(c *api.Context) { c.Conn.Write([]byte("hello, api\n")) })

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

	diag.Verbose("starting....")

	// init sighandlers
	signal.Notify(sigchan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGHUP, syscall.SIGUSR2)
	go sigHandle()

	ping.Init()
	sched.Init()
	resolv.Init()
	notify.Init()

	// start, http, test servers
	web.Init()

	// change user/group

	// init stats dir, etal
	createStatsDirs()

	// read large config
	cf := config.Cf()
	construct.ReadConfig(cf.Monitor_config)

	// prepare web serving
	web.Configured()
	api.Init()           // start local api server
	darp.Init(&MakeIt{}) // after config is loaded

	// block + wait
	sched.Wait()

	// finish
	diag.Verbose("shutting down...")
	notify.Stop()
	monel.Stop()
	diag.Verbose("stopped")
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
			sched.Stop()
		}
	}
}

func createStatsDirs() {

	cf := config.Cf()

	if cf.Datadir == "" {
		return
	}

	os.Mkdir(cf.Datadir+"/stats", 0777)

	for a := 'A'; a <= 'Z'; a++ {
		dir := fmt.Sprintf("%s/stats/%c", cf.Datadir, a)
		os.Mkdir(dir, 0777)
		for b := 'A'; b <= 'Z'; b++ {
			dir := fmt.Sprintf("%s/stats/%c/%c", cf.Datadir, a, b)
			os.Mkdir(dir, 0777)
		}
	}
}
