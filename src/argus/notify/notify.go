// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Sep-07 19:28 (EDT)
// Function: notifications

package notify

import (
	"sync"
	"time"

	"argus/argus"
	"argus/clock"
	"argus/configure"
	"argus/diag"
)

type GlobalConf struct {
	Mail_From     string
	Message_Fmt   string
	Message_Style string
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

var globalDefaults = GlobalConf{
	Message_Fmt: "{{.IDNO}} {{.CREATED}} {{.CONTENT}}",
}
var NotifyCfDefaults = Conf{
	Renotify:      300,
	UnAck_Timeout: 3600,
	Ack_On_Worse:  [argus.CRITICAL + 1]bool{int(argus.UNKNOWN): true},
}

// ################################################################

func (n *N) Update(status argus.Status) {

	n.lock.Lock()
	defer n.lock.Unlock()
	n.p.CurrOv = status
	notechan <- n
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

func notifMaintenance() {

	lock.Lock()
	defer lock.Unlock()

	for _, n := range actives {
		n.maybeQueue()
	}

	// RSN - delete old?
}
