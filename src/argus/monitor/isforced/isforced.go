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
}

func init() {
	// register with service factory
	service.Register("Force/UP", New)
}

func New() service.Monitor {
	return &D{}
}

func (d *D) Config(conf *configure.CF, s *service.Service) error {

	s.SName = "Force/UP"
	return nil
}

func (d *D) Init() error {
	return nil
}

func (d *D) Recycle() {
}

func (d *D) Start(s *service.Service) {

	s.SetResult(argus.CLEAR, "1", "OK")
	s.Done()
}

func (d *D) Abort() {

}
