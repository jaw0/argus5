// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Sep-02 16:51 (EDT)
// Function: monitor elements

package monel

import (
	"argus/argus"
	"argus/configure"
	"argus/darp"
	"argus/diag"
)

var dl = diag.Logger("monel")

// Service, Group, Alias
type Moneler interface {
	Persist(map[string]interface{})
	Restore(map[string]interface{})
	Config(*configure.CF) error
	Init() error
	Name() string
	FriendlyName() string
}

type Conf struct {
	Uname       string
	Unique      string
	Sort        bool
	Overridable bool
	Note        string
	Info        string
	Details     string
	Comment     string
	Debug       bool
	Passive     bool
	Depends     string
	Siren       bool
	Nostatus    bool
	Gravity     darp.Gravity
	Countstop   bool
	Severity    argus.Status
	// notify, web, acl, graph
}

var defaults = Conf{
	Overridable: true,
	Siren:       true,
	Gravity:     darp.GRAV_DN,
	Severity:    argus.CRITICAL,
}

type Persist struct {
}

type M struct {
	Me       Moneler
	Parent   []*M
	Cf       Conf
	p        Persist
	config   *configure.CF
	Filename string
	Pathname string

	// stats, logs, notif
	// ov, anno
}

func New(me Moneler, parent *M) *M {

	m := &M{
		Me: me,
	}
	if parent != nil {
		m.Parent = []*M{parent}
	}
	// defaults
	return m
}

func (m *M) Config(conf *configure.CF) error {

	if len(m.Parent) != 0 {
		conf.SetParent(m.Parent[0].config)
	}
	conf.InitFromConfig(&m.Cf, "monel", "")

	err := m.Me.Config(conf)
	if err != nil {
		return err
	}

	m.whoami()

	m.Restore()
	m.Init()

	return nil
}

func (m *M) Init() {

	// RSN - init
	// byname{} = m

	m.Me.Init()
}

func (m *M) Update(status argus.Status) {

}

func (m *M) Debug(x ...interface{}) {

}
func (m *M) Loggit(x ...interface{}) {

}

func (m *M) Persist() {

}

func (m *M) Restore() {

	// defer recover
}

func (m *M) Unique() string {
	return m.Cf.Unique
}

func (m *M) whoami() {

	if m.Cf.Unique != "" {
		return
	}

	if m.Cf.Uname == "" {
		m.Cf.Uname = m.Me.Name()
	}

	name := ""

	if len(m.Parent) != 0 {
		name = m.Parent[0].Cf.Unique + ":"
	}

	name = name + cleanName(m.Cf.Uname)
	m.Cf.Unique = name

	// filename
	// pathname
}

func cleanName(n string) string {

	buf := make([]byte, len(n))
	j := 0

	for i := 0; i < len(n); i++ {
		switch {
		case n[i] >= 0 && n[i] < 32:
			break
		case n[i] == ' ':
			buf[j] = '_'
			j++
		default:
			buf[j] = n[i]
			j++
		}
	}

	return string(buf[:j])
}
