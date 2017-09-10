// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Sep-09 16:55 (EDT)
// Function: monitor tcp

package tcp

import (
	"argus/configure"
	"argus/diag"
	"argus/service"
)

type Conf struct {
	Hostname string
	Port     int
	Send     string
	Expect   string
	ReadHow  string
	SSL      bool
}

type TCP struct {
	cf Conf
}

func init() {
	// register with service factory
	diag.Verbose("tcp new")
	service.Register("TCP", New)
}

func New() service.Monitor {
	t := &TCP{}
	t.InitNew()
	return t
}

func (t *TCP) InitNew() {
	// ...
}

func (d *TCP) Config(conf *configure.CF, s *service.Service) error {

	conf.InitFromConfig(&d.cf, "tcp", "")
	diag.Verbose("tcp config")
	return nil
}

func (d *TCP) Init() error {
	diag.Verbose("tcp init")
	return nil
}

func (d *TCP) Recycle() {
}

func (d *TCP) Start(s *service.Service) {
	//d.Debug("start tcp")
	s.Done()
}

func (d *TCP) Abort() {

}
