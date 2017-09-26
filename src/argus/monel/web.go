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
	"argus/clock"
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
	Up       []*objectDescr
	Dn       []*objectDescr
	Ov       []*objectDescr
	Sev      argus.Status // max sev of Dn[]
}

const (
	WEBMAXLOG    = 100
	WEBMAXNOTIFY = 100
)

func init() {
	web.Add(web.PRIVATE, "/api/page", webJson)
	web.Add(web.WRITE, "/api/annotate", webAnnotate)
}

func webJson(ctx *web.Context) {

	since, _ := strconv.ParseInt(ctx.Get("since"), 10, 64)

	m, creds := webObjUserCheck(ctx)
	if m == nil {
		return
	}

	d := m.newWebMetaResponse(ctx)

	// only include these if something has changed
	if !m.webChangedSince(since) {
		d["unchanged"] = true
	} else {
		mond := make(map[string]interface{})
		deco := make(map[string]interface{})
		d["mon"] = mond
		d["deco"] = deco

		m.webDecor(creds, deco)
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

func (m *M) webMeta(ctx *web.Context, md map[string]interface{}) {

	m.Lock.RLock()
	defer m.Lock.RUnlock()

	md["alarm"] = m.P.Alarm
	md["sirentime"] = m.P.SirenTime
	md["sirenhush"] = ctx.Hush > m.P.SirenTime
	md["webtime"] = m.WebTime
	md["unacked"] = notify.NumActive()
	md["hasErrors"] = argus.HasErrors()
	md["hasWarns"] = argus.HasWarnings()
}

func (m *M) webJson(creds []string, md map[string]interface{}) {

	m.Lock.RLock()

	md["status"] = m.P.Status
	md["ovstatus"] = m.P.OvStatus
	md["transtime"] = m.P.TransTime
	md["override"] = m.P.Override
	md["annotation"] = m.P.Annotation
	md["reason"] = m.P.Reason
	md["stats"] = m.ExportStats()

	// reverse and truncate notifies, logs
	log := make([]*Log, 0, min(WEBMAXLOG, len(m.P.Log)))
	not := make([]*notify.ExportInfo, 0, min(WEBMAXNOTIFY, len(m.Notifies)))

	for i := len(m.Notifies) - 1; i >= 0; i-- {
		n := m.Notifies[i]
		not = append(not, n.WebExport())
		if len(not) >= WEBMAXNOTIFY {
			break
		}
	}

	for i := len(m.P.Log) - 1; i >= 0; i-- {
		log = append(log, m.P.Log[i])
		if len(log) >= WEBMAXLOG {
			break
		}
	}

	md["notify"] = not
	md["log"] = log

	childs := m.Me.Children()
	m.Lock.RUnlock()

	// children summary
	var childsum []*childSummary

	for _, c := range childs {
		if c.Cf.Hidden || !argus.ACLPermitsUser(c.Cf.ACL_Page, creds) {
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
		if cc.Cf.Hidden || !argus.ACLPermitsUser(cc.Cf.ACL_Page, creds) {
			continue
		}

		cc.Lock.RLock()
		st := cc.P.OvStatus
		desc := &objectDescr{cc.Cf.Unique, cc.Cf.Uname, cc.Cf.Label}

		switch st {
		case argus.OVERRIDE:
			cs.Ov = append(cs.Ov, desc)
		case argus.CLEAR, argus.UNKNOWN, argus.DEPENDS:
			cs.Up = append(cs.Up, desc)
		default:
			cs.Dn = append(cs.Dn, desc)
			if st > cs.Sev {
				cs.Sev = st
			}
		}
		cs.HasChild = true
		cc.Lock.RUnlock()
	}

	return cs
}

// gather the data needed for static labels, buttons, etc
func (m *M) webDecor(creds []string, md map[string]interface{}) {

	m.Lock.RLock()
	defer m.Lock.RUnlock()

	md["name"] = m.Cf.Uname
	md["unique"] = m.Cf.Unique
	md["label"] = m.Cf.Label
	md["friendly"] = m.Cf.Friendlyname
	md["note"] = m.Cf.Note
	md["info"] = m.Cf.Info
	md["details"] = m.Cf.Details
	md["comment"] = m.Cf.Comment

	md["canOverride"] = m.Cf.Overridable && argus.ACLPermitsUser(m.Cf.ACL_Override, creds)
	md["canAnnotate"] = argus.ACLPermitsUser(m.Cf.ACL_Annotate, creds)
	md["canCheckNow"] = argus.ACLPermitsUser(m.Cf.ACL_CheckNow, creds)

	var parent []objectDescr

	for _, p := range m.Parent {
		parent = append(parent, objectDescr{p.Cf.Unique, p.Cf.Uname, p.Cf.Label})
	}

	md["parent"] = parent
}

// ################################################################

func webAnnotate(ctx *web.Context) {

	m, creds := webObjUserCheck(ctx)
	if m == nil {
		return
	}

	if !argus.ACLPermitsUser(m.Cf.ACL_Annotate, creds) {
		dl.Debug("denied")
		ctx.W.WriteHeader(403)
		return
	}

	d := m.newWebMetaResponse(ctx)

	text := ctx.Get("text")

	m.Lock.Lock()
	m.P.Annotation = text
	d["annotation"] = m.P.Annotation
	m.WebTime = clock.Nano()
	m.Lock.Unlock()

	if text == "" {
		m.Loggit("ANNOTATION", "removed by "+ctx.User.Name)
	} else {
		m.Loggit("ANNOTATION", "added by "+ctx.User.Name)
	}

	js, _ := json.MarshalIndent(d, "", "  ")
	ctx.W.Header().Set("Content-Type", "application/json; charset=utf-8")
	ctx.W.Write(js)
}

// ################################################################

func (m *M) newWebMetaResponse(ctx *web.Context) map[string]interface{} {

	d := make(map[string]interface{})
	// general metadata - always include on every response
	m.webMeta(ctx, d)
	return d
}

func webObjUserCheck(ctx *web.Context) (*M, []string) {

	obj := ctx.Get("obj")
	m := Find(obj)

	if m == nil {
		dl.Verbose("obj not found %s", obj)
		ctx.W.WriteHeader(404)
		return nil, nil
	}

	var creds []string
	if ctx.User != nil {
		creds = strings.Fields(ctx.User.Groups)
	}

	if !argus.ACLPermitsUser(m.Cf.ACL_Page, creds) {
		dl.Debug("denied")
		ctx.W.WriteHeader(403)
		return nil, nil
	}

	return m, creds
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
