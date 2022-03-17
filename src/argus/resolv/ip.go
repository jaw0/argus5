// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Aug-30 21:35 (EDT)
// Function: async resolver

package resolv

import (
	"errors"
	"net"
	"strings"
	"time"

	"argus.domain/argus/clock"
	"argus.domain/argus/configure"
)

const TOOLONG = int64(300 * time.Second)

type Conf struct {
	Hostname  string
	Hostname_ string
	IPVersion string
}

type IP struct {
	Cf      Conf
	Fqdn    string
	name    string
	cache   *cacheResult
	ipvpref []int
	asis    bool
	ipv     int
	idx     int
	created int64
}

// for testing
func New(name string, ver []int) *IP {

	ip := &IP{name: name, ipvpref: ver, created: clock.Nano()}
	Request(name, ip.ipvpref)
	return ip
}

func Config(conf *configure.CF) (*IP, error) {

	ip := &IP{created: clock.Nano()}
	conf.InitFromConfig(&ip.Cf, "ip", "")

	if ip.Cf.Hostname == "" && ip.Cf.Hostname_ != "" {
		// backdoor default (see url)
		ip.Cf.Hostname = ip.Cf.Hostname_
	}

	if ip.Cf.Hostname == "" {
		return nil, errors.New("hostname not specified")
	}
	name := ip.Cf.Hostname

	if name == "localhost" {
		name = "127.0.0.1"
	}

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
	Request(name, ip.ipvpref)

	return ip, nil
}

func (a *IP) Addr() (string, int, bool) {

	if a.asis {
		return a.name, a.ipv, false
	}

	now := clock.Nano()
	e := Get(a.useName())
	if e != nil {
		a.cache = e
	}
	e = a.cache

	if e == nil {
		Request(a.name, a.ipvpref)

		if a.created+TOOLONG < now {
			return "", 0, true
		}
		return "", 0, false
	}

	if e.expires <= now {
		dl.Debug("expired %s; %v", e.fqdn, e)
		e.Refresh(0)
	}
	if a.Fqdn == "" && e.fqdn != "" {
		a.Fqdn = e.fqdn
	}

	idx := a.idx
	aln := len(e.addrv4) + len(e.addrv6)
	if aln > 0 {
		idx %= aln
	}

	for _, pref := range a.ipvpref {

		switch pref {
		case 4:
			if len(a.ipvpref) == 1 && len(e.addrv4) > 0 {
				idx %= len(e.addrv4)
			}
			if idx >= len(e.addrv4) {
				idx -= len(e.addrv4)
			} else {
				return e.addrv4[idx], 4, false
			}
		case 6:
			if len(a.ipvpref) == 1 && len(e.addrv6) > 0 {
				idx %= len(e.addrv6)
			}
			if idx >= len(e.addrv6) {
				idx -= len(e.addrv6)
			} else {
				return e.addrv6[idx], 6, false
			}
		}
	}

	return "", 0, true
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
		ip.ipvpref = []int{4, 6}
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

	if a.asis {
		return
	}

	e := a.cache
	if e != nil {
		e.Refresh(secs)
	} else {
		Request(a.useName(), a.ipvpref)
	}
}

func (a *IP) useName() string {
	if a.Fqdn != "" {
		return a.Fqdn
	}
	if a.cache != nil && a.cache.fqdn != "" {
		return a.cache.fqdn
	}
	return a.name
}

func (a *IP) TryAnother() {
	a.idx++

}

func (ip *IP) Hostname() string {
	return ip.name
}
