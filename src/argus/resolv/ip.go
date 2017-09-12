// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Aug-30 21:35 (EDT)
// Function: async resolver

package resolv

import (
	"net"
	"strings"

	"argus/clock"
)

type IP struct {
	name  string
	ipv   int
	cache *cacheE
	idx   int
	asis  bool
}

func New(name string) *IP {

	// check for dotted quad / cologned octopus
	ip := net.ParseIP(name)

	if ip != nil {
		ipv := 4
		if len(ip) > 4 {
			ipv = 6
		}
		return &IP{name: name, ipv: ipv, asis: true}
	}

	// parse out ipv spec: eg. hostname._ipv6

	ipv := 0
	ldot := strings.LastIndexByte(name, '.')
	if ldot != -1 && name[ldot+1:len(name)-2] == "_ipv" {

		if ipv == 0 {
			switch name[len(name)-1] {
			case '4':
				ipv = 4
			case '6':
				ipv = 6
			}
		}
		name = name[:ldot]
	}

	name = strings.ToLower(name)
	lookup(name)

	return &IP{
		name:  name,
		ipv:   ipv,
		cache: getCache(name),
	}
}

func (a *IP) Addr() (string, int, bool) {

	if a.asis {
		return a.name, a.ipv, false
	}

	e := a.cache
	if e == nil {
		e = getCache(a.name)
	}

	if e == nil {
		lookup(a.name)
		return "", 0, false
	}

	now := clock.Nano()
	e.lock.RLock()
	defer e.lock.RUnlock()

	if e.expire < clock.Nano() && !e.underway {
		lookup(a.name)
	}

	// RSN - rotate
	// first matching result
	for i, _ := range e.result {
		r := &e.result[i]

		if a.ipv == 0 || a.ipv == r.ipv {
			return r.addr, r.ipv, false
		}
	}

	if e.created+TOOLONG < now {
		return "", 0, true
	}

	return "", 0, false
}

// return ipv6 addrs with brackets
func (a *IP) AddrWB() (string, bool) {

	addr, ipv, ok := a.Addr()

	if addr == "" {
		return addr, ok
	}

	if ipv == 6 {
		return "[" + addr + "]", ok
	}
	return addr, ok
}

func (a *IP) WillNeedIn(secs int) {

}

func (a *IP) TryAnother() {

}

func (a *IP) IsValid() {

}

func (a *IP) IsTimedOut() {

}
