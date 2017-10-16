// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Sep-17 14:25 (EDT)
// Function: emit json for web

package monel

import (
	"encoding/json"
	"sort"
	"strconv"
	"strings"

	"argus/argus"
	"argus/clock"
	"argus/darp"
	"argus/graph"
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
	web.Add(web.PRIVATE, "/api/graph", webGraphInfo)
	web.Add(web.PRIVATE, "/api/graphd", webGraphDJson)
	web.Add(web.PRIVATE, "/api/listdown", webDownList)
	web.Add(web.PRIVATE, "/api/listov", webOverrideList)
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
		mond["IsService"] = false // to simplify web view code

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

	md["alarm"] = m.P.Alarm
	md["sirentime"] = m.P.SirenTime
	md["sirenhush"] = ctx.Hush > m.P.SirenTime
	md["webtime"] = m.WebTime
	md["unacked"] = notify.NumActive()
	md["hasErrors"] = argus.HasErrors()
	md["hasWarns"] = argus.HasWarnings()

	m.Lock.RUnlock()

	m.Me.WebMeta(md)
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

func (mm *M) childSummary(creds []string) *childSummary {

	m := mm.Me.Self() // alias redirect
	m.Lock.RLock()
	cs := &childSummary{
		Desc:   objectDescr{m.Cf.Unique, mm.Cf.Uname, mm.Cf.Label},
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

	for _, cx := range childs {
		cc := cx.Me.Self() // alias redirect

		if cc.Cf.Hidden || !argus.ACLPermitsUser(cc.Cf.ACL_Page, creds) {
			continue
		}

		cc.Lock.RLock()
		st := cc.P.OvStatus
		desc := &objectDescr{cc.Cf.Unique, cx.Cf.Uname, cx.Cf.Label}

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

	md["name"] = m.Name
	md["uname"] = m.Cf.Uname
	md["unique"] = m.Cf.Unique
	md["label"] = m.Cf.Label
	md["friendly"] = m.Cf.Friendlyname
	md["note"] = m.Cf.Note
	md["info"] = m.Cf.Info
	md["details"] = m.Cf.Details
	md["comment"] = m.Cf.Comment
	md["graph"] = m.Cf.Graph

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

func webGraphDJson(ctx *web.Context) {

	// obj, tag, since, which, width

	m, _ := webObjUserCheck(ctx)
	if m == nil {
		return
	}

	if !m.Cf.Graph {
		ctx.W.WriteHeader(404)
		return
	}

	d := m.newWebMetaResponse(ctx)

	since, _ := strconv.ParseInt(ctx.Get("since"), 10, 64)
	tag := ctx.Get("tag")
	which := ctx.Get("which")
	width, _ := strconv.ParseInt(ctx.Get("width"), 10, 64)

	if tag == "local" || tag == darp.MyId {
		tag = ""
	}

	d["data"] = graph.Get(m.Pathname(tag, ""), which, since, int(width))

	js, _ := json.MarshalIndent(d, "", "  ")
	ctx.W.Header().Set("Content-Type", "application/json; charset=utf-8")
	ctx.W.Write(js)
}

func webGraphInfo(ctx *web.Context) {

	m, _ := webObjUserCheck(ctx)
	if m == nil {
		return
	}

	d := m.newWebMetaResponse(ctx)

	if !m.Cf.Graph {
		ctx.W.WriteHeader(404)
		return
	}

	gi := struct {
		Title  string
		YLabel string
		MyId   string
		List   []interface{}
	}{m.Cf.Title, m.Cf.YLabel, darp.MyId, nil}

	gi.List = m.graphList("", gi.List)
	d["graph"] = gi

	js, _ := json.MarshalIndent(d, "", "  ")
	ctx.W.Header().Set("Content-Type", "application/json; charset=utf-8")
	ctx.W.Write(js)
}

func (m *M) graphList(label string, gl []interface{}) []interface{} {

	m.Lock.RLock()
	childs := m.Me.Children()
	m.Lock.RUnlock()

	gl = m.Me.GraphList(label, gl)

	for _, c := range childs {
		if c.Cf.Hidden {
			continue
		}

		clabel := c.Cf.GraphLabel
		if clabel == "" {
			clabel = c.Cf.Label
		}
		if label != "" {
			clabel = label + ":" + clabel
		}

		gl = c.graphList(clabel, gl)
	}

	return gl
}

// ################################################################

func webDownList(ctx *web.Context) {

	webList(ctx, isdown, true)
}
func webOverrideList(ctx *web.Context) {

	webList(ctx, inoverride, false)
}

func webList(ctx *web.Context, list map[string]*M, interesting bool) {

	d := make(map[string]interface{})

	d["unacked"] = notify.NumActive()
	d["hasErrors"] = argus.HasErrors()
	d["hasWarns"] = argus.HasWarnings()

	if ctx.User == nil {
		ctx.W.WriteHeader(403)
		return
	}

	creds := strings.Fields(ctx.User.Groups)

	type schwartz struct {
		m *M
		T int64
	}

	var all []schwartz
	lock.RLock()
	for _, m := range list {
		m.Lock.RLock()
		all = append(all, schwartz{m, m.P.TransTime})
		m.Lock.RUnlock()
	}
	lock.RUnlock()

	sort.Slice(all, func(i, j int) bool { return all[i].T > all[j].T })

	type export struct {
		Unique string
		Status argus.Status
	}
	var out []export

	for i, _ := range all {
		m := all[i].m
		if !argus.ACLPermitsUser(m.Cf.ACL_Page, creds) {
			continue
		}
		if m.Cf.Hidden {
			continue
		}
		if interesting && !m.Interesting {
			continue
		}

		m.Lock.RLock()
		st := m.P.OvStatus
		m.Lock.RUnlock()

		out = append(out, export{m.Cf.Unique, st})
	}

	d["list"] = out

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
