// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Sep-14 19:02 (EDT)
// Function: distributed argus redundancy protocol

package darp

import (
	"errors"
	"fmt"
	"strings"
	"sync"

	"argus/api"
	"argus/argus"
	"argus/clock"
	"argus/config"
	"argus/configure"
	"argus/diag"
	"argus/resolv"
)

type Maker interface {
	Make(map[string]string) error
}

type DARP struct {
	Name         string
	Pass         string
	Type         string
	Fetch_Config string // name of master
	Hostname     string
	Port         int
	ip           *resolv.IP
	ch           chan *sendMsg
	// ...
}
type Status struct {
	IsUp  bool
	Lastt int64
}

type sendMsg struct {
	f string
	m map[string]string
}

type DarpServerer struct{}

const DEFAULTPORT = 2055 // not officially assigned

var dl = diag.Logger("darp")

// no lock, these are never modified after startup
var MyId = "local"
var MyDarp *DARP
var iHaveSlaves bool
var IsEnabled bool
var objMaker Maker
var allDarp = make(map[string]*DARP)
var masters = make(map[string]*DARP)

var lock sync.RWMutex
var darpStatus = make(map[string]*Status)
var noDarp = make(map[string]bool)

func New(conf *configure.CF) error {

	d := &DARP{Port: DEFAULTPORT, Name: conf.Name}
	conf.InitFromConfig(d, "darp", "")

	name := conf.Name

	// validate
	t := strings.ToLower(d.Type)

	if name == "" {
		return errors.New("darp name?")
	}
	if allDarp[name] != nil {
		return errors.New("duplicate darp name")
	}
	if t != "master" && t != "slave" {
		return errors.New("must specify type: master or slave")
	}
	if d.Hostname == "" {
		return errors.New("hostname not specified")
	}
	if d.Pass == "" {
		return errors.New("pass not specified")
	}

	allDarp[name] = d
	darpStatus[name] = &Status{}
	IsEnabled = true

	ccf := config.Cf()
	if ccf.DARP_Name != "" {
		MyId = ccf.DARP_Name
	}

	if name == MyId {
		MyDarp = d
	}
	if d.Type == "master" && d.Name != MyId {
		masters[d.Name] = d
	}

	d.ip = resolv.New(d.Hostname)

	conf.CheckTypos()
	return nil
}

func Init(mo Maker) {

	objMaker = mo

	if !IsEnabled {
		noDarp[MyId] = true
		return
	}
	if MyDarp == nil {
		argus.ConfigWarning("", 0, "Darp configured, but no entry for myself. Disabling")
		return
	}

	// will anyone send me status updates?
	if MyDarp.Type == "master" {
		if len(allDarp) > 1 {
			iHaveSlaves = true
		}
	}

	// start server
	ob := &DarpServerer{}
	api.ServerNew(ob, "darp", "tcp", fmt.Sprintf(":%d", MyDarp.Port))

	// start clients
	for _, d := range masters {
		dx := d
		dl.Debug("starting darp client to %s", d.Name)
		dx.ch = make(chan *sendMsg, 100)
		go dx.StartClient()
	}
}

// ################################################################

func copyStatus() map[string]*Status {

	new := make(map[string]*Status)

	for n, s := range darpStatus {
		ns := Status{}
		ns = *s
		new[n] = &ns
	}

	return new
}

func (x *DarpServerer) Connected(name string) {

	now := clock.Unix()

	lock.Lock()
	defer lock.Unlock()

	s := darpStatus[name]
	if s == nil {
		return
	}
	s.IsUp = true
	s.Lastt = now
}

func (x *DarpServerer) Disco(name string) {

	lock.Lock()
	defer lock.Unlock()

	s := darpStatus[name]
	if s == nil {
		return
	}

	s.IsUp = false
}

func GetStatuses() map[string]bool {

	if !IsEnabled {
		return noDarp
	}

	now := clock.Unix()
	st := make(map[string]bool)

	lock.RLock()
	defer lock.RUnlock()

	for n, s := range darpStatus {
		if n == MyId {
			st[n] = true
		} else if s.IsUp && s.Lastt > now-120 {
			st[n] = true
		} else {
			st[n] = false
		}
	}

	return st
}

// ################################################################

func IncludesTag(tags string, tag string) bool {
	return argus.IncludesTag(tags, tag, true)
}

func TellMyMasters(f string, m map[string]string) {

	msg := &sendMsg{f: f, m: m}

	for _, m := range masters {
		if m.ch == nil {
			continue
		}

		select {
		case m.ch <- msg:
		default:
			dl.Debug("send queue full")
		}
	}
}

func SendUpdate(obj string, status argus.Status, result string, reason string) {

	if !IsEnabled {
		return
	}

	TellMyMasters("update", map[string]string{
		"obj":    obj,
		"status": fmt.Sprintf("%d", status),
		"result": result,
		"reason": reason,
	})
}

// func FloodAll
