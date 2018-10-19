// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Oct-16 21:54 (EDT)
// Function: monitor remote agent

package agent

import (
	"crypto/tls"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"argus/api/client"
	"argus/configure"
	"argus/diag"
	"argus/resolv"
	"argus/sec"
	"argus/service"
)

type TConf struct {
	Command string
	Pluck   string
	JPath   string
	Column  string
}

type Command struct {
	Name string
	MCf  map[string]*TConf // [os]
}

type Conf struct {
	Param      string
	Arg        string
	Agent_Port int
}

type Agent struct {
	S    *service.Service
	Cf   Conf
	Ip   *resolv.IP
	Addr string // for debugging
	cmd  *Command
	os   string
}

var dl = diag.Logger("agent")
var commands = make(map[string]*Command)

func init() {
	// register with service factory
	service.Register("Agent", New)
}

func New(conf *configure.CF, s *service.Service) service.Monitor {
	a := &Agent{S: s}
	return a
}

func (a *Agent) PreConfig(conf *configure.CF, s *service.Service) error {
	return nil
}
func (a *Agent) Config(conf *configure.CF, s *service.Service) error {

	// Service Agent/param[/arg]
	f := strings.Split(conf.Name, "/")
	if len(f) > 1 {
		a.Cf.Param = strings.ToLower(f[1])
	}
	if len(f) > 2 {
		a.Cf.Arg = f[2]
	}

	conf.InitFromConfig(&a.Cf, "agent", "")

	ip, err := resolv.Config(conf)
	if err != nil {
		return err
	}
	a.Ip = ip

	// validate
	if a.Cf.Agent_Port == 0 {
		return errors.New("agent_port not specified")
	}
	if a.Cf.Param == "" {
		return errors.New("param not specified")
	}

	if a.Cf.Param != "self" {
		a.cmd = commands[a.Cf.Param]
		if a.cmd == nil {
			return fmt.Errorf("unknown Agent command '%s'", a.Cf.Param)
		}
	}

	uname := "AGENT_" + a.Cf.Param
	if a.Cf.Arg != "" {
		uname += "_" + a.Cf.Arg
	}
	label := a.Cf.Param
	if label == "self" {
		label = a.Cf.Arg
	}

	friendly := fmt.Sprintf("Agent %s on %s", a.Cf.Param, a.Ip.Hostname())

	s.SetNames(uname, label, friendly)
	return nil
}

func (a *Agent) Start(s *service.Service) {

	defer s.Done()
	timeout := time.Duration(a.S.Cf.Timeout) * time.Second
	// connect
	conn, fail := a.Connect()
	if fail {
		return
	}
	defer conn.Close()

	if a.os == "" {
		// determine OS
		res, fail := a.docmd(conn, "getos", nil, timeout)
		if fail {
			return
		}
		a.os = res["os"]
	}

	// send command
	args, err := a.buildCommand()
	if err != nil {
		s.Fail(err.Error())
		return
	}

	res, fail := a.docmd(conn, "agent", args, timeout)
	if fail {
		return
	}

	os := res["os"]
	if os == "" {
		a.S.Fail("protocol botched")
		return
	}

	if os != a.os {
		a.os = os
		a.S.Debug("OS change detected -> %s", a.os)
		return
	}

	if res["fail"] != "" {
		a.S.Fail("failed")
		return
	}

	s.Debug("got result: %s", res["result"])
	s.CheckValue(res["result"], "data")
}

func (a *Agent) buildCommand() (map[string]string, error) {

	if a.Cf.Param == "self" {
		return map[string]string{
			"self": a.Cf.Arg,
		}, nil
	}

	if a.cmd == nil {
		return nil, errors.New("agent not properly configured")
	}

	m := a.cmd.MCf[a.os]
	if m == nil {
		m = a.cmd.MCf[""]
	}

	if m == nil {
		return nil, fmt.Errorf("agent not configured for OS '%s'", a.os)
	}

	// expand args
	args := strings.Fields(a.Cf.Arg)
	cmd := os.Expand(m.Command, func(x string) string {

		switch x {
		case "*", "ARGS":
			return a.Cf.Arg
		case "1", "2", "3", "4", "5", "6", "7", "8", "9":
			n, _ := strconv.Atoi(x)
			if n < len(args) {
				return args[n]
			}
		}
		return ""
	})

	return map[string]string{
		"command": cmd,
		"pluck":   m.Pluck,
		"jpath":   m.JPath,
		"column":  m.Column,
		"os":      a.os,
	}, nil
}

func (a *Agent) docmd(conn *client.Conn, cmd string, args map[string]string, timeout time.Duration) (map[string]string, bool) {

	resp, err := conn.GetMap(cmd, args, timeout)
	if err != nil {
		a.S.Debug("error: %v", err)
		a.S.Fail("command failed")
		return nil, true
	}
	return resp.Map(), false
}

func (a *Agent) Connect() (*client.Conn, bool) {

	addr, fail := a.Ip.AddrWB()
	if fail {
		a.S.FailNow("cannot resolve hostname")
		return nil, true
	}
	if addr == "" {
		a.S.Debug("hostname still resolving")
		return nil, true
	}

	a.Ip.WillNeedIn(a.S.Cf.Frequency)
	addrport := fmt.Sprintf("%s:%d", addr, a.Cf.Agent_Port)
	a.Addr = addrport

	a.S.Debug("connecting to tcp %s", addrport)

	timeout := time.Duration(a.S.Cf.Timeout) * time.Second
	conn, err := client.NewTLS(addrport, timeout, &tls.Config{
		Certificates:       []tls.Certificate{*sec.Cert},
		RootCAs:            sec.Root,
		InsecureSkipVerify: true, // other side will verify us, not vice-versa
	})

	if err != nil {
		a.S.Fail("connect failed")
		a.S.Debug("connect failed: %v", err)
		a.Ip.TryAnother()
		return nil, true
	}

	return conn, false
}

func (a *Agent) Hostname() string {
	return a.Ip.Hostname()
}
func (a *Agent) Init() error {
	return nil
}

func (a *Agent) Recycle() {
}
func (a *Agent) Abort() {
}
func (a *Agent) DoneConfig() {
}
func (a *Agent) DumpInfo() map[string]interface{} {
	return map[string]interface{}{
		"service/ip/CF":      &a.Ip.Cf,
		"service/ip/FQDN":    &a.Ip.Fqdn,
		"service/agent/CF":   a.Cf,
		"service/agent/addr": a.Addr,
	}
}
func (a *Agent) WebJson(md map[string]interface{}) {

}

// ################################################################

func NewAgent(conf *configure.CF) error {

	cf := &TConf{}
	conf.InitFromConfig(cf, "agent", "")

	name := strings.ToLower(conf.Name)

	cmd := commands[name]
	if cmd == nil {
		cmd = &Command{
			Name: name,
			MCf:  make(map[string]*TConf),
		}
		commands[name] = cmd
	}

	oslist := conf.Extra
	if len(oslist) == 0 {
		oslist = []string{""}
	}

	for _, os := range oslist {
		if cmd.MCf[os] != nil {
			return fmt.Errorf("redefinition of agent %s %s", conf.Name, os)
		}

		cmd.MCf[os] = cf
	}

	return nil
}
