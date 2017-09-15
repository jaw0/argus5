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
	"argus/config"
	"argus/configure"
	"argus/diag"
)

type DARP struct {
	Name     string
	Pass     string
	Type     string
	Hostname string
	Port     int
}
type Status struct {
	IsUp  bool
	Lastt int64
}

type DarpServerer struct{}

const DEFAULTPORT = 2055 // not officially assigned

var dl = diag.Logger("darp")

// no lock, these are never modified after startup
var MyId = "local"
var MyDarp *DARP
var iAmSlave bool
var iHaveSlaves bool
var darpEnabled bool
var allDarp = make(map[string]*DARP)
var masters = make(map[string]*DARP)
var slaves = make(map[string]*DARP)

var lock sync.RWMutex
var darpStatus = make(map[string]*Status)

func New(conf *configure.CF) error {

	d := &DARP{Port: DEFAULTPORT}
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
	darpEnabled = true

	ccf := config.Cf()
	if ccf.DARP_Name != "" {
		MyId = ccf.DARP_Name
	}

	if name == MyId {
		MyDarp = d
		if t == "slave" {
			iAmSlave = true
		} else if len(slaves) > 0 {
			iHaveSlaves = true
		}
	} else if t == "slave" && MyDarp != nil && MyDarp.Type == "master" {
		iHaveSlaves = true
	}

	if t == "slave" {
		slaves[name] = d
	}
	if t == "master" {
		masters[name] = d
	}

	conf.CheckTypos()
	return nil
}

func Init() {

	if !darpEnabled {
		return
	}
	if MyDarp == nil {
		argus.ConfigWarning("", 0, "Darp configured, but no entry for myself. Disabling")
		return
	}

	// start server
	ob := &DarpServerer{}
	api.ServerNew(ob, "darp", "tcp", fmt.Sprintf(":%d", MyDarp.Port))

	// start clients

}

func (*DarpServerer) Connected(name string) {
}
func (*DarpServerer) Disco(name string) {

}
