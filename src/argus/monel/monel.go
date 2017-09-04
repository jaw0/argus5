// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Sep-02 16:51 (EDT)
// Function: monitor elements

package monel

import (
	"fmt"

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
	Sendnotify  bool
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
	Status          argus.Status
	OvStatus        argus.Status
	Override        *argus.Override
	AncInOv         bool
	Alarm           bool
	OvStatusSummary [argus.MAXSTATUS + 1]int
	Interesting     bool
	TransTime       int64
	SirenTime       int64
}

type M struct {
	Me       Moneler
	Parent   []*M
	Children []*M
	Cf       Conf
	p        Persist
	config   *configure.CF
	Filename string
	DirName  string

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

	m.Cf = defaults

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

	if len(m.Parent) != 0 {
		m.Parent[0].AddChild(m)
	}

	m.Init()

	return nil
}

func (m *M) Init() {

	// RSN - init
	// byname{} = m

	m.Me.Init()
}

func (m *M) DoneConfiguring() {
	// recalc:
	// ovstatus
	// ovstatussummary
}

func (m *M) AddChild(n *M) {
	m.Children = append(m.Children, n)
}
func (m *M) AddParent(n *M) {
	m.Parent = append(m.Parent, n)
}

func (m *M) Update(status argus.Status) {

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
	m.Filename = argus.Encode(name)
	m.DirName = argus.HashedDirectory(name)
}

func (m *M) Pathname(pre, suf string) string {

	return m.DirName + "/" + pre + m.Filename + suf

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

func (m *M) Debug(text string, args ...interface{}) {

	if m.Cf.Debug {
		msg := fmt.Sprintf(text, args...)
		diag.Verbose("%s %s", m.Cf.Unique, msg)
	}
}

func (m *M) Loggit(text string, args ...interface{}) {

	msg := fmt.Sprintf(text, args...)
	diag.Verbose("%s %s", m.Cf.Unique, msg)

	// RSN append m.log
}
