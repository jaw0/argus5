// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Sep-17 14:25 (EDT)
// Function: emit json for web

package monel

import (
	"encoding/json"

	"argus/argus"
	"argus/web"
)

func init() {
	web.Add(web.PUBLIC, "/api/page", apiJson)
}

func apiJson(ctx *web.Context) {

	//since := ctx.Get("since")
	obj := ctx.Get("obj")

	m := Find(obj)

	if m == nil {
		ctx.W.WriteHeader(404)
		return
	}
	// RSN - check acl

	d := make(map[string]interface{})
	// general
	m.webMeta(d)

	// since?

	mond := make(map[string]interface{})
	deco := make(map[string]interface{})
	d["mon"] = mond
	d["deco"] = deco

	// decorations - if since == 0
	m.webDecor(deco)

	m.webJson(mond)
	m.Me.WebJson(mond)

	// marshal
	js, _ := json.MarshalIndent(d, "", "  ")

	ctx.W.Header().Set("Content-Type", "application/json; charset=utf-8")
	ctx.W.Write(js)
}

// ################################################################

func (m *M) webMeta(md map[string]interface{}) {

	m.Lock.RLock()
	m.Lock.RUnlock()

	md["alarm"] = m.P.Alarm
	md["sirentime"] = m.P.SirenTime
	md["webtime"] = m.WebTime
}

func (m *M) webJson(md map[string]interface{}) {

	m.Lock.RLock()

	md["status"] = m.P.Status
	md["ovstatus"] = m.P.OvStatus
	md["override"] = m.P.Override
	md["annotation"] = m.P.Annotation
	md["reason"] = m.P.Reason
	md["stats"] = m.P.Stats
	md["log"] = m.P.Log

	childs := m.Children
	m.Lock.RUnlock()

	// children summary
	var childsum []*childSummary

	for _, c := range childs {
		if c.Cf.Hidden {
			continue
		}
		c.Lock.RLock()
		cs := &childSummary{
			Desc:   objectDescr{c.Cf.Unique, c.Cf.Uname, c.Cf.Label},
			Status: c.P.OvStatus,
		}
		childsum = append(childsum, cs)

		// summarize ovstatussummy
		cs.Counts.Sev = c.P.OvStatus
		cs.Counts.Up = c.P.OvStatusSummary[int(argus.CLEAR)] + c.P.OvStatusSummary[int(argus.DEPENDS)]
		cs.Counts.Ov = c.P.OvStatusSummary[int(argus.OVERRIDE)]

		for sev := argus.WARNING; sev <= argus.CRITICAL; sev++ {
			cs.Counts.Dn += c.P.OvStatusSummary[int(sev)]
			if c.P.OvStatusSummary[int(sev)] > 0 {
				cs.Counts.Sev = sev
			}
		}
		cs.Counts.Tot = cs.Counts.Up + cs.Counts.Dn + cs.Counts.Ov

		dl.Debug("%s %v -> %v", c.Cf.Unique, c.P.OvStatusSummary, cs)
		//for _, cc := range c.Children {
		//}

		c.Lock.RUnlock()
	}

	md["childsum"] = childsum
}

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
	Desc   objectDescr
	Counts childSummaryCount
	Status argus.Status
	Summy  [argus.MAXSTATUS][]string
}

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
