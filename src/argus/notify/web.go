// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Sep-23 18:16 (EDT)
// Function: web interface

package notify

import (
	"encoding/json"
	"sort"
	"strconv"
	"strings"

	"argus/argus"
	"argus/web"
)

func init() {
	web.Add(web.PRIVATE, "/api/notify", webJson)
	web.Add(web.PRIVATE, "/api/listnotify", webList)
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

func webList(ctx *web.Context) {

	if ctx.User == nil {
		ctx.W.WriteHeader(403)
		return
	}

	creds := strings.Fields(ctx.User.Groups)

	if !argus.ACLPermitsUser(globalDefaults.ACL_NotifyList, creds) ||
		!argus.ACLPermitsUser(globalDefaults.ACL_NotifyDetail, creds) {
		ctx.W.WriteHeader(403)
		return
	}

	type export struct {
		IdNo     int
		Created  int64
		IsActive bool
		CanAck   bool
		OvStatus argus.Status
		Message  string
		Unique   string
	}

	canAck := argus.ACLPermitsUser(globalDefaults.ACL_NotifyAck, creds)

	lock.RLock()
	var all []*N
	for _, n := range byid {
		all = append(all, n)
	}
	lock.RUnlock()

	sort.Slice(all, func(i, j int) bool { return all[i].p.Created < all[j].p.Created })

	var res []export

	for _, n := range all {
		n.lock.RLock()
		res = append(res, export{n.p.IdNo, n.p.Created * SECSNANO, n.p.IsActive,
			canAck, n.p.OvStatus, n.p.Message, n.p.Unique})
		n.lock.RUnlock()
	}

	js, _ := json.MarshalIndent(res, "", "  ")
	ctx.W.Header().Set("Content-Type", "application/json; charset=utf-8")
	ctx.W.Write(js)

}
