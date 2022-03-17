// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Sep-18 20:00 (EDT)
// Function: test sip

package sip

import (
	"errors"
	"fmt"
	"math/rand"
	"net"
	"strings"

	"argus.domain/argus/argus"
	"argus.domain/argus/configure"
	"github.com/jaw0/acgo/diag"
	"argus.domain/argus/monitor/tcp"
	"argus.domain/argus/monitor/udp"
	"argus.domain/argus/service"
)

type Conf struct {
	To        string
	From      string
	UserAgent string
}

type TCPSIP struct {
	tcp.TCP
	SCf Conf
}
type UDPSIP struct {
	udp.UDP
	SCf Conf
}

var dl = diag.Logger("tcp")

func init() {
	// register with service factory
	service.Register("TCP/SIP", NewTCP)
	service.Register("UDP/SIP", NewUDP)
}

func NewTCP(conf *configure.CF, s *service.Service) service.Monitor {

	d := &TCPSIP{}
	// set defaults
	d.TCP.Cf.Port = 5060
	d.SCf.UserAgent = "Argus/" + argus.Version
	d.TCP.InitNew(conf, s)
	d.TCP.Cf.ReadHow = "toblank"
	d.TCP.FSend = &d.SCf
	return d
}

func NewUDP(conf *configure.CF, s *service.Service) service.Monitor {

	d := &UDPSIP{}
	// set defaults
	d.UDP.Cf.Port = 5060
	d.SCf.UserAgent = "Argus/" + argus.Version
	d.UDP.InitNew(conf, s)
	d.UDP.FSend = &d.SCf
	return d
}

func (d *TCPSIP) Config(conf *configure.CF, s *service.Service) error {
	conf.InitFromConfig(&d.SCf, "SIP", "")

	err := d.TCP.Config(conf, s)
	if err != nil {
		return err
	}
	return nil
}
func (d *UDPSIP) Config(conf *configure.CF, s *service.Service) error {
	conf.InitFromConfig(&d.SCf, "SIP", "")

	err := d.UDP.Config(conf, s)
	if err != nil {
		return err
	}
	return nil
}

// ################################################################

func (cf *Conf) Packet(conn net.Conn) (string, error) {

	laddr := conn.LocalAddr()
	proto := strings.ToUpper(laddr.Network())
	addr := laddr.String()

	var host string

	if addr == "" {
		return "", errors.New("no addr")
	}

	if addr[0] == '[' {
		end := strings.IndexByte(addr, ']')
		if end == -1 {
			return "", errors.New("no addr")
		}
		host = addr[1:end]
	} else {
		end := strings.IndexByte(addr, ':')
		if end == -1 {
			return "", errors.New("no addr")
		}
		host = addr[:end]
	}

	to := cf.To
	from := cf.From

	if to == "" {
		to = "sip:" + conn.RemoteAddr().String()
	}
	if from == "" {
		from = "sip:argus@" + addr
	}

	unique := fmt.Sprintf("%x", rand.Int31())
	tag := unique
	branch := "z9hG4bK" + unique
	callid := unique

	sip := "OPTIONS " + to + " SIP/2.0\r\n" +
		"Via: SIP/2.0/" + proto + " " + addr + ";branch=" + branch + "\r\n" +
		"Max-Forwards: 70\r\n" +
		"From: <" + from + ">;tag=" + tag + "\r\n" +
		"To: <" + to + ">\r\n" +
		"Call-ID: " + callid + "@" + host + "\r\n" +
		"Cseq: 1 OPTIONS\r\n" +
		"User-Agent: " + cf.UserAgent + "\r\n" +
		"Content-Length: 0\r\n" +
		"Accept: */*\r\n\r\n"

	return sip, nil
}
