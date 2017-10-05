// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Sep-12 20:31 (EDT)
// Function: ping tests

package ping

import (
	"bufio"
	"context"
	"errors"
	"expvar"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"
	"time"

	"argus/config"
	"argus/configure"
	"argus/diag"
	"argus/resolv"
	"argus/service"
)

type Conf struct {
	Hostname string
}

type Ping struct {
	S      *service.Service
	Cf     Conf
	IpAddr *resolv.IP
}

type pingWork struct {
	ipaddr string
	s      *service.Service
}

const (
	WORKERS = 10
	MAXPING = 250
	TIMEOUT = 60 * time.Second
)

const QUEUESIZE = 16

var dl = diag.Logger("ping")
var pingChan = make(chan pingWork, QUEUESIZE)
var PingQueue = expvar.NewInt("pingqueue")
var PingIdle = expvar.NewInt("pingidle")
var justOne sync.Mutex // only one worker collects work at a time

var lock sync.Mutex
var nIdle = 0

func init() {
	// register with service factory
	service.Register("Ping", New)
}

func Init() {

	cf := config.Cf()
	nwork := cf.Ping_maxrun
	if nwork < 1 {
		nwork = WORKERS
	}

	// start workers
	for i := 0; i < nwork; i++ {
		go worker()
	}
}

func New(conf *configure.CF, s *service.Service) service.Monitor {
	p := &Ping{S: s}
	return p
}

func (p *Ping) PreConfig(conf *configure.CF, s *service.Service) error {
	return nil
}
func (p *Ping) Config(conf *configure.CF, s *service.Service) error {

	conf.InitFromConfig(&p.Cf, "ping", "")

	// validate
	if p.Cf.Hostname == "" {
		return errors.New("hostname not specified")
	}

	p.IpAddr = resolv.New(p.Cf.Hostname)

	// set names + labels
	uname := "PING_" + p.Cf.Hostname

	s.SetNames(uname, "Ping", "Ping")

	return nil
}

func (p *Ping) Init() error {
	return nil
}

func (p *Ping) Hostname() string {
	return p.Cf.Hostname
}
func (p *Ping) Recycle() {
}
func (p *Ping) Abort() {
	p.S.Debug("aborting")
	p.S.Done()
}

func (p *Ping) DoneConfig() {
}

func (p *Ping) Start(s *service.Service) {

	s.Debug("ping start")

	addr, _, fail := p.IpAddr.Addr()
	s.Debug("addr: %s, %v", addr, fail)
	if fail {
		s.FailNow("cannot resolve hostname")
		s.Done()
		return
	}
	if addr == "" {
		s.Debug("hostname still resolving")
		s.Done()
		return
	}

	PingQueue.Set(int64(len(pingChan)))
	PingIdle.Set(int64(numIdle()))
	// RSN - more workers?

	// send it off to worker
	pingChan <- pingWork{addr, s}
}

/*
  queue full(ish):
    idle workers -> problem? overloaded?
    no idle workers && idle-enough -> start more
*/

func worker() {

	for {
		amIdle(true)

		// only one worker gathers work at a time
		justOne.Lock()
		// lock is released in ping(), below

		amIdle(false)
		select {
		case pw := <-pingChan:
			ping(pw)
		}
	}
}

func ping(pw pingWork) {

	underway := make(map[string][]*service.Service)
	nping := 0

	// start fping
	// get more work
	// get results

	dl.Debug("starting fping")
	ctx, cancel := context.WithTimeout(context.Background(), TIMEOUT)
	defer cancel()

	cmd := exec.CommandContext(ctx, "fping", "-r", "3", "-t", "500", "-a", "-e")

	stdin, err := cmd.StdinPipe()
	if err != nil {
		dl.Problem("cannot run fping: %v", err)
		return
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		dl.Problem("cannot run fping: %v", err)
		return
	}

	err = cmd.Start()
	if err != nil {
		dl.Problem("cannot run fping: %v", err)
		return
	}

	addPing(pw, stdin, underway)
	nping++

	nping += addMore(stdin, underway)
	justOne.Unlock() // allow another worker to proceed
	stdin.Close()

	bfd := bufio.NewReader(stdout)

	for {
		line, _, err := bfd.ReadLine()
		dl.Debug("fping> %s, %v", string(line), err)
		if err != nil {
			dl.Debug("err: %v", err)
			break
		}

		processResult(underway, string(line))
	}

	dl.Debug("waiting....")
	err = cmd.Wait()
	dl.Debug("fping finished %v", err)

	// fail anything not returned by fping
	for addr, ss := range underway {
		for _, s := range ss {
			dl.Debug("no response: %s", addr)
			s.Fail("no response")
			s.Done()
		}
	}
}

func addPing(pw pingWork, stdin io.Writer, uw map[string][]*service.Service) {

	_, ok := uw[pw.ipaddr]
	if !ok {
		dl.Debug("+ %s", pw.ipaddr)
		io.WriteString(stdin, pw.ipaddr+"\n")
	}
	uw[pw.ipaddr] = append(uw[pw.ipaddr], pw.s)
}

func addMore(stdin io.Writer, uw map[string][]*service.Service) int {

	// gather any other pings todo
	nping := 0
	for {
		select {
		case pw := <-pingChan:
			addPing(pw, stdin, uw)
			nping++
			if nping >= MAXPING {
				return nping
			}
		case <-time.After(time.Second):
			return nping
		}
	}
}

func processResult(uw map[string][]*service.Service, line string) {

	space := strings.IndexAny(line, " \t")
	paren := strings.IndexByte(line, '(')
	if space == -1 || paren == -1 {
		dl.Debug("cannot parse '%s' %X", line, line)
		return
	}
	addr := line[:space]

	var elapsed float32
	fmt.Sscan(line[paren+1:], &elapsed)
	res := fmt.Sprintf("%f", elapsed)

	dl.Debug("%s -> %f", addr, elapsed)

	ss := uw[addr]
	for _, s := range ss {
		s.CheckValue(res, "string")
		s.Done()
	}
	delete(uw, addr)
}

func (p *Ping) DumpInfo() map[string]interface{} {
	return map[string]interface{}{
		"service/ping/CF/": p.Cf,
	}
}
func (p *Ping) WebJson(md map[string]interface{}) {
}

func amIdle(y bool) {

	lock.Lock()
	defer lock.Unlock()
	if y {
		nIdle++
	} else {
		nIdle--
	}
	PingIdle.Set(int64(nIdle))
}
func numIdle() int {
	lock.Lock()
	defer lock.Unlock()
	return nIdle
}
