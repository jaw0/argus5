// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Sep-12 20:31 (EDT)
// Function: self tests

package self

import (
	"errors"
	"expvar"
	"os"
	"runtime"
	"strings"

	"argus.domain/argus/configure"
	"argus.domain/argus/service"
	"github.com/jaw0/acdiag"
)

type Conf struct {
	Param string
}

/*
  some interesting params:

    services
    objects
    notifies
    monrate
    idlerate

  for a complete list, run
    argusctl self

*/
type Self struct {
	S  *service.Service
	Cf Conf
	v  expvar.Var
}

var dl = diag.Logger("self")

func init() {
	// register with service factory
	service.Register("Self", New)

	expvar.Publish("mem.alloc", expvar.Func(func() interface{} { return memstats().Alloc }))
	expvar.Publish("mem.sys", expvar.Func(func() interface{} { return memstats().Sys }))
	expvar.Publish("mem.stack", expvar.Func(func() interface{} { return memstats().StackInuse }))
	expvar.Publish("goroutines", expvar.Func(func() interface{} { return runtime.NumGoroutine() }))
	expvar.Publish("currfd", expvar.Func(currfd))
}

func New(conf *configure.CF, s *service.Service) service.Monitor {
	p := &Self{S: s}
	return p
}

func (c *Self) PreConfig(conf *configure.CF, s *service.Service) error {
	return nil
}
func (c *Self) Config(conf *configure.CF, s *service.Service) error {

	// parse spec
	c.Cf.Param = strings.TrimPrefix(conf.Name, "Self/")
	dl.Debug("self %s -> %s", conf.Name, c.Cf.Param)

	conf.InitFromConfig(&c.Cf, "self", "")

	// validate
	if c.Cf.Param == "" {
		return errors.New("param not specified")
	}

	c.v = expvar.Get(c.Cf.Param)

	if c.v == nil {
		return errors.New("invalid param")
	}

	// set names + labels
	uname := "SELF_" + c.Cf.Param
	friendly := "test of argus internal " + c.Cf.Param
	s.SetNames(uname, c.Cf.Param, friendly)

	return nil
}

func (c *Self) Start(s *service.Service) {

	s.Debug("self start")
	defer s.Done()

	s.CheckValue(c.v.String(), "data")
}

func (c *Self) Init() error {
	return nil
}
func (c *Self) Priority() bool {
	return true
}
func (c *Self) Recycle() {
}
func (c *Self) Abort() {
}
func (c *Self) DoneConfig() {
}

func (c *Self) DumpInfo() map[string]interface{} {
	return map[string]interface{}{
		"service/self/CF": c.Cf,
	}
}
func (c *Self) WebJson(md map[string]interface{}) {
}

func (c *Self) Hostname() string {
	return ""
}

func memstats() *runtime.MemStats {
	stats := new(runtime.MemStats)
	runtime.ReadMemStats(stats)
	return stats
}

// a proxy for insight into the number of currently open file descriptors
func currfd() interface{} {

	f, err := os.Open("/")
	if err != nil {
		return 0
	}
	defer f.Close()
	return f.Fd()
}
