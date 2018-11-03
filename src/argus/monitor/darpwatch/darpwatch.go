// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Oct-04 19:12 (EDT)
// Function: is darp connection up?

package darpwatch

import (
	"errors"
	"strings"

	"argus/configure"
	"argus/darp"
	"github.com/jaw0/acgo/diag"
	"argus/service"
)

type Conf struct {
	Watch string
	Mode  string
}

type Watch struct {
	S  *service.Service
	Cf Conf
}

var dl = diag.Logger("darpwatch")

func init() {
	// register with service factory
	service.Register("DARP/Watch", New)
}

func New(conf *configure.CF, s *service.Service) service.Monitor {
	p := &Watch{S: s}
	return p
}

func (w *Watch) PreConfig(conf *configure.CF, s *service.Service) error {
	return nil
}
func (w *Watch) Config(conf *configure.CF, s *service.Service) error {

	name := conf.Name
	name = strings.TrimPrefix(name, "DARP/Watch")
	if name != "" {
		w.Cf.Watch = name[1:]
	}

	w.Cf.Mode = "slave"

	conf.InitFromConfig(&w.Cf, "darp/watch", "")

	// valid darp?
	if !darp.IsValid(w.Cf.Watch) {
		return errors.New("invalid darp name")
	}

	// set names + labels
	uname := "DARP_WATCH_" + w.Cf.Mode + "_" + w.Cf.Watch

	s.SetNames(uname, w.Cf.Watch, "DARP "+w.Cf.Watch)

	return nil
}

func (w *Watch) Start(s *service.Service) {

	s.Debug("darp/watch start")
	defer s.Done()

	if darp.IsUp(w.Cf.Watch) {
		s.Pass()
	} else {
		s.Fail("connection is down")
	}
	return
}

func (w *Watch) Init() error {
	return nil
}
func (c *Watch) Priority() bool {
	return true
}

func (w *Watch) Recycle() {
}

func (w *Watch) Abort() {
}
func (w *Watch) DoneConfig() {
}

func (w *Watch) DumpInfo() map[string]interface{} {
	return map[string]interface{}{
		"service/darpwatch/CF": w.Cf,
	}
}
func (w *Watch) WebJson(md map[string]interface{}) {
	md["watch"] = w.Cf.Watch
}

func (w *Watch) Hostname() string {
	return ""
}
