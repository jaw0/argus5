// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Aug-29 12:11 (EDT)
// Function: scheduler

package sched

import (
	"math/rand"
	"sort"
	"sync"
	"time"

	"argus/clock"
	"argus/config"
)

type Conf struct {
	Freq  int
	Phase int
	Auto  bool
	Text  string
}

type D struct {
	freq  int
	phase int
	auto  bool
	text  string
	when  int64
	obj   Starter
	locte *te
}

type Starter interface {
	Start()
}

const (
	MAXWORKER = 100 // override in config file
)

var schedchan = make(chan *D, 1000)
var workchan = make(chan *D, 1000)
var stopchan = make(chan struct{})
var done sync.WaitGroup

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

	d.ReSchedule(0)
	return d
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

func (d *D) ReSchedule(delay int) {

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

	schedchan <- d
}

func (d *D) Remove() {
	d.when = 0
	schedchan <- d
}

func Init() {

	done.Add(1)
	go mainloop()

	cf := config.Cf()
	nwork := cf.Mon_maxrun

	if nwork <= 0 {
		nwork = MAXWORKER
	}

	for i := 0; i < nwork; i++ {
		done.Add(1)
		go worker()
	}
}

func Stop() {
	close(stopchan)
}

func Wait() {
	<-stopchan
	done.Wait()
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
	tock := time.NewTicker(time.Second)
	defer tock.Stop()

	for {
		select {
		case <-stopchan:
			return
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
}

func worker() {

	defer done.Done()

	for {
		select {
		case <-stopchan:
			return
		case d := <-workchan:
			d.obj.Start()

			if d.auto {
				d.ReSchedule(0)
			}
		}
	}
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
			workchan <- d
		}
	}

	schedule = schedule[i:]
}
