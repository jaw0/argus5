// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Sep-09 17:00 (EDT)
// Function: monitor freeswitch

package freeswitch

import (
	"argus.domain/argus/configure"
	"argus.domain/argus/monitor/tcp"
	"argus.domain/argus/service"
)

type Conf struct {
	Pass string
	Cmd  string
}

type Freeswitch struct {
	tcp.TCP
}

func init() {
	// register with service factory
	service.Register("TCP/Freeswitch", New)
}

func New(conf *configure.CF, s *service.Service) service.Monitor {

	f := &Freeswitch{}
	// set defaults
	f.TCP.InitNew(conf, s)
	return f
}

func (d *Freeswitch) Config(conf *configure.CF, s *service.Service) error {

	cf := Conf{}
	conf.InitFromConfig(&cf, "freeswitch", "")

	// build Send
	send := "auth " + cf.Pass + "\r\n\r\n"

	if cf.Cmd != "" {
		send += "api " + cf.Cmd + "\r\n\r\n"
	}
	send += "exit\r\n\r\n"

	d.Cf.Send = send

	// set tcp config
	d.TCP.Config(conf, s)

	// determine names
	uname := "Freeswitch_" + d.Ip.Hostname()
	if cf.Cmd != "" {
		uname = uname + "_" + cf.Cmd
	}

	s.SetNames(uname, "Freeswitch", "Freeswitch on "+d.Ip.Hostname())

	return nil
}
