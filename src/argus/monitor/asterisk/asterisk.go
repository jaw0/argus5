// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Sep-09 17:00 (EDT)
// Function: monitor asterisk

package asterisk

import (
	"argus/configure"
	"argus/monitor/tcp"
	"argus/service"
)

type Conf struct {
	User string
	Pass string
	Cmd  string
}

type Asterisk struct {
	tcp.TCP
}

func init() {
	// register with service factory
	service.Register("TCP/Asterisk", New)
}

func New() service.Monitor {

	a := &Asterisk{}
	a.TCP.InitNew()
	return a
}

func (d *Asterisk) Config(conf *configure.CF, s *service.Service) error {

	acf := Conf{}
	conf.InitFromConfig(&acf, "asterisk", "")

	// set tcp defaults
	d.TCP.Config(conf, s)

	return nil
}
