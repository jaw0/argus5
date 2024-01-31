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

	"argus.domain/argus/configure"
	"argus.domain/argus/resolv"
	"argus.domain/argus/service"
	"github.com/jaw0/acdiag"
)

type Packeter interface {
	Packet(net.Conn) (string, error)
}

type Conf struct {
	Port                 int
	Send                 string
	SendHex              string
	Verify_Response_IP   bool
	Verify_Response_Port bool
}

type UDP struct {
	S       *service.Service
	Cf      Conf
	Ip      *resolv.IP
	Addr    string // for debugging
	ToSend  string
	FSend   Packeter
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

func (t *UDP) PreConfig(conf *configure.CF, s *service.Service) error {
	return nil
}

func (t *UDP) Config(conf *configure.CF, s *service.Service) error {

	dl.Debug("udp config")
	conf.InitFromConfig(&t.Cf, "udp", "")

	ip, err := resolv.Config(conf)
	if err != nil {
		return err
	}
	t.Ip = ip

	// validate
	if t.Cf.Port == 0 {
		return errors.New("port not specified")
	}

	if t.Cf.SendHex != "" {
		t.Cf.Send = hxd(t.Cf.SendHex)
	}

	if t.Cf.Send == "" && t.FSend == nil {
		return errors.New("send not specified")
	}

	// set names + labels
	host := t.Ip.Hostname()
	name := protoName(conf.Name)
	friendly := ""
	uname := ""
	label := ""

	if name != "" {
		label = name
		uname = name + "_" + host
		friendly = name + " on " + host

	} else {
		label = "UDP"
		uname = fmt.Sprintf("UDP_%d_%s", t.Cf.Port, host)
		friendly = fmt.Sprintf("UDP/%d on %s", t.Cf.Port, host)
	}
	s.SetNames(uname, label, friendly)

	return nil
}

func (t *UDP) Init() error {
	return nil
}
func (t *UDP) Priority() bool {
	return false
}
func (t *UDP) Hostname() string {
	return t.Ip.Hostname()
}
func (t *UDP) Recycle() {
}
func (t *UDP) Abort() {
}
func (t *UDP) DoneConfig() {
}

func (t *UDP) Start(s *service.Service) {

	s.Debug("udp start")
	defer s.Done()

	t.ToSend = t.Cf.Send
	res, fail := t.MakeRequest()
	if fail {
		t.Ip.TryAnother()
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

	if t.FSend != nil {
		p, err := t.FSend.Packet(conn)
		if err != nil {
			t.S.Debug("build packet failed: %v", err)
			t.S.Fail("send failed")
			return true
		}
		t.ToSend = p
	}

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
	t.S.Debug("read: %d from %v: %v [%X]", n, addr, err, buf[:n])

	if err != nil || addr == nil {
		t.S.Debug("read failed: %v", err)
		t.S.Fail("read failed")
		return nil, true
	}

	// check response
	fail := t.checkResponse(addr)
	if fail {
		return nil, true
	}

	return buf[:n], false
}

func (t *UDP) Connect() (*net.UDPConn, bool) {

	addr, fail := t.Ip.AddrWB()

	if fail {
		t.S.FailNow("cannot resolve hostname")
		return nil, true
	}

	if addr == "" {
		t.S.Debug("hostname still resolving")
		return nil, true
	}

	t.Ip.WillNeedIn(t.S.Cf.Frequency)
	addrport := fmt.Sprintf("%s:%d", addr, t.Cf.Port)
	t.Addr = addrport

	t.S.Debug("connecting to udp %s", addrport)
	uaddr, err := net.ResolveUDPAddr("udp", addrport)
	if err != nil {
		dl.Problem("error? %v", err)
		t.S.Fail("error")
		return nil, true
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

func (u *UDP) DumpInfo() map[string]interface{} {
	return map[string]interface{}{
		"service/ip/CF":    &u.Ip.Cf,
		"service/ip/FQDN":  &u.Ip.Fqdn,
		"service/udp/CF":   &u.Cf,
		"service/udp/addr": u.Addr,
	}
}
func (u *UDP) WebJson(md map[string]interface{}) {
}
