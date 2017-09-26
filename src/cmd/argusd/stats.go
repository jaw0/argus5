// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Sep-25 20:47 (EDT)
// Function: stats

package main

import (
	"expvar"
	"syscall"
	"time"
)

var monrate = expvar.NewFloat("monrate")
var idlerate = expvar.NewFloat("idlerate")

func statsCollector() {

	runs := expvar.Get("runs").(*expvar.Int)
	var prun int64
	var mr float64
	var idle float64
	var pusage syscall.Rusage
	syscall.Getrusage(0, &pusage)

	for {
		time.Sleep(30 * time.Second)

		// monitoring per second
		crun := runs.Value()
		drun := crun - prun
		prun = crun
		cmr := float64(drun) / 30

		if mr == 0 {
			mr = cmr
		}

		mr = (10*mr + cmr) / 11
		monrate.Set(mr)

		// cpu/idle
		var usage syscall.Rusage
		syscall.Getrusage(0, &usage)
		dutime := usage.Utime.Sec - pusage.Utime.Sec
		dstime := usage.Stime.Sec - pusage.Stime.Sec
		pusage = usage

		cidle := float64(30-dutime-dstime) / 30
		idle = (10*idle + cidle) / 11
		idlerate.Set(idle)

	}
}
