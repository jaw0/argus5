// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Sep-09 16:55 (EDT)
// Function: monitor tcp

package tcp

import (
	"argus/argus"
	"argus/configure"
	"argus/diag"
	"argus/service"
)

type Conf struct {
	Hostname string
	Port     int
	Send     string
	ReadHow  string
	SSL      bool
}

type TCP struct {
	Cf          Conf
	MaybeExpect string
}

var dl = diag.Logger("tcp")

func init() {
	// register with service factory
	service.Register("TCP", New)
}

func New(conf *configure.CF, s *service.Service) service.Monitor {
	t := &TCP{}
	t.InitNew(conf, s)
	return t
}

func (t *TCP) InitNew(conf *configure.CF, s *service.Service) {

	// set defaults from table
	name := conf.Name
	if len(name) < 4 {
		return
	}

	proto := name[4:]
	pdat, ok := tcpProtoTab[proto]
	if !ok {
		return
	}

	t.Cf.Port = pdat.Port
	t.Cf.Send = pdat.Send
	t.Cf.ReadHow = pdat.ReadHow
	t.Cf.SSL = pdat.SSL
	s.Cf.Expect[int(argus.UNKNOWN)] = pdat.Expect
}

func (t *TCP) Config(conf *configure.CF, s *service.Service) error {

	conf.InitFromConfig(&t.Cf, "tcp", "")
	// s.SetNames( )
	dl.Debug("tcp config")

	// validate
	// set names

	return nil
}

func (t *TCP) Init() error {
	//dl.Debug("tcp init: %#v", t)
	return nil
}

func (t *TCP) Recycle() {
}

func (t *TCP) Start(s *service.Service) {
	//d.Debug("start tcp")
	s.Done()
}

func (t *TCP) Abort() {

}
