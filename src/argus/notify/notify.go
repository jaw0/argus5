// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Sep-07 19:28 (EDT)
// Function: notifications

package notify

import (
	"sync"

	"argus/argus"
	"argus/configure"
)

type GlobalConf struct {
	Mail_From string
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
	Message_Fmt   string
	Message_Style string
	UnAck_Timeout int64 `cfconv:"timespec"`
	Renotify      int64 `cfconv:"timespec"`
	AutoAck       [argus.CRITICAL + 1]bool
	Ack_On_Better [argus.CRITICAL + 1]bool
	Ack_On_Worse  [argus.CRITICAL + 1]bool
	Escalate      [argus.CRITICAL + 1]string
}

type Persist struct {
	IdNo int
}

type N struct {
	cf   *Conf
	lock sync.RWMutex
	p    Persist
}

var lock sync.RWMutex
var byid = make(map[int]*N)

//var glcf = globalDefaults
// var NotifyCfDefaults =

func New(ncf *NewConf) *N {

	n := &N{
		cf: ncf.Conf,
		p: Persist{
			IdNo: nextIdNo(),
		},
	}

	n.Save()

	return n
}

func Load(cf *Conf, idno int) *N {

	return nil
}

func (n *N) Save() {

}

func (n *N) Update(status argus.Status) {

}

// ################################################################

// transmit( dst, []*N, escalated, )bool
//  pass to method

// ################################################################

func nextIdNo() int {
	return 0
}

func Init() {
	// start goro
}
func Stop() {

}

func Configure(cf *configure.CF) {

}
