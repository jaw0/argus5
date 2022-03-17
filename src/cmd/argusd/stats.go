// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Sep-25 20:47 (EDT)
// Function: stats

package main

import (
	"expvar"
	"syscall"
	"time"

	"argus.domain/argus/clock"
)

type taUsage struct {
	utime int64
	stime int64
}

const DELAY = 10

var LAMBDA = []float64{60 / DELAY, 300 / DELAY, 900 / DELAY}

// NB - idle = 1 - cpu
var monrate = []*expvar.Float{expvar.NewFloat("monrate"), expvar.NewFloat("monrate5"), expvar.NewFloat("monrate15")}
var cpurate = []*expvar.Float{expvar.NewFloat("cpurate"), expvar.NewFloat("cpurate5"), expvar.NewFloat("cpurate15")}
var idlerate = []*expvar.Float{expvar.NewFloat("idlerate"), expvar.NewFloat("idlerate5"), expvar.NewFloat("idlerate15")}
var uptime = expvar.NewInt("uptime")

func statsCollector() {

	runs := expvar.Get("runs").(*expvar.Int)
	lambda := []float64{0, 0, 0}
	var prun int64
	pusage := getUsage()

	for {
		time.Sleep(DELAY * time.Second)

		// uptime
		uptime.Set(clock.Unix() - starttime)

		// monitoring per second
		crun := runs.Value()
		drun := crun - prun
		prun = crun
		cmr := float64(drun) / DELAY

		// cpu/idle
		curr := getUsage()

		dutime := curr.utime - pusage.utime
		dstime := curr.stime - pusage.stime

		ccpu := float64(dutime+dstime) / float64(DELAY*time.Second)

		pusage = curr

		if ccpu < 0 {
			ccpu = 0
		}
		if ccpu > 1 {
			ccpu = 1
		}

		cidle := 1 - ccpu
		dl.Debug("usage: u %d, s %d; %v", dutime, dstime, cidle)

		for i, l := range lambda {

			mr := (l*monrate[i].Value() + cmr) / (l + 1)
			monrate[i].Set(mr)

			idle := (l*idlerate[i].Value() + cidle) / (l + 1)
			idlerate[i].Set(idle)

			cpu := (l*cpurate[i].Value() + ccpu) / (l + 1)
			cpurate[i].Set(cpu)

			dl.Debug("L %v, mon %v -> %v idle %v -> %v", l, cmr, mr, cidle, idle)
		}

		for i := range lambda {
			if lambda[i] < LAMBDA[i] {
				lambda[i]++
			}
		}
	}
}

func getUsage() taUsage {

	var self, childn syscall.Rusage
	syscall.Getrusage(0, &self)
	syscall.Getrusage(-1, &childn)

	return taUsage{
		utime: self.Utime.Nano() + childn.Utime.Nano(),
		stime: self.Stime.Nano() + childn.Stime.Nano(),
	}
}
