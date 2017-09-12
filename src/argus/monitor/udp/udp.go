// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Sep-09 16:55 (EDT)
// Function: monitor udp

package udp

import (
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	//"argus/argus"
	"argus/configure"
	"argus/diag"
	"argus/resolv"
	"argus/service"
)

type Conf struct {
	Hostname             string
	Port                 int
	Send                 string
	SendHex              string
	Verify_Response_IP   bool
	Verify_Response_Port bool
}

type UDP struct {
	S       *service.Service
	Cf      Conf
	IpAddr  *resolv.IP
	ToSend  string
	dstaddr *net.UDPAddr
}

var dl = diag.Logger("udp")

func init() {
	// register with service factory
	service.Register("UDP", New)
}

func New(conf *configure.CF, s *service.Service) service.Monitor {
	t := &UDP{}
	t.InitNew(conf, s)
	return t
}

func (t *UDP) InitNew(conf *configure.CF, s *service.Service) {

	t.S = s
	// set defaults from table
	proto := protoName(conf.Name)
	pdat, ok := udpProtoTab[proto]
	if !ok {
		return
	}

	t.Cf.Port = pdat.Port
	t.Cf.Send = pdat.Send
	t.Cf.Verify_Response_IP = true
	t.Cf.Verify_Response_Port = true
	s.Cf.Unpack = pdat.Unpack
	s.Cf.Scale = pdat.Scale
}

func (t *UDP) Config(conf *configure.CF, s *service.Service) error {

	conf.InitFromConfig(&t.Cf, "udp", "")
	dl.Debug("udp config")

	// validate
	if t.Cf.Hostname == "" {
		return errors.New("hostname not specified")
	}
	if t.Cf.Port == 0 {
		return errors.New("port not specified")
	}

	t.IpAddr = resolv.New(t.Cf.Hostname)

	if t.Cf.SendHex != "" {
		t.Cf.Send = hxd(t.Cf.SendHex)
	}

	if t.Cf.Send == "" {
		return errors.New("send not specified")
	}

	// set names + labels
	name := protoName(conf.Name)
	friendly := ""
	uname := ""
	label := ""

	if name != "" {
		label = name
		uname = name + "_" + t.Cf.Hostname
		friendly = name + " on " + t.Cf.Hostname

	} else {
		label = "UDP"
		uname = fmt.Sprintf("UDP_%d_%s", t.Cf.Port, t.Cf.Hostname)
		friendly = fmt.Sprintf("UDP/%d on %s", t.Cf.Port, t.Cf.Hostname)
	}
	s.SetNames(uname, label, friendly)

	return nil
}

func (t *UDP) Init() error {
	return nil
}

func (t *UDP) Recycle() {
}
func (t *UDP) Abort() {
}

func (t *UDP) Start(s *service.Service) {

	s.Debug("udp start")
	defer s.Done()

	t.ToSend = t.Cf.Send
	res, fail := t.MakeRequest()
	if fail {
		return
	}

	s.CheckValue(res, "data")
}

func (t *UDP) MakeRequest() (string, bool) {

	conn, cfail := t.Connect()
	if cfail {
		return "", true
	}

	defer conn.Close()

	// send
	sfail := t.Send(conn)
	if sfail {
		return "", true
	}

	// read
	res, wfail := t.Read(conn)
	if wfail {
		return "", true
	}

	return string(res), false
}

func (t *UDP) Send(conn *net.UDPConn) bool {

	t.S.Debug("send %d", len(t.ToSend))
	n, err := conn.Write([]byte(t.ToSend))
	if err != nil {
		t.S.Debug("write failed: %v", err)
		t.S.Fail("write failed")
		return true
	}

	t.S.Debug("wrote %d", n)

	return false
}

func (t *UDP) Read(conn *net.UDPConn) ([]byte, bool) {

	t.S.Debug("reading...")
	buf := make([]byte, 8192)
	n, addr, err := conn.ReadFromUDP(buf)
	t.S.Debug("read: %d %v", n, err)

	// check response
	fail := t.checkResponse(addr)
	if fail {
		return nil, true
	}

	return buf, false
}

func (t *UDP) Connect() (*net.UDPConn, bool) {

	addr, fail := t.IpAddr.AddrWB()
	if addr == "" {
		t.S.Debug("hostname still resolving")
		return nil, true
	}
	if fail {
		t.S.Fail("cannot resolve hostname")
		return nil, true
	}

	addrport := fmt.Sprintf("%s:%d", addr, t.Cf.Port)

	t.S.Debug("connecting to udp %s", addrport)
	uaddr, err := net.ResolveUDPAddr("udp", addrport)
	if err != nil {
		dl.Problem("error? %v", err)
		t.S.Fail("error")
	}
	t.dstaddr = uaddr
	conn, err := net.DialUDP("udp", nil, uaddr)

	if err != nil {
		t.S.Fail("connect failed")
		t.S.Debug("connect failed: %v", err)
		return nil, true
	}

	// set timeout
	timeout := time.Duration(t.S.Cf.Timeout) * time.Second
	conn.SetDeadline(time.Now().Add(timeout))

	return conn, false
}

func (t *UDP) checkResponse(addr *net.UDPAddr) bool {

	if addr == nil {
		return true
	}

	ok := true

	if t.Cf.Verify_Response_IP && !t.dstaddr.IP.Equal(addr.IP) {
		ok = false
	}
	if t.Cf.Verify_Response_Port && addr.Port != t.dstaddr.Port {
		ok = false
	}

	if !ok {
		t.S.Debug("unexpected response from %s", addr.String())
		t.S.Fail("recv check failed")
		return true
	}

	return false

}

func protoName(name string) string {

	if strings.HasPrefix(name, "UDP/") {
		return name[4:]
	}

	return ""
}
