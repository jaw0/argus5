// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Sep-23 18:16 (EDT)
// Function: web interface

package notify

import (
	"encoding/json"
	"strconv"
	"strings"

	"argus/argus"
	"argus/web"
)

func init() {
	web.Add(web.PRIVATE, "/api/notify", webJson)
}

func webGetNotifyCreds(ctx *web.Context) (*N, []string) {

	if ctx.User == nil {
		ctx.W.WriteHeader(403)
		return nil, nil
	}

	creds := strings.Fields(ctx.User.Groups)

	if !argus.ACLPermitsUser(globalDefaults.ACL_NotifyDetail, creds) {
		ctx.W.WriteHeader(403)
		return nil, nil
	}

	idno, _ := strconv.ParseInt(ctx.Get("idno"), 10, 32)
	lock.RLock()
	n := byid[int(idno)]
	lock.RUnlock()

	if n == nil {
		ctx.W.WriteHeader(404)
		return nil, nil
	}

	return n, creds
}

func webJson(ctx *web.Context) {

	n, creds := webGetNotifyCreds(ctx)
	if n == nil {
		return
	}

	canAck := argus.ACLPermitsUser(globalDefaults.ACL_NotifyAck, creds)

	n.lock.RLock()
	d := struct {
		N      *Persist
		CanAck bool
	}{&n.p, canAck}
	js, _ := json.MarshalIndent(d, "", "  ")
	n.lock.RUnlock()

	ctx.W.Header().Set("Content-Type", "application/json; charset=utf-8")
	ctx.W.Write(js)

}

/*
func webList(ctx *web.Context) {

	n, creds := webGetNotifyCreds(ctx)
	if n == nil {
		return
	}

	if !argus.ACLPermitsUser(globalDefaults.ACL_NotifyList, creds) {
		ctx.W.WriteHeader(403)
		return nil, nil
	}

	canAck := argus.ACLPermitsUser(globalDefaults.ACL_NotifyAck, creds)

}
*/
