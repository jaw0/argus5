// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Sep-17 14:25 (EDT)
// Function: emit json for web

package monel

import (
	"encoding/json"
	"strconv"
	"strings"

	"argus/argus"
	"argus/notify"
	"argus/web"
)

type objectDescr struct {
	Unique string
	Name   string
	Label  string
}

type childSummaryCount struct {
	Up  int
	Dn  int
	Ov  int
	Tot int
	Sev argus.Status
}
type childSummary struct {
	Desc     objectDescr
	Counts   childSummaryCount
	Status   argus.Status
	HasChild bool
	Summy    [argus.MAXSTATUS + 1][]*objectDescr
}

func init() {
	web.Add(web.PRIVATE, "/api/page", apiJson)
}

func apiJson(ctx *web.Context) {

	since, _ := strconv.ParseInt(ctx.Get("since"), 10, 64)
	obj := ctx.Get("obj")

	m := Find(obj)

	if m == nil {
		ctx.W.WriteHeader(404)
		return
	}

	var creds []string
	if ctx.User != nil {
		creds = strings.Fields(ctx.User.Groups)
	}

	if !ACLPermitsUser(m.Cf.ACL_Page, creds) {
		ctx.W.WriteHeader(403)
		return
	}

	d := make(map[string]interface{})

	// general metadata - always include
	m.webMeta(d)

	// only include these if something has changed
	if !m.webChangedSince(since) {
		d["unchanged"] = true
	} else {
		mond := make(map[string]interface{})
		deco := make(map[string]interface{})
		d["mon"] = mond
		d["deco"] = deco

		m.webDecor(deco)

		m.webJson(creds, mond)
		m.Me.WebJson(mond)
	}

	// marshal
	js, _ := json.MarshalIndent(d, "", "  ")

	ctx.W.Header().Set("Content-Type", "application/json; charset=utf-8")
	ctx.W.Write(js)
}

func (m *M) webChangedSince(since int64) bool {

	m.Lock.RLock()
	wt := m.WebTime
	childs := m.Me.Children()
	m.Lock.RUnlock()

	if wt > since {
		return true
	}

	for _, c := range childs {
		c.Lock.RLock()
		wt := c.WebTime
		childs := c.Me.Children()
		c.Lock.RUnlock()
		if wt > since {
			return true
		}

		for _, cc := range childs {
			cc.Lock.RLock()
			wt := cc.WebTime
			cc.Lock.RUnlock()
			if wt > since {
				return true
			}
		}
	}

	return false
}

// ################################################################

func (m *M) webMeta(md map[string]interface{}) {

	m.Lock.RLock()
	m.Lock.RUnlock()

	md["alarm"] = m.P.Alarm
	md["sirentime"] = m.P.SirenTime
	md["webtime"] = m.WebTime
}

func (m *M) webJson(creds []string, md map[string]interface{}) {

	m.Lock.RLock()

	md["status"] = m.P.Status
	md["ovstatus"] = m.P.OvStatus
	md["transtime"] = m.P.TransTime
	md["override"] = m.P.Override
	md["annotation"] = m.P.Annotation
	md["reason"] = m.P.Reason
	md["stats"] = m.P.Stats // XXX
	md["log"] = m.P.Log     // ok. log is only ever appended to

	// notifies
	not := make([]*notify.ExportInfo, 0, len(m.Notifies))
	for _, n := range m.Notifies {
		not = append(not, n.WebExport())
	}
	md["notify"] = not

	childs := m.Me.Children()
	m.Lock.RUnlock()

	// children summary
	var childsum []*childSummary

	for _, c := range childs {
		if c.Cf.Hidden || !ACLPermitsUser(c.Cf.ACL_Page, creds) {
			continue
		}

		cs := c.childSummary(creds)
		childsum = append(childsum, cs)
	}

	md["childsum"] = childsum
}

func (m *M) childSummary(creds []string) *childSummary {

	m.Lock.RLock()
	cs := &childSummary{
		Desc:   objectDescr{m.Cf.Unique, m.Cf.Uname, m.Cf.Label},
		Status: m.P.OvStatus,
	}

	// summarize ovstatussummy
	cs.Counts.Sev = m.P.OvStatus
	cs.Counts.Up = m.P.OvStatusSummary[int(argus.CLEAR)] + m.P.OvStatusSummary[int(argus.DEPENDS)]
	cs.Counts.Ov = m.P.OvStatusSummary[int(argus.OVERRIDE)]

	for sev := argus.WARNING; sev <= argus.CRITICAL; sev++ {
		cs.Counts.Dn += m.P.OvStatusSummary[int(sev)]
		if m.P.OvStatusSummary[int(sev)] > 0 {
			cs.Counts.Sev = sev
		}
	}
	cs.Counts.Tot = cs.Counts.Up + cs.Counts.Dn + cs.Counts.Ov
	childs := m.Me.Children()
	m.Lock.RUnlock()

	dl.Debug("%s %v -> %v", m.Cf.Unique, m.P.OvStatusSummary, cs)

	for _, cc := range childs {
		if cc.Cf.Hidden || !ACLPermitsUser(cc.Cf.ACL_Page, creds) {
			continue
		}

		cc.Lock.RLock()
		st := cc.P.OvStatus
		desc := &objectDescr{cc.Cf.Unique, cc.Cf.Uname, cc.Cf.Label}
		cs.Summy[st] = append(cs.Summy[st], desc)
		cs.HasChild = true
		cc.Lock.RUnlock()
	}

	return cs
}

// gather the data needed for static labels, buttons, etc
func (m *M) webDecor(md map[string]interface{}) {

	m.Lock.RLock()
	m.Lock.RUnlock()

	md["name"] = m.Cf.Uname
	md["unique"] = m.Cf.Unique
	md["label"] = m.Cf.Label
	md["friendly"] = m.Cf.Friendlyname
	md["note"] = m.Cf.Note
	md["info"] = m.Cf.Info
	md["details"] = m.Cf.Details
	md["comment"] = m.Cf.Comment

	var parent []objectDescr
	var child []objectDescr

	for _, p := range m.Parent {
		parent = append(parent, objectDescr{p.Cf.Unique, p.Cf.Uname, p.Cf.Label})
	}
	for _, c := range m.Children {
		if c.Cf.Hidden {
			continue
		}
		child = append(child, objectDescr{c.Cf.Unique, c.Cf.Uname, c.Cf.Label})
	}
}
