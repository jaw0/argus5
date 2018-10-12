// Copyright (c) 2018
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2018-Oct-12 13:47 (EDT)
// Function: debugging dump

package monel

import (
	"encoding/json"
	"fmt"

	"argus/argus"
	"argus/web"
)

func init() {
	web.Add(web.PUBLIC, "/api/dump", webDump)
}

func (m *M) dump(dx argus.Dumper) {

	m.Lock.RLock()

	var parent []string
	var children []string
	for _, p := range m.Parent {
		parent = append(parent, p.Cf.Unique)
	}
	for _, c := range m.Children {
		children = append(children, c.Cf.Unique)
	}

	dx.Dump("monel/Filename", m.Filename)
	dx.Dump("monel/DirName", m.DirName)
	dx.Dump("monel/Label", m.Label)
	dx.Dump("monel/Friendlyname", m.Friendlyname)
	dx.Dump("monel/Interesting", fmt.Sprintf("%v", m.Interesting))
	dx.Dump("monel/Parent", fmt.Sprintf("%v", parent))
	dx.Dump("monel/Children", fmt.Sprintf("%v", children))

	argus.Dump(dx, "monel/CF", &m.Cf)
	argus.Dump(dx, "monel/Notify/CF", m.NotifyCf)
	argus.Dump(dx, "monel", &m.P)

	m.Me.Dump(dx)
	m.Lock.RUnlock()
}

type webKVP struct {
	K string
	V string
}
type webDumpMap []webKVP

func (w *webDumpMap) Dump(key string, val string) {
	*w = append(*w, webKVP{key, val})
}

func webDump(ctx *web.Context) {

	m, creds := webObjUserCheck(ctx)
	if m == nil {
		return
	}

	if !argus.ACLPermitsUser(m.Cf.ACL_About, creds) {
		dl.Debug("denied")
		ctx.W.WriteHeader(403)
		return
	}

	// build debug info - array of {K, V}
	wx := &webDumpMap{}
	m.dump(wx)

	dump := struct {
		Dump interface{}
	}{wx}

	// map => js
	js, _ := json.MarshalIndent(dump, "", "  ")
	ctx.W.Header().Set("Content-Type", "application/json; charset=utf-8")
	ctx.W.Write(js)

}
