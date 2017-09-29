// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Sep-12 20:31 (EDT)
// Function: compute tests

package compute

import (
	"errors"
	"fmt"

	"argus/configure"
	"argus/diag"
	"argus/expr"
	"argus/monel"
	"argus/service"
)

type Conf struct {
	Expr string
}

type Compute struct {
	S     *service.Service
	Cf    Conf
	objs  map[string]bool
	srvc  []*service.Service
	valid bool
}

var dl = diag.Logger("compute")

func init() {
	// register with service factory
	service.Register("Compute", New)
}

func New(conf *configure.CF, s *service.Service) service.Monitor {
	p := &Compute{S: s} // vars: make(map[string]*exprParam), objs: make(map[string]bool)}
	return p
}

func (c *Compute) PreConfig(conf *configure.CF, s *service.Service) error {
	return nil
}
func (c *Compute) Config(conf *configure.CF, s *service.Service) error {

	conf.InitFromConfig(&c.Cf, "compute", "")

	// validate
	if c.Cf.Expr == "" {
		return errors.New("expr not specified")
	}

	expr, obj, err := expr.Parse(c.Cf.Expr)
	if err != nil {
		return fmt.Errorf("invalid compute expr: %v", err)
	}
	c.objs = obj
	dl.Debug("expr: %v", expr)

	// set names + labels
	uname := "COMPUTE_" + c.Cf.Expr

	s.SetNames(uname, "Compute", "Compute")

	return nil
}

func (c *Compute) Start(s *service.Service) {

	s.Debug("compute start")
	defer s.Done()

	if !c.valid {
		return
	}

	// all ready?
	for _, cs := range c.srvc {
		if !cs.IsReady() {
			s.Debug("not ready: %s", cs.Unique())
			return
		}
	}

	// CheckValue will run the computation
	s.CheckValue("0", "data")
}

func (c *Compute) Init() error {
	return nil
}

func (c *Compute) Recycle() {

	for _, s := range c.srvc {
		s.RemoveAlsoRun(c.S)
	}
}

func (c *Compute) Abort() {
}
func (c *Compute) DoneConfig() {

	// resolve depends
	// add also runs

	c.valid = true

	for obj, _ := range c.objs {
		m := monel.Find(obj)

		if m == nil {
			c.S.CFError("Cannot resolve compute dependency: '%s'", obj)
			c.valid = false
			continue
		}

		c.addDepends(m)
	}
}

func (c *Compute) addDepends(m *monel.M) {

	children := m.Me.Children()
	s := service.Find(m.Cf.Unique)

	if s != nil {
		c.srvc = append(c.srvc, s)
		s.AddAlsoRun(c.S)
		return
	}

	for _, ch := range children {
		c.addDepends(ch)
	}
}

func (c *Compute) DumpInfo() map[string]interface{} {
	return map[string]interface{}{
		"service/compute/CF/": c.Cf,
	}
}
func (c *Compute) Hostname() string {
	return ""
}
