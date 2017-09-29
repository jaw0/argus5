// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Sep-03 11:23 (EDT)
// Function: always up - for testing

package isforced

import (
	"argus/argus"
	"argus/configure"
	"argus/service"
)

type D struct {
	Value bool
	Osc   int
	count int
}

func init() {
	// register with service factory
	service.Register("Force", New)
}

func New(conf *configure.CF, s *service.Service) service.Monitor {
	return &D{Value: true}
}

func (d *D) PreConfig(conf *configure.CF, s *service.Service) error {
	return nil
}
func (d *D) Config(conf *configure.CF, s *service.Service) error {

	conf.InitFromConfig(d, "force", "")
	s.SetNames("Force", "UP", "Testy McTestface")
	return nil
}

func (d *D) Init() error {
	return nil
}

func (d *D) Hostname() string {
	return ""
}
func (d *D) Recycle() {
}
func (d *D) DoneConfig() {
}

func (d *D) Start(s *service.Service) {

	if d.Osc > 0 {
		d.count = (d.count + 1) % d.Osc

		if d.count > d.Osc/2 {
			s.SetResult(argus.CRITICAL, "0", "OHNOS")
		} else {
			s.SetResult(argus.CLEAR, "1", "OK")
		}
	} else if d.Value {
		s.SetResult(argus.CLEAR, "1", "OK")

	} else {
		s.SetResult(argus.CRITICAL, "0", "OHNOS")

	}

	s.Done()
}

func (d *D) Abort() {

}

func (d *D) DumpInfo() map[string]interface{} {
	return nil
}
