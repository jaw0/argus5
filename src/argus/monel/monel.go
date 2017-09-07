// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Sep-02 16:51 (EDT)
// Function: monitor elements

package monel

import (
	"fmt"
	"sync"

	"argus/argus"
	"argus/configure"
	"argus/diag"
)

var dl = diag.Logger("monel")

var lock sync.RWMutex
var byname = make(map[string]*M)

// Service, Group, Alias
type Moneler interface {
	Persist(map[string]interface{})
	Restore(map[string]interface{})
	Config(*configure.CF) error
	Init() error
	DoneConfig()
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
	Gravity     argus.Gravity
	Countstop   bool
	Sendnotify  bool
	// notify, web, acl, graph
}

var defaults = Conf{
	Overridable: true,
	Siren:       true,
	Gravity:     argus.GRAV_DN,
}

type Persist struct {
	Status          argus.Status
	OvStatus        argus.Status
	Override        *argus.Override
	Result          string // not current, only as of the most recent transition
	Reason          string
	AncInOv         bool
	Alarm           bool
	OvStatusSummary [argus.MAXSTATUS + 1]int
	Interesting     bool
	TransTime       int64
	SirenTime       int64
	Culprit         string
	Stats           Stats
	Log             []*Log
}

type M struct {
	Me       Moneler
	Lock     sync.RWMutex
	Parent   []*M
	Children []*M
	Cf       Conf
	P        Persist
	ConfCF   *configure.CF
	Filename string
	DirName  string
	Depends  []string

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

	m.ConfCF = conf
	conf.InitFromConfig(&m.Cf, "monel", "")

	err := m.Me.Config(conf)
	if err != nil {
		return err
	}

	m.whoami()

	if Find(m.Cf.Unique) != nil {
		return fmt.Errorf("Duplicate object '%s' (%s)", m.Cf.Unique, conf.Type)
	}

	m.Restore()

	m.Init()

	return nil
}

func (m *M) Init() {

	// RSN - init...

	if len(m.Parent) != 0 {
		m.Parent[0].AddChild(m)
	}

	lock.Lock()
	byname[m.Cf.Unique] = m
	lock.Unlock()

	m.Me.Init()
}

func (m *M) Status() (argus.Status, argus.Status) {
	m.Lock.RLock()
	defer m.Lock.RUnlock()
	return m.P.Status, m.P.OvStatus
}

func (m *M) DoneConfig() {

	for _, child := range m.Children {
		child.DoneConfig()
	}

	m.Me.DoneConfig()
	m.ConfCF.DrainCache()
	m.sortChildren()

	// resolve_depends
	// determine interestingness

	// recalc:
	// ovstatus
	// ovstatussummary
	// sort children

	m.determineStatus()

}

func (m *M) sortChildren() {

	if !m.Cf.Sort {
		return
	}

	// RSN - ...
}

func (m *M) AddChild(n *M) {
	m.Lock.Lock()
	defer m.Lock.Unlock()
	m.Children = append(m.Children, n)
}
func (m *M) AddParent(n *M) {
	m.Lock.Lock()
	defer m.Lock.Unlock()
	m.Parent = append(m.Parent, n)
}

func (m *M) Unique() string {
	return m.Cf.Unique
}

func Find(name string) *M {
	lock.RLock()
	defer lock.RUnlock()
	return byname[name]
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

func (m *M) Loggit(tag string, msg string) {
	m.Lock.Lock()
	defer m.Lock.Unlock()
	m.loggitL(tag, msg)
}

func (m *M) loggitL(tag string, msg string) {

	diag.Verbose("%s %s %s", m.Cf.Unique, tag, msg)
	m.appendToLog(tag, msg)

}
