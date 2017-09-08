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
	"argus/notify"
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
}

type Conf struct {
	Uname        string
	Unique       string
	Label        string
	Friendlyname string
	Sort         bool
	Overridable  bool
	Note         string
	Info         string
	Details      string
	Comment      string
	Debug        bool
	Passive      bool
	Depends      string
	Siren        bool
	Nostatus     bool
	Gravity      argus.Gravity
	Countstop    bool
	Sendnotify   [argus.CRITICAL + 1]*argus.Schedule `cfconv:"dotsev"`
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
	Annotation      string
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
	// notif
}

type M struct {
	Me           Moneler
	Lock         sync.RWMutex
	Parent       []*M
	Children     []*M
	Cf           Conf
	NotifyCf     *notify.Conf
	P            Persist
	ConfCF       *configure.CF
	Filename     string
	DirName      string
	Uname        string // default set by subclass, conf overrides
	Label        string // ""
	Friendlyname string // ""
	Depends      []string
	Notifies     []*notify.N
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

func (m *M) SetNames(uname string, label string, friendly string) {
	m.Uname = uname
	m.Label = label
	m.Friendlyname = friendly
}

func (m *M) Config(conf *configure.CF) error {

	m.ConfCF = conf
	conf.InitFromConfig(&m.Cf, "monel", "")
	m.configureNotify(conf)

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

	m.statsInit()

	if len(m.Parent) != 0 {
		m.Parent[0].AddChild(m)
	}

	m.Me.Init()

	lock.Lock()
	byname[m.Cf.Unique] = m
	lock.Unlock()

}

func (m *M) configureNotify(conf *configure.CF) {
	// only configure if sendnotify is set

	confed := false
	for i := 0; i < len(m.Cf.Sendnotify); i++ {
		if m.Cf.Sendnotify[i] != nil {
			confed = true
		}
	}

	if !confed {
		return
	}

	m.NotifyCf = &notify.Conf{}
	// RSN - defaults
	conf.InitFromConfig(&m.NotifyCf, "notify", "")
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

	m.ConfCF.CheckTypos()
	m.ConfCF.DrainCache()
	m.sortChildren()
	m.resolveDepends()

	// determine interestingness

	m.Me.DoneConfig()
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
		m.Cf.Uname = m.Uname
	}
	if m.Cf.Label == "" {
		m.Cf.Label = m.Label
	}
	if m.Cf.Friendlyname == "" {
		m.Cf.Friendlyname = m.Friendlyname
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
