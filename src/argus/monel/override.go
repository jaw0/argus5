// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Sep-21 22:34 (EDT)
// Function: overrides

package monel

import (
	"encoding/json"
	"strconv"

	"argus/argus"
	"argus/clock"
	"argus/sched"
	"argus/web"
)

func init() {
	web.Add(web.WRITE, "/api/override", webOverride)
}

func (m *M) SetOverride(ov *argus.Override) {

	if ov.Text != "" {
		m.Loggit("OVERRIDE", ov.Text)
	}
	if ov.User != "" {
		m.Loggit("OVERRIDE", "enabled by "+ov.User)
	}

	m.Lock.Lock()
	m.P.Override = ov
	m.WebTime = clock.Nano()
	m.Lock.Unlock()

	lock.Lock()
	inoverride[m.Cf.Unique] = m
	lock.Unlock()

	m.ovPropDown()
	m.ReUpdate("override")
	m.setOverrideExpire()
}

func (m *M) setOverrideExpire() {

	var exp int64

	m.Lock.RLock()
	if m.P.Override != nil {
		exp = m.P.Override.Expires
	}
	m.Lock.RUnlock()

	if exp != 0 {
		sched.At(exp, "override expire", func() {
			m.overrideExpire()
		})
	}
}

func (m *M) overrideExpire() {

	var exp int64

	m.Lock.RLock()
	if m.P.Override != nil {
		exp = m.P.Override.Expires
	}
	m.Lock.RUnlock()

	if exp != 0 && exp <= clock.Unix() {
		m.DelOverride("system", "expired")
	}
}

func (m *M) DelOverride(user string, reason string) {

	if reason != "" {
		m.Loggit("OVERRIDE", reason)
	}
	if user != "" {
		m.Loggit("OVERRIDE", "removed by "+user)
	}

	m.Lock.Lock()
	m.P.Override = nil
	m.WebTime = clock.Nano()
	m.Lock.Unlock()

	lock.Lock()
	delete(inoverride, m.Cf.Unique)
	lock.Unlock()

	m.ovPropDown()
	m.ReUpdate("override")
}

// set AncInOv on all descendants
func (m *M) ovPropDown() {

	m.Lock.RLock()
	v := false
	if m.P.Override != nil || m.P.AncInOv {
		v = true
	}
	childs := m.Children
	m.Lock.RUnlock()

	for _, c := range childs {
		c.Lock.Lock()
		c.P.AncInOv = v
		c.Lock.Unlock()

		c.ovPropDown()
	}
}

func webOverride(ctx *web.Context) {

	m, creds := webObjUserCheck(ctx)

	if !argus.ACLPermitsUser(m.Cf.ACL_Override, creds) ||
		!m.Cf.Overridable {
		dl.Debug("denied")
		ctx.W.WriteHeader(403)
		return
	}

	text := ctx.Get("text")
	mode := ctx.Get("mode")
	expires, _ := strconv.ParseInt(ctx.Get("expires"), 10, 64)
	remove := ctx.Get("remove")

	if expires > 0 {
		expires += clock.Unix()
	}

	d := m.newWebMetaResponse(ctx)

	if remove != "" {
		m.DelOverride(ctx.User.Name, "")
	} else {

		ov := &argus.Override{
			User:    ctx.User.Name,
			Text:    text,
			Auto:    mode == "auto",
			Expires: expires,
			// ticket
		}

		m.SetOverride(ov)

		d["override"] = ov
	}

	js, _ := json.MarshalIndent(d, "", "  ")
	ctx.W.Header().Set("Content-Type", "application/json; charset=utf-8")
	ctx.W.Write(js)
}
