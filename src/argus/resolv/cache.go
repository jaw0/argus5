// Copyright (c) 2018
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2018-Oct-16 11:07 (EDT)
// Function:

package resolv

import (
	"sync"

	"argus.domain/argus/clock"
)

const (
	F_IP4 = 1
	F_IP6 = 2
)

type cacheResult struct {
	fqdn    string
	expires int64
	ttl     int
	addrv4  []string
	addrv6  []string
}

type cacheEntry struct {
	lock     sync.RWMutex
	name     string
	result   *cacheResult
	neededby int64
	updated  int64
	prefm    int
	underway bool
}

var cacheLock sync.RWMutex
var cache = make(map[string]*cacheEntry)

var whenLock sync.Mutex
var needWhen = make(map[string]int64)

func Request(name string, pref []int) {

	prefm := prefToMask(pref)
	dl.Debug("+ %d %s", prefm, name)

	ce := get(name)
	if ce == nil {
		ce = newCacheE(name)
	}

	ce.lock.Lock()
	defer ce.lock.Unlock()

	if prefm&ce.prefm != prefm {
		ce.request(name, prefm)
		return
	}
}

func refresh(name string) {

	ce := get(name)
	if ce == nil {
		return
	}

	ce.lock.Lock()
	defer ce.lock.Unlock()

	ce.request(name, 0)
}

func (ce *cacheEntry) request(name string, prefm int) {

	ce.prefm |= prefm
	ce.underway = true

	dl.Debug("query %d %s", ce.prefm, name)

	whenLock.Lock()
	defer whenLock.Unlock()
	delete(needWhen, name)

	// send to resolver
	if !query(name, ce.prefm) {
		ce.prefm = 0
		ce.underway = false
	}

}

// process results from query
func Result(name string, prefm int, res *cacheResult) {

	dl.Debug("cache+ %s -> %v", name, res)
	addCache(name, prefm, res)
	if res != nil && res.fqdn != name {
		addCache(res.fqdn, prefm, res)
	}
}

func addCache(name string, prefm int, res *cacheResult) {

	ce := get(name)
	if ce == nil {
		ce = newCacheE(name)
	}

	now := clock.Nano()

	ce.lock.Lock()
	defer ce.lock.Unlock()

	ce.prefm |= prefm
	ce.underway = false

	if ce.result != nil && ce.result.expires < now {
		// always replace result if expired
		dl.Debug("expired %s %d %d %d", name, ce.result.expires, now, ce.result.expires-now)
		ce.result = nil
	}

	if ce.result.size() <= res.size() {
		// don't replace a full result set with an empty or partial
		dl.Debug("cache %s +! %d->%d", name, ce.result.size(), res.size())
		ce.result = res
		ce.updated = now
	}
}

func get(name string) *cacheEntry {
	cacheLock.RLock()
	defer cacheLock.RUnlock()
	return cache[name]
}

func Get(name string) *cacheResult {

	ce := get(name)
	if ce == nil {
		dl.Debug("not found %s", name)
		return nil
	}

	ce.lock.RLock()
	defer ce.lock.RUnlock()
	return ce.result
}

func (e *cacheResult) Refresh(secs int) {

	if secs == 0 {
		refresh(e.fqdn)
		return
	}

	now := clock.Nano()
	treq := now + int64(secs-SOONER)*NANOSECONDS

	if e.expires > treq {
		// results will still be good
		return
	}

	refreshWhen(e.fqdn, treq)
}

func refreshWhen(name string, when int64) {

	whenLock.Lock()
	defer whenLock.Unlock()

	curr, ok := needWhen[name]

	if ok && curr < when {
		return
	}

	needWhen[name] = when
}

func needMaint() []string {

	var res []string
	now := clock.Nano()
	whenLock.Lock()
	defer whenLock.Unlock()

	for name, when := range needWhen {
		if when > now {
			continue
		}
		res = append(res, name)
	}

	return res
}

func cacheMaint() {

	need := needMaint()

	for _, name := range need {
		refresh(name)
	}
}

func newCacheE(name string) *cacheEntry {
	cacheLock.Lock()
	defer cacheLock.Unlock()

	ce := &cacheEntry{name: name}
	cache[name] = ce
	return ce
}

func prefToMask(pref []int) int {

	prefm := 0

	for _, p := range pref {
		switch p {
		case 4:
			prefm |= F_IP4
		case 6:
			prefm |= F_IP6
		}
	}

	return prefm
}

func (res *cacheResult) size() int {
	s := 0

	if res == nil {
		return 0
	}
	if len(res.addrv4) != 0 {
		s++
	}
	if len(res.addrv6) != 0 {
		s++
	}
	return s
}
