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
		return nil, fmt.Errorf("unknown service '%s'", "XXX")
	}

	s := &Service{}
	s.cf = defaults
	s.p.Statuses = make(map[string]argus.Status)
	s.p.Results = make(map[string]string)
	s.check = check()

	s.mon = monel.New(s, parent)

	err := s.mon.Config(conf)
	if err != nil {
		return nil, err
	}

	// override monel.defaults
	if parent != nil {
		parent.Cf.Sendnotify = true
		parent.Cf.Countstop = true
	}

	return s.mon, nil
}

func (s *Service) Config(conf *configure.CF) error {

	conf.InitFromConfig(&s.cf, "service", "")

	hwab := false
	for i := argus.CLEAR; i <= argus.CRITICAL; i++ {
		if !math.IsNaN(s.cf.Maxdeviation[i]) {
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

	// uname

	if s.cf.Frequency == 0 {
		s.cf.Frequency = 60
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

	s.sched = sched.New(&sched.Conf{
		Freq: s.cf.Frequency,
		Auto: true,
		Text: s.mon.Unique(),
	}, s)

	return nil
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
	construct func() Monitor
}

var monitorProbe []probeCf

func Register(name string, construct func() Monitor) {
	monitorProbe = append(monitorProbe, probeCf{strings.ToLower(name), construct})
}

func probe(name string) func() Monitor {

	name = strings.ToLower(name)
	var bestc func() Monitor
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
