// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Sep-03 00:02 (EDT)
// Function: service construction

package service

import (
	"argus/argus"
	"argus/configure"
	"argus/monel"
)

func New(conf *configure.CF, parent *monel.M) (*monel.M, error) {

	// probe
	// run := ...

	s := &Service{}
	s.cf = defaults
	s.p.Statuses = make(map[string]argus.Status)
	s.p.Results = make(map[string]string)

	s.mon = monel.New(s, parent)

	err := s.mon.Config(conf)
	if err != nil {
		return nil, err
	}

	return s.mon, nil
}

func (s *Service) Config(conf *configure.CF) error {

	// conf.InitFromConfig(&s.cf, "service", prefix)
	// hwab
	err := s.run.Config(conf)
	if err != nil {
		return err
	}

	return nil
}

func (s *Service) Init() error {

	// schedule
	return nil
}
