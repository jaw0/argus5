// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Sep-09 16:55 (EDT)
// Function: monitor tcp

package tcp

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"argus/argus"
	"argus/configure"
	"argus/diag"
	"argus/resolv"
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
	S      *service.Service
	Cf     Conf
	IpAddr *resolv.IP
	ToSend string
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

	t.S = s
	// set defaults from table
	proto := protoName(conf.Name)
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

func (t *TCP) PreConfig(conf *configure.CF, s *service.Service) error {
	return nil
}
func (t *TCP) Config(conf *configure.CF, s *service.Service) error {

	conf.InitFromConfig(&t.Cf, "tcp", "")
	dl.Debug("tcp config")

	// validate
	if t.Cf.Hostname == "" {
		return errors.New("hostname not specified")
	}
	if t.Cf.Port == 0 {
		return errors.New("port not specified")
	}

	t.IpAddr = resolv.New(t.Cf.Hostname)

	if conf.Name == "TCP/HTTP" || conf.Name == "TCP/HTTPS" {
		t.Cf.Send = t.httpSend()
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
		if strings.HasPrefix(name, "NFS") {
			uname = "TCP_" + uname
		}

	} else {
		label = "TCP"
		uname = fmt.Sprintf("TCP_%d_%s", t.Cf.Port, t.Cf.Hostname)
		friendly = fmt.Sprintf("TCP/%d on %s", t.Cf.Port, t.Cf.Hostname)
	}
	s.SetNames(uname, label, friendly)

	return nil
}

func (t *TCP) Init() error {
	//dl.Debug("tcp init: %#v", t)
	return nil
}

func (t *TCP) Hostname() string {
	return t.Cf.Hostname
}
func (t *TCP) Recycle() {
}
func (t *TCP) Abort() {
}
func (t *TCP) DoneConfig() {
}

func (t *TCP) Start(s *service.Service) {

	s.Debug("tcp start")
	defer s.Done()

	t.ToSend = t.Cf.Send
	res, fail := t.MakeRequest()
	if fail {
		return
	}

	s.CheckValue(res, "text")
}

func (t *TCP) MakeRequest() (string, bool) {

	rconn, cfail := t.Connect()
	if cfail {
		return "", true
	}

	defer rconn.Close()
	t.S.Debug("connected")

	conn := rconn
	var tconn *tls.Conn
	if t.Cf.SSL {
		t.S.Debug("enabling SSL")
		tconn = tls.Client(rconn, &tls.Config{InsecureSkipVerify: true})
		conn = tconn
	}

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

	if tconn != nil {
		//cs := tconn.ConnectionState()
		// .PeerCertificates[].TimeAfter
		// RSN - check cert?
	}

	return string(res), false
}

func (t *TCP) Send(conn net.Conn) bool {

	if t.ToSend != "" {
		t.S.Debug("send %d", len(t.ToSend))
		n, err := conn.Write([]byte(t.ToSend))
		if err != nil {
			t.S.Debug("write failed: %v", err)
			t.S.Fail("write failed")
			return true
		}

		t.S.Debug("wrote %d", n)
	}

	return false
}

func (t *TCP) Read(conn net.Conn) ([]byte, bool) {

	var res []byte
	for {
		t.S.Debug("reading...")
		buf := make([]byte, 8192)
		n, err := conn.Read(buf)
		t.S.Debug("read: %d %v", n, err)

		if n > 0 {
			res = append(res, buf...)
		}
		if len(res) > 0 && t.Cf.ReadHow == "once" {
			return res, false
		}
		if err != nil {
			if t.Cf.ReadHow == "toeof" {
				return res, false
			}
			t.S.Fail("read failed")
			return res, true
		}

		if t.Cf.ReadHow == "banner" && strings.IndexByte(string(res), '\n') != -1 {
			return res, false
		}
		if t.Cf.ReadHow == "toblank" && strings.Index(string(res), "\r\n\r\n") != -1 {
			return res, false
		}
		if t.Cf.ReadHow == "dns" && len(res) > 2 {
			rlen := int(res[0])<<8 | int(res[1])
			if len(res) >= rlen+2 {
				return res[2:], false
			}
		}
	}
}

func (t *TCP) Connect() (net.Conn, bool) {

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

	t.S.Debug("connecting to tcp %s", addrport)
	timeout := time.Duration(t.S.Cf.Timeout) * time.Second
	conn, err := net.DialTimeout("tcp", addrport, timeout)

	if err != nil {
		t.S.Fail("connect failed")
		t.S.Debug("connect failed: %v", err)
		return nil, true
	}

	// set timeout
	conn.SetDeadline(time.Now().Add(timeout))

	return conn, false
}

func (t *TCP) httpSend() string {

	send := "GET / HTTP/1.1\r\n" +
		"Host: " + t.Cf.Hostname + "\r\n" +
		"Connection: Close\r\n" +
		"User-Agent: Argus\r\n" +
		"\r\n"

	return send
}

func protoName(name string) string {

	if strings.HasPrefix(name, "TCP/") {
		return name[4:]
	}

	return ""
}

func (t *TCP) DumpInfo() map[string]interface{} {
	return map[string]interface{}{
		"service/tcp/CF/": t.Cf,
	}
}
