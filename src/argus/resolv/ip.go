// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Aug-30 21:35 (EDT)
// Function: async resolver

package resolv

import (
	"errors"
	"net"
	"strings"

	"argus/clock"
	"argus/configure"
)

type Conf struct {
	Hostname  string
	Hostname_ string
	IPVersion string
}

type IP struct {
	Cf      Conf
	name    string
	cache   *cacheE
	ipvpref []int
	asis    bool
	ipv     int
}

// for testing
func New(name string, ver int) *IP {

	ip := &IP{name: name, ipv: ver, ipvpref: []int{ver}}
	lookup(name)
	return ip
}

func Config(conf *configure.CF) (*IP, error) {

	ip := &IP{}
	conf.InitFromConfig(&ip.Cf, "ip", "")

	if ip.Cf.Hostname == "" && ip.Cf.Hostname_ != "" {
		// backdoor default (see url)
		ip.Cf.Hostname = ip.Cf.Hostname_
	}

	if ip.Cf.Hostname == "" {
		return nil, errors.New("hostname not specified")
	}
	name := ip.Cf.Hostname

	// check for dotted quad / cologned octopus
	ipp := net.ParseIP(name)

	if ipp != nil {
		ip.ipv = 4
		if len(ipp) > 4 {
			ip.ipv = 6
		}

		ip.name = name
		ip.asis = true
		return ip, nil
	}

	// parse out ipv spec: eg. hostname._ipv6
	// overrides IPVersion

	dl.Debug("New: %s", name)

	ip.confPref()

	ldot := strings.LastIndex(name, "._ipv")
	dl.Debug("ldot %d; len %d", ldot, len(name))

	if ldot != -1 && ldot <= len(name)-6 {

		switch name[len(name)-1] {
		case '4':
			ip.ipvpref = []int{4}
		case '6':
			ip.ipvpref = []int{6}
		}
		name = name[:ldot]
		dl.Debug("ipv %d => %s", ip.ipv, name)
	}

	name = strings.ToLower(name)
	ip.name = name
	lookup(name)

	return ip, nil
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

	if len(e.result) > 0 && len(a.ipvpref) == 0 {
		// no preference - return first result
		r := e.result[0]
		return r.addr, r.ipv, false
	}

	var a4 []string
	var a6 []string

	for i, _ := range e.result {
		r := &e.result[i]

		switch r.ipv {
		case 4:
			a4 = append(a4, r.addr)
		case 6:
			a6 = append(a6, r.addr)
		}
	}

	for _, pref := range a.ipvpref {
		// return first result matching configured preference
		switch pref {
		case 4:
			if len(a4) > 0 {
				return a4[0], 4, false
			}
		case 6:
			if len(a6) > 0 {
				return a6[0], 6, false
			}
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

func (ip *IP) confPref() {

	if ip.Cf.IPVersion == "" {
		return
	}

	p := strings.Fields(ip.Cf.IPVersion)

	for _, v := range p {
		switch v {
		case "4":
			ip.ipvpref = append(ip.ipvpref, 4)
		case "6":
			ip.ipvpref = append(ip.ipvpref, 6)
		}
	}
}

func (a *IP) WillNeedIn(secs int) {

}

func (a *IP) TryAnother() {

}

func (a *IP) IsValid() {

}

func (a *IP) IsTimedOut() {

}

func (ip *IP) Hostname() string {
	return ip.name
}
