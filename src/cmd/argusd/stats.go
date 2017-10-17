// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Sep-25 20:47 (EDT)
// Function: stats

package main

import (
	"expvar"
	"syscall"
	"time"

	"argus/clock"
)

const LAMBDA = 10
const DELAY = 30 * time.Second

var monrate = expvar.NewFloat("monrate")
var cpurate = expvar.NewFloat("cpurate")
var uptime = expvar.NewInt("uptime")

func statsCollector() {

	runs := expvar.Get("runs").(*expvar.Int)
	lambda := 0.0
	var prun int64
	var mr float64
	idle := 1.0
	var pusage syscall.Rusage
	syscall.Getrusage(0, &pusage)

	for {
		time.Sleep(DELAY)

		lambda++
		if lambda > LAMBDA {
			lambda = LAMBDA
		}

		// uptime
		uptime.Set(clock.Unix() - starttime)

		// monitoring per second
		crun := runs.Value()
		drun := crun - prun
		prun = crun
		cmr := float64(drun) / 30

		if mr == 0 {
			mr = cmr
		}

		mr = (lambda*mr + cmr) / (lambda + 1)
		monrate.Set(mr)

		// cpu/idle
		var usage syscall.Rusage
		syscall.Getrusage(0, &usage)
		dutime := usage.Utime.Nano() - pusage.Utime.Nano()
		dstime := usage.Stime.Nano() - pusage.Stime.Nano()
		pusage = usage

		dl.Debug("usage: u %d, s %d", dutime, dstime)
		cidle := float64(int64(DELAY)-dutime-dstime) / float64(DELAY)
		if cidle < 0 {
			cidle = 0
		}
		if cidle > 1 {
			cidle = 1
		}

		idle = (lambda*idle + cidle) / (lambda + 1)
		cpurate.Set(idle)

	}
}
