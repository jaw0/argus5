// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Sep-03 00:02 (EDT)
// Function: service construction

package service

import (
	"fmt"
	"math"
	"strings"

	"argus/argus"
	"argus/configure"
	"argus/monel"
	"argus/sched"
)

// construction starts here:
func New(conf *configure.CF, parent *monel.M) (*monel.M, error) {

	check := probe(conf.Name)

	if check == nil {
		return nil, fmt.Errorf("unknown service '%s'", conf.Name)
	}

	s := &Service{}
	s.Cf = defaults
	s.p.Statuses = make(map[string]argus.Status)
	s.p.Results = make(map[string]string)
	s.check = check(conf, s)

	s.mon = monel.New(s, parent)

	// override monel.defaults
	s.mon.Cf.Sendnotify[int(argus.UNKNOWN)] = argus.ScheduleAlwaysYes
	s.mon.Cf.Countstop = true

	err := s.mon.Config(conf)
	if err != nil {
		return nil, err
	}

	return s.mon, nil
}

func (s *Service) Config(conf *configure.CF) error {

	conf.InitFromConfig(&s.Cf, "service", "")

	hwab := false
	for i := argus.CLEAR; i <= argus.CRITICAL; i++ {
		if !math.IsNaN(s.Cf.Maxdeviation[i]) {
			hwab = true
		}
	}
	if hwab {
		s.HwabConfig(conf)
	}

	err := s.check.Config(conf, s)
	if err != nil {
		return err
	}

	if s.Cf.Frequency == 0 {
		s.Cf.Frequency = 60
	}

	return nil
}

func (s *Service) Init() error {

	if s.p.Hwab != nil {
		s.p.Hwab.Init()
	}

	err := s.check.Init()
	if err != nil {
		return err
	}

	// QQQ - or wait until DoneConfig?
	s.sched = sched.New(&sched.Conf{
		Freq: s.Cf.Frequency,
		Auto: true,
		Text: s.mon.Unique(),
	}, s)

	return nil
}

func (s *Service) DoneConfig() {
	s.check.DoneConfig()
}

// destruction
func (s *Service) Recycle() {

	if s.sched != nil {
		s.sched.Remove()
	}
	s.check.Recycle()
}

// ################################################################

type probeCf struct {
	name      string
	construct func(*configure.CF, *Service) Monitor
}

var monitorProbe []probeCf

func Register(name string, construct func(*configure.CF, *Service) Monitor) {
	monitorProbe = append(monitorProbe, probeCf{strings.ToLower(name), construct})
}

func probe(name string) func(*configure.CF, *Service) Monitor {

	name = strings.ToLower(name)
	var bestc func(*configure.CF, *Service) Monitor
	var bestl int

	for i, _ := range monitorProbe {
		p := &monitorProbe[i]

		if strings.HasPrefix(name, p.name) && len(p.name) > bestl {
			bestl = len(p.name)
			bestc = p.construct
		}
	}

	return bestc
}
