// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Sep-02 16:51 (EDT)
// Function: monitor elements

package monel

import (
	"expvar"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"sync"

	"argus/argus"
	"argus/clock"
	"argus/configure"
	"argus/diag"
	"argus/notify"
)

const STOPWORKERS = 5

var dl = diag.Logger("monel")

var lock sync.RWMutex
var byname = make(map[string]*M)
var isdown = make(map[string]*M)
var inoverride = make(map[string]*M)
var NMonel = expvar.NewInt("objects")

// Service, Group, Alias
type Moneler interface {
	Persist(map[string]interface{})
	Restore(map[string]interface{})
	WebJson(map[string]interface{})
	WebMeta(map[string]interface{})
	Config(*configure.CF) error
	Dump(argus.Dumper)
	CheckNow()
	Init() error
	DoneConfig()
	Recycle()
	Children() []*M
	Self() *M
	GraphList(string, []interface{}) []interface{}
}

type Conf struct {
	Uname        string
	Unique       string
	Label        string
	Friendlyname string
	Note         string
	Info         string
	Details      string
	Comment      string
	Depends      string
	Debug        bool
	Sort         bool
	Overridable  bool
	Passive      bool
	Countstop    bool
	Hidden       bool
	Graph        bool
	Siren        [argus.CRITICAL + 1]bool            `cfconv:"dotsev"`
	Sendnotify   [argus.CRITICAL + 1]*argus.Schedule `cfconv:"dotsev"`
	Gravity      argus.Gravity
	ACL_Page     string
	ACL_Override string
	ACL_Annotate string
	ACL_CheckNow string
	ACL_About    string
	GraphLabel   string
	YLabel       string
	Title        string
	// ...
}

var defaults = Conf{
	Sort:         true,
	Overridable:  true,
	Siren:        [...]bool{true, false, false, false, false, false},
	Gravity:      argus.GRAV_DN,
	ACL_Page:     "user staff root",
	ACL_Override: "staff root",
	ACL_Annotate: "staff root",
	ACL_CheckNow: "root",
	ACL_About:    "root",
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
	OvStatusSummary []int // NB - mapstructure cannot array, it can slice...
	Interesting     bool
	TransTime       int64
	SirenTime       int64
	Culprit         string
	Stats           Stats
	Log             []*Log
}

type M struct {
	Name         string
	Me           Moneler
	Lock         sync.RWMutex
	Parent       []*M
	Children     []*M
	Cf           Conf
	NotifyCf     *notify.Conf
	P            Persist
	ConfCF       *configure.CF
	WebTime      int64
	Filename     string
	DirName      string
	Uname        string // default set by subclass, conf overrides
	Label        string // ""
	Friendlyname string // ""
	Depends      []string
	Notifies     []*notify.N
	Interesting  bool
}

func New(me Moneler, parent *M) *M {

	m := &M{
		WebTime: clock.Nano(),
		Me:      me,
	}

	m.P.OvStatusSummary = make([]int, int(argus.MAXSTATUS+1))
	if parent != nil {
		m.Parent = []*M{parent}
	}

	m.Cf = defaults

	if parent == nil {
		m.Cf.Sort = false
	}

	return m
}

func Stop() {

	// save state
	lock.RLock()
	defer lock.RUnlock()

	c := make(chan *M)
	for i := 0; i < STOPWORKERS; i++ {
		go stopper(c)
	}

	for _, m := range byname {
		c <- m
	}
	close(c)
}

func stopper(c chan *M) {

	for m := range c {
		m.Persist()
	}
}

// ################################################################

func (m *M) SetNames(uname string, label string, friendly string) {
	m.Uname = uname
	m.Label = label
	m.Friendlyname = friendly
}

func (m *M) Config(conf *configure.CF) error {

	m.ConfCF = conf
	m.Name = conf.Name

	conf.InitFromConfig(&m.Cf, "monel", "")

	if m.Cf.Passive {
		for i := 0; i < len(m.Cf.Sendnotify); i++ {
			m.Cf.Sendnotify[i] = nil
		}
		for i := 0; i < len(m.Cf.Siren); i++ {
			m.Cf.Siren[i] = false
		}
	}

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
	NMonel.Set(int64(len(byname)))
	lock.Unlock()
}

func (m *M) Recycle(cascade bool) {

	m.Persist()

	lock.Lock()
	delete(byname, m.Cf.Unique)
	delete(isdown, m.Cf.Unique)
	delete(inoverride, m.Cf.Unique)
	NMonel.Set(int64(len(byname)))
	lock.Unlock()

	m.Lock.Lock()
	defer m.Lock.Unlock()

	for _, c := range m.Children {
		c.Lock.Lock()
		c.Parent = removeFromList(c.Parent, m)
		c.Lock.Unlock()

		if cascade {
			c.Recycle(cascade)
		}
	}

	for _, c := range m.Parent {
		c.Lock.Lock()
		c.Parent = removeFromList(c.Children, m)
		c.Lock.Unlock()
	}

	m.Children = nil
	m.Parent = nil

	m.Me.Recycle()
}

func removeFromList(list []*M, x *M) []*M {

	idx := -1
	for i, l := range list {
		if l == x {
			idx = i
		}
	}
	if idx == -1 {
		// not found
		return list
	}

	// delete
	copy(list[idx:], list[idx+1:])
	list[len(list)-1] = nil
	list = list[:len(list)-1]
	return list
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

	m.NotifyCf = notify.NewCF()
	conf.InitFromConfig(m.NotifyCf, "notify", "")
}

func (m *M) Status() (argus.Status, argus.Status) {
	m.Lock.RLock()
	defer m.Lock.RUnlock()
	return m.P.Status, m.P.OvStatus
}

func (m *M) GetResult() string {
	m.Lock.RLock()
	defer m.Lock.RUnlock()
	return m.P.Result
}

func (m *M) DoneConfig() {

	m.determineInteresting()

	for _, child := range m.Children {
		child.DoneConfig()
		wt := child.WebTime
		if wt > m.WebTime {
			m.WebTime = wt
		}
	}

	m.ConfCF.CheckTypos()
	m.ConfCF.DrainCache()
	m.sortChildren()
	m.resolveDepends()

	m.Me.DoneConfig()
	m.determineStatus()
	m.determineSummary()
	m.setAlarm()
	m.updateIsDown(m.P.OvStatus)
}

func (m *M) determineInteresting() {

	ip := false

	if len(m.Parent) > 0 && m.Parent[0].Interesting {
		// He smiled a kind of sickly smile and curled up on the floor
		// And the subsequent proceedings interested him no more.
		//   -- Francis Bret Harte, The Society upon the Stanislaus
		return
	}
	// Thank you,' said Alice, `it's very interesting.
	//   -- Alice in Wonderland
	if m.Cf.Countstop {
		ip = true
	}
	if m.Cf.Gravity == argus.GRAV_UP {
		ip = true
	}
	if len(m.Children) == 0 {
		ip = true
	}
	if m.Cf.Passive {
		ip = false
	}
	if m.Cf.Hidden {
		ip = false
	}

	m.Interesting = ip
}

func (m *M) sortChildren() {

	if !m.Cf.Sort || len(m.Children) == 0 {
		return
	}

	re, err := regexp.Compile("\\d+")
	if err != nil {
		dl.Bug("regexp failed: %v", err)
		return
	}
	type schwartz struct {
		key string
		mon *M
	}

	childs := make([]schwartz, len(m.Children))

	for i, c := range m.Children {
		name := re.ReplaceAllStringFunc(c.Cf.Unique, func(s string) string {
			v, _ := strconv.Atoi(s)
			return fmt.Sprintf("%09d", v)
		})

		childs[i].mon = c
		childs[i].key = name
	}

	sort.Slice(childs, func(i, j int) bool { return childs[i].key < childs[j].key })

	for i := range m.Children {
		m.Children[i] = childs[i].mon
	}

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

	if m.Cf.Label == "" && m.Cf.Uname != "" {
		m.Cf.Label = m.Cf.Uname
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
	m.Filename = argus.FileEncode(name)
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

	if tag == "TRANSITION" && msg == "" {
		msg = m.P.OvStatus.String()
	}

	diag.Verbose("%s %s %s", m.Cf.Unique, tag, msg)
	m.appendToLog(tag, msg)

}

// ################################################################

func (m *M) RemoveNotify(remove *notify.N) {

	m.Lock.Lock()
	defer m.Lock.Unlock()

	for i, n := range m.Notifies {
		if n != remove {
			continue
		}

		copy(m.Notifies[i:], m.Notifies[i+1:])
		m.Notifies = m.Notifies[:len(m.Notifies)-1]
		break
	}
}
