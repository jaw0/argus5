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

	"argus.domain/argus/argus"
	"argus.domain/argus/web"
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

	type schwartz struct {
		n *N
		s int64
	}

	lock.RLock()
	var all []schwartz
	for _, n := range byid {
		n.lock.RLock()
		s := n.p.Created
		if n.p.IsActive {
			s *= 2
		}
		n.lock.RUnlock()
		all = append(all, schwartz{n, s})
	}
	lock.RUnlock()

	// order by IsActive, Created
	sort.Slice(all, func(i, j int) bool { return all[j].s < all[i].s })

	d := make(map[string]interface{})

	d["unacked"] = NumActive()
	d["hasErrors"] = argus.HasErrors()
	d["hasWarns"] = argus.HasWarnings()

	var res []export

	for i, _ := range all {
		n := all[i].n
		n.lock.RLock()
		res = append(res, export{n.p.IdNo, n.p.Created * SECSNANO, n.p.IsActive,
			canAck, n.p.OvStatus, n.p.Message, n.p.Unique})
		n.lock.RUnlock()
	}

	d["list"] = res

	js, _ := json.MarshalIndent(d, "", "  ")
	ctx.W.Header().Set("Content-Type", "application/json; charset=utf-8")
	ctx.W.Write(js)

}
