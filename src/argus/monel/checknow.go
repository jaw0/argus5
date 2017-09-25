// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Sep-23 10:18 (EDT)
// Function: check now

package monel

import (
	"encoding/json"

	"argus/api"
	"argus/argus"
	"argus/web"
)

func init() {
	web.Add(web.WRITE, "/api/checknow", webCheckNow)
	api.Add(true, "checknow", apiCheckNow)
}

func webCheckNow(ctx *web.Context) {

	m, creds := webObjUserCheck(ctx)
	if m == nil {
		return
	}

	if !argus.ACLPermitsUser(m.Cf.ACL_CheckNow, creds) {
		dl.Debug("denied")
		ctx.W.WriteHeader(403)
		return
	}

	m.checkNow()

	d := m.newWebMetaResponse()
	js, _ := json.MarshalIndent(d, "", "  ")
	ctx.W.Header().Set("Content-Type", "application/json; charset=utf-8")
	ctx.W.Write(js)
}

func apiCheckNow(ctx *api.Context) {

	uid := ctx.Args["obj"]
	m := Find(uid)

	if m == nil {
		ctx.Send404()
		return
	}

	m.checkNow()
	ctx.SendOKFinal()
}

func (m *M) checkNow() {

	m.Lock.RLock()
	childs := m.Me.Children()
	m.Lock.RUnlock()

	for _, c := range childs {
		c.checkNow()
	}

	m.Me.CheckNow()
}
