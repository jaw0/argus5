// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Aug-29 12:11 (EDT)
// Function: scheduler

package sched

import (
	"expvar"
	"math/rand"
	"runtime"
	"sort"
	"sync"
	"time"

	"argus/argus"
	"argus/clock"
	"argus/config"
	"argus/diag"
)

const (
	PRIO_NONE    = 0
	PRIO_EXPRESS = 1
	PRIO_HIGH    = 2
)

type Conf struct {
	Freq  int
	Phase int
	Auto  bool
	Prio  bool // true=high prio; false=auto
	Text  string
}

type D struct {
	freq  int
	phase int
	prio  int
	auto  bool
	xdt   int
	text  string
	when  int64
	obj   Starter
	locte *te
}

type Starter interface {
	Start()
}

const (
	MINWORKER     = 2
	MINEXPRESS    = 2
	NUMWORKER     = 100 // override in config file
	MAXWORKER     = 10000
	QUEUELEN      = 1000
	EXPRLEN       = 1000
	PRIOLEN       = 10
	EXPRESS_LIMIT = 50 // millisecs. RSN - autotune
)

var schedchan = make(chan *D, QUEUELEN)
var workchan = make(chan *D, QUEUELEN)
var expresschan = make(chan *D, EXPRLEN)
var priochan = make(chan *D, PRIOLEN)
var stopchan = make(chan struct{})
var done sync.WaitGroup
var dl = diag.Logger("sched")

var NRun = expvar.NewInt("runs")
var PrioRun = expvar.NewInt("runsprio")
var ExprRun = expvar.NewInt("runsexpress")
var SchedIdle = expvar.NewInt("workidle")
var SchedQueue = expvar.NewInt("schedqueue")
var WorkQueue = expvar.NewInt("workqueue")
var PrioQueue = expvar.NewInt("workqueueprio")
var ExprQueue = expvar.NewInt("workqueueexpress")
var WorkDefer = expvar.NewInt("workdefer")
var ExprDefer = expvar.NewInt("workdeferexpress")
var PrioDefer = expvar.NewInt("workdeferprio")

func New(cf *Conf, obj Starter) *D {

	if cf.Phase <= 0 {
		cf.Phase = rand.Intn(cf.Freq)
	}
	cf.Phase %= cf.Freq

	d := &D{
		freq:  cf.Freq,
		phase: cf.Phase,
		auto:  cf.Auto,
		text:  cf.Text,
		obj:   obj,
	}

	if cf.Prio {
		d.prio = PRIO_HIGH
	}

	d.ReSchedule(0, 0)
	return d
}

func At(unix int64, text string, f func()) {

	d := &D{
		when: unix,
		text: text,
		auto: false,
		prio: PRIO_HIGH,
		obj:  Func(f),
	}
	schedchan <- d
}

func NewFunc(cf *Conf, f func()) *D {
	return New(cf, Func(f))
}

type schedFunc struct {
	f func()
}

func Func(f func()) schedFunc {
	return schedFunc{f}
}
func (f schedFunc) Start() {
	f.f()
}

func (d *D) ReSchedule(delay int, exp int) {

	// prioritize based on expected runtime
	if d.xdt == 0 {
		d.xdt = exp
	}
	d.xdt = (3*d.xdt + exp) / 4

	if d.prio != PRIO_HIGH {
		switch {
		case d.xdt == 0:
			// initial scheduling, no runtime data, no priority
			d.prio = PRIO_NONE
		case d.xdt < EXPRESS_LIMIT:
			d.prio = PRIO_EXPRESS
		default:
			d.prio = PRIO_NONE
		}
	}

	d.ReScheduleSamePrio(delay)
}

func (d *D) ReScheduleSamePrio(delay int) {

	now := clock.Unix()

	freq := int64(d.freq)
	if freq <= 0 {
		freq = 60
	}

	if delay > 0 {
		d.when = now + int64(delay)
	} else {
		d.when = ((now-int64(d.phase))/freq)*freq + freq + int64(d.phase)
		for d.when < now {
			d.when += freq
		}
	}

	if d.when%3600 == 0 {
		// delay to avoid top of the hour busy time
		d.when += int64(rand.Intn(4) + 1)
	}

	SchedQueue.Set(int64(len(schedchan)))
	schedchan <- d
}

func (d *D) Remove() {
	d.when = 0
	schedchan <- d
}

func (d *D) sendWork() bool {

	switch d.prio {
	case PRIO_HIGH:
		select {
		case priochan <- d:
			PrioRun.Add(1)
			return true
		default:
		}
		PrioDefer.Add(1)
		fallthrough
	case PRIO_EXPRESS:
		select {
		case expresschan <- d:
			ExprRun.Add(1)
			return true
		default:
		}
		ExprDefer.Add(1)
		fallthrough
	default:
		select {
		case workchan <- d:
			return true
		default:
			return false
		}
	}
}

func (d *D) ASAP() {

	// if the work queue has room, send it straight in
	// otherwise, reschedule it
	if !d.sendWork() {
		WorkDefer.Add(1)
		d.ReScheduleSamePrio(1)
	}
}

func startWorker(dev bool, ch chan *D) {
	done.Add(1)
	go worker(dev, ch)
}

func Init() {

	done.Add(1)
	go mainloop()

	cf := config.Cf()
	nwork := cf.Mon_maxrun

	if nwork == 0 {
		nwork = NUMWORKER
	}
	if nwork < MINWORKER {
		nwork = MINWORKER
	}
	if nwork > MAXWORKER {
		nwork = MAXWORKER
	}

	for i := 0; i < nwork; i++ {
		startWorker(cf.DevMode, workchan)
	}

	nexpr := nwork / 10
	if nexpr < MINEXPRESS {
		nexpr = MINEXPRESS
	}
	for i := 0; i < nexpr; i++ {
		startWorker(cf.DevMode, expresschan)
	}

	startWorker(cf.DevMode, priochan)
	startWorker(cf.DevMode, priochan)

	go autotune(nwork, nexpr, cf.Mon_maxrun, cf.DevMode)
}

func Stop() {

	select {
	case <-stopchan:
		// already shutting down
		break
	default:
		close(stopchan)
	}
}

func Wait() {
	<-stopchan
	// done.Wait() QQQ
}

// ################################################################

type te struct {
	when int64
	todo map[*D]bool
}

// NB - the schedule is read+modified only by the mainloop goro
var schedule []*te

func mainloop() {

	defer done.Done()
	tock := time.NewTicker(time.Second / 10)
	defer tock.Stop()

	for {
		SchedQueue.Set(int64(len(schedchan)))
		select {
		case <-stopchan:
			break
		case d := <-schedchan:
			if d.locte != nil {
				d.del()
			}
			if d.when != 0 {
				d.add()
			}

		case <-tock.C:
			dispatch()
		}
	}

	// shutting down, discard requests
	for {
		select {
		case <-schedchan:
		}
	}
}

func worker(devmode bool, work chan *D) {

	defer done.Done()

	for {
		amIdle(true)
		WorkQueue.Set(int64(len(workchan) + len(expresschan) + len(priochan)))
		ExprQueue.Set(int64(len(expresschan)))
		PrioQueue.Set(int64(len(priochan)))

		select {
		case d := <-work:
			amIdle(false)
			d.run(devmode)

			if d.auto {
				d.ReScheduleSamePrio(0)
			}
		}
	}
}

func (d *D) run(devmode bool) {

	if !devmode {
		// if we are not in dev mode, try to recover from crashes
		defer func() {
			if err := recover(); err != nil {
				dl.Bug("CRASH RECOVERY - '%s'", d.text)
			}
		}()
	}

	NRun.Add(1)
	d.obj.Start()
}

func (d *D) add() {
	if d.when == 0 {
		return
	}

	i := sort.Search(len(schedule), func(j int) bool {
		return schedule[j].when >= d.when
	})

	switch {
	case i == len(schedule):
		// append to end
		l := &te{when: d.when, todo: map[*D]bool{d: true}}
		d.locte = l
		schedule = append(schedule, l)
	case schedule[i].when == d.when:
		// append to existing bucket
		d.locte = schedule[i]
		schedule[i].todo[d] = true
	default:
		// insert new bucket
		l := &te{when: d.when, todo: map[*D]bool{d: true}}
		d.locte = l
		schedule = append(schedule, nil)
		copy(schedule[i+1:], schedule[i:])
		schedule[i] = l
	}
}

func (d *D) del() {

	if d.locte == nil {
		return
	}

	t := d.locte.todo
	delete(t, d)
	d.locte = nil
}

func dispatch() {

	var i int
	now := clock.Unix()

	for i = 0; i < len(schedule); i++ {
		if schedule[i] == nil {
			continue
		}

		if schedule[i].when > now {
			break
		}

		for d, _ := range schedule[i].todo {
			d.locte = nil
			WorkQueue.Set(int64(len(workchan) + len(expresschan) + len(priochan)))
			ExprQueue.Set(int64(len(expresschan)))
			PrioQueue.Set(int64(len(priochan)))

			if d.sendWork() {
				delete(schedule[i].todo, d)
				runtime.Gosched()
			} else {
				// queue full - take a break...
				schedule = schedule[i:]
				WorkDefer.Add(1)
				return
			}
		}
	}

	schedule = schedule[i:]
}

func amIdle(y bool) {

	if y {
		SchedIdle.Add(1)
	} else {
		SchedIdle.Add(-1)
	}
}

// increase number of workers if needed
// if user set mon_maxrun, only issue a warning, do not actually adjust
func autotune(nwork int, nexpr int, max int, dev bool) {

	const MINLIM = 60
	const MAXLIM = 600
	const SEC = 10

	ovlim := MINLIM
	ovct := 0
	pdeferd := WorkDefer.Value()
	pexpdfr := ExprDefer.Value()

	time.Sleep(60 * time.Second)

	for {
		time.Sleep(SEC * time.Second)

		deferd := WorkDefer.Value()
		expdfr := ExprDefer.Value()

		// no idle workers? queues full?
		if deferd == pdeferd {
			if ovct == 0 && ovlim < MAXLIM {
				ovlim++
			}
			if ovct > 0 {
				ovct -= SEC
			}
			continue
		}

		exprok := pexpdfr == expdfr

		pdeferd = deferd
		pexpdfr = expdfr
		ovct += SEC

		if ovct < ovlim {
			continue
		}

		if nwork < MAXWORKER && max == 0 {
			more := nwork / 5
			for i := 0; i < more; i++ {
				startWorker(dev, workchan)
				nwork++
			}

			if !exprok {
				more = (more + 5) / 10
				for i := 0; i < more; i++ {
					startWorker(dev, expresschan)
					nexpr++
				}
			}

			dl.Verbose("increasing number of workers: %d + %d", nwork, nexpr)
		} else {
			dl.Problem("argus overload - frequency, mon_maxrun, or faster server")
			argus.ConfigWarning("", 0, "argus overload - frequency, mon_maxrun, or faster server")
			return
		}

		ovct = 0
	}
}
