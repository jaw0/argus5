// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Sep-09 11:35 (EDT)
// Function: ack notifications

package notify

import (
	"encoding/json"
	"strconv"

	"argus.domain/argus/api"
	"argus.domain/argus/argus"
	"argus.domain/argus/web"
)

func init() {
	web.Add(web.WRITE, "/api/notifyack", webAck)
	api.Add(true, "notifyack", apiAck)
}

// package lock is already held
func (n *N) maybeAck() {

	n.lock.Lock()
	defer n.lock.Unlock()

	ost := n.p.OvStatus
	nst := n.p.CurrOv

	if nst == argus.CLEAR && int(ost) < len(n.cf.AckOnUp) {
		if n.cf.AckOnUp[int(ost)] || n.cf.AckOnUp[int(argus.UNKNOWN)] {
			dl.Debug("%d - auto ack on up", n.p.IdNo)
			n.ack("auto")
			return
		}
	}

	if nst > ost && int(ost) < len(n.cf.Ack_On_Worse) {
		// got worse
		if n.cf.Ack_On_Worse[int(ost)] || n.cf.Ack_On_Worse[int(argus.UNKNOWN)] {
			dl.Debug("%d - auto ack on worse", n.p.IdNo)
			n.ack("auto")
			return
		}
	}
	if nst < ost && int(ost) < len(n.cf.Ack_On_Better) {
		// got better
		if n.cf.Ack_On_Better[int(ost)] || n.cf.Ack_On_Better[int(argus.UNKNOWN)] {
			dl.Debug("%d - auto ack on better", n.p.IdNo)
			n.ack("auto")
			return
		}
	}
}

// package lock is already held
func (n *N) maybeAutoAck() {

	n.lock.Lock()
	defer n.lock.Unlock()

	st := n.p.OvStatus
	if int(st) >= len(n.cf.AutoAck) {
		return
	}

	if n.cf.AutoAck[st] || n.cf.AutoAck[int(argus.UNKNOWN)] {
		n.ack("auto")
	}
}

// package lock + notify are already held
func (n *N) ack(who string) {

	n.p.IsActive = false
	delete(actives, n.p.IdNo)
	NActive.Set(int64(len(actives)))
	n.Save()
	n.log(who, "acked")

	for dst, _ := range n.p.Status {
		n.p.Status[dst] = "acked"
	}
}

func webAck(ctx *web.Context) {

	n, creds := webGetNotifyCreds(ctx)
	if n == nil {
		return
	}

	if !argus.ACLPermitsUser(globalDefaults.ACL_NotifyAck, creds) {
		dl.Debug("denied")
		ctx.W.WriteHeader(403)
		return
	}

	n.lock.Lock()
	n.ack(ctx.User.Name)
	js, _ := json.MarshalIndent(n.p, "", "  ")
	n.lock.Unlock()

	ctx.W.Header().Set("Content-Type", "application/json; charset=utf-8")
	ctx.W.Write(js)
}

func apiAck(ctx *api.Context) {

	idno, _ := strconv.ParseInt(ctx.Args["idno"], 10, 32)
	lock.RLock()
	n := byid[int(idno)]
	lock.RUnlock()

	if n == nil {
		ctx.Send404()
		return
	}

	n.lock.Lock()
	n.ack("api")
	n.lock.Unlock()
	ctx.SendOKFinal()
}
