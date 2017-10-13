// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Sep-07 19:28 (EDT)
// Function: notifications

package notify

import (
	"expvar"
	"os"
	"strconv"
	"sync"
	"time"

	"argus/argus"
	"argus/clock"
	"argus/config"
	"argus/configure"
	"argus/diag"
	"argus/sched"
)

type Remover interface {
	RemoveNotify(*N)
}

type GlobalConf struct {
	Mail_From        string
	Message_Fmt      string
	Message_Style    string
	ACL_NotifyDetail string
	ACL_NotifyList   string
	ACL_NotifyAck    string
	Notify_Discard   int64 `cfconv:"timespec"`
}

type NewConf struct {
	Conf         *Conf
	Unique       string
	ShortName    string
	FriendlyName string
	Reason       string
	Result       string
	OvStatus     argus.Status
	PrevOv       argus.Status
}

type Conf struct {
	Notify        [argus.CRITICAL + 1]*argus.Schedule `cfconv:"dotsev"`
	NotifyAlso    string
	NotifyAudit   string
	MessageUp     string
	MessageDn     string
	UnAck_Timeout int64                      `cfconv:"timespec"`
	Renotify      int64                      `cfconv:"timespec"`
	AutoAck       [argus.CRITICAL + 1]bool   `cfconv:"dotsev"`
	AckOnUp       [argus.CRITICAL + 1]bool   `cfconv:"dotsev"`
	Ack_On_Better [argus.CRITICAL + 1]bool   `cfconv:"dotsev"`
	Ack_On_Worse  [argus.CRITICAL + 1]bool   `cfconv:"dotsev"`
	Escalate      [argus.CRITICAL + 1]string `cfconv:"dotsev"`
	// QQQ - ack on override?
}

type Persist struct {
	IdNo         int
	Created      int64
	LastSent     int64
	IsActive     bool
	StepNo       int
	Escalated    bool
	Message      string
	MessageFmted string // result of message_fmt
	Unique       string
	ShortName    string
	FriendlyName string
	Reason       string
	Result       string
	OvStatus     argus.Status // status that caused the notification
	PrevOv       argus.Status // status prior to OvStatus
	CurrOv       argus.Status // current status
	Status       map[string]string
	SendTo       []SendDat
	Log          []LogDat
}
type ExportInfo struct {
	IdNo     int
	Created  int64
	IsActive bool
	OvStatus argus.Status
}

type SendDat struct {
	When int64
	Dst  []string
}
type LogDat struct {
	When int64
	Who  string
	Msg  string
}

type N struct {
	cf   *Conf
	mon  Remover
	lock sync.RWMutex
	p    Persist
}

var lock sync.RWMutex
var byid = make(map[int]*N)
var actives = make(map[int]*N)
var dstQueue = make(map[string]*queuedat)

var dl = diag.Logger("notify")
var notechan = make(chan *N, 16)
var idno = 1000
var NActive = expvar.NewInt("notifies")

var globalDefaults = GlobalConf{
	Mail_From:        "argus",
	Message_Fmt:      "{{.IDNO}} {{.CREATED}} {{.CONTENT}}",
	ACL_NotifyDetail: "staff root",
	ACL_NotifyList:   "staff root",
	ACL_NotifyAck:    "staff root",
	Notify_Discard:   30 * 24 * 3600,
}
var NotifyCfDefaults = Conf{
	Renotify:      300,
	UnAck_Timeout: 3600,
	Ack_On_Worse:  [argus.CRITICAL + 1]bool{int(argus.UNKNOWN): true},
}

const SECSNANO = 1000000000

// ################################################################

func NumActive() int {
	lock.RLock()
	defer lock.RUnlock()
	return len(actives)
}

func (n *N) Update(status argus.Status) {

	if n.updateStatus(status) {
		notechan <- n
	}
}

func (n *N) updateStatus(status argus.Status) bool {

	n.lock.Lock()
	defer n.lock.Unlock()

	if !n.p.IsActive {
		return false
	}
	n.p.CurrOv = status
	return true
}

func (n *N) log(who string, msg string) {

	n.p.Log = append(n.p.Log, LogDat{
		When: clock.Unix(),
		Who:  who,
		Msg:  msg,
	})
}

func (n *N) IdNo() int {
	return n.p.IdNo
}

func (n *N) WebExport() *ExportInfo {
	n.lock.RLock()
	defer n.lock.RUnlock()

	return &ExportInfo{n.p.IdNo, n.p.Created * SECSNANO, n.p.IsActive, n.p.OvStatus}
}

// ################################################################

// transmit( dst, []*N, escalated, )bool
//  pass to method

// ################################################################

func nextIdNo() int {

	lock.Lock()
	defer lock.Unlock()
	idno++
	saveIdNo()
	return idno
}

// ################################################################

func Init() {
	loadIdNo()
	go worker()

	sched.NewFunc(&sched.Conf{
		Freq: 3600,
		Text: "notify clean up",
		Auto: true,
	}, janitor)
}

func Stop() {

	lock.RLock()
	defer lock.RUnlock()

	for _, n := range byid {
		n.Save()
	}
}

func Configure(cf *configure.CF) {
	cf.InitFromConfig(&globalDefaults, "notify", "")
}

// ################################################################

func worker() {

	tock := time.NewTicker(60 * time.Second)
	defer tock.Stop()

	for {
		select {
		// case <-stop:
		case n := <-notechan:
			n.work()
			runQueues()

		case <-tock.C:
			notifMaintenance()
			runQueues()
			break

		}
	}
}

func (n *N) work() {

	dl.Debug("recvd %d", n.p.IdNo)
	lock.Lock()
	defer lock.Unlock()
	if byid[n.p.IdNo] == nil {
		// new
		dl.Debug("new notify")
		byid[n.p.IdNo] = n
		if n.p.IsActive {
			actives[n.p.IdNo] = n
			NActive.Set(int64(len(actives)))
		}
		n.maybeQueue()
	} else {
		dl.Debug("update notify")
		n.statusChange()
	}
}

// package lock is already held
func (n *N) statusChange() {

	if !n.p.IsActive {
		return
	}

	n.maybeAck()
}

func (n *N) discard() {

	delete(byid, n.p.IdNo)
	delete(actives, n.p.IdNo)
	NActive.Set(int64(len(actives)))

	go n.mon.RemoveNotify(n)
}

func notifMaintenance() {

	now := clock.Unix()

	lock.Lock()
	defer lock.Unlock()

	// resend? timeout?
	for _, n := range actives {
		if n.cf.UnAck_Timeout > 0 && n.p.Created+n.cf.UnAck_Timeout < now {
			n.ack("timeout")
			continue
		}
		n.maybeQueue()
	}

	// discard old
	for _, n := range byid {

		if globalDefaults.Notify_Discard > 0 && n.p.Created+globalDefaults.Notify_Discard < now {
			n.discard()
		}
	}
}

func janitor() {
	cleanOldFiles()
}

func cleanOldFiles() {

	cf := config.Cf()
	if cf.Datadir == "" {
		return
	}
	dir := cf.Datadir + "/notify"

	f, err := os.Open(dir)
	if err != nil {
		dl.Verbose("cannot open notify dir: %v", err)
		return
	}

	all, _ := f.Readdirnames(-1)
	f.Close()

	for _, id := range all {
		if id[0] == '.' {
			continue
		}

		idno, _ := strconv.Atoi(id)

		if byid[idno] != nil {
			continue
		}

		dl.Debug("removing old orphaned file '%s'", id)
		os.Remove(dir + "/" + id)
	}
}
