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

func New(conf *configure.CF, s *service.Service) service.Monitor {

	a := &Asterisk{}
	// set defaults
	a.TCP.InitNew(conf, s)
	return a
}

func (d *Asterisk) Config(conf *configure.CF, s *service.Service) error {

	acf := Conf{}
	conf.InitFromConfig(&acf, "asterisk", "")

	// build Send
	send := "Action: Login\r\n" +
		"Username: " + acf.User + "\r\n" +
		"Secret: " + acf.Pass + "\r\n" +
		"\r\n"
	if acf.Cmd != "" {
		send = send + "Action: Command\r\nCommand: " + acf.Cmd + "\r\n\r\n"
	}
	send = send + "Action: Logoff\r\n\r\n"

	d.Cf.Send = send

	// set tcp config
	d.TCP.Config(conf, s)

	// determine names
	uname := "Asterisk_" + d.Cf.Hostname
	if acf.Cmd != "" {
		uname = uname + "_" + acf.Cmd
	}

	s.SetNames(uname, "Asterisk", "Asterisk on "+d.Cf.Hostname)

	return nil
}
