// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Sep-08 19:42 (EDT)
// Function: creation

package notify

import (
	"strings"

	"argus.domain/argus/argus"
	"argus.domain/argus/clock"
)

func New(ncf *NewConf, mon Remover) *N {

	if ncf.OvStatus > argus.CRITICAL {
		return nil
	}

	n := &N{
		cf:  ncf.Conf,
		mon: mon,
		p: Persist{
			IdNo:         nextIdNo(),
			Created:      clock.Unix(),
			IsActive:     true,
			Unique:       ncf.Unique,
			ShortName:    ncf.ShortName,
			FriendlyName: ncf.FriendlyName,
			Reason:       ncf.Reason,
			Result:       ncf.Result,
			OvStatus:     ncf.OvStatus,
			PrevOv:       ncf.PrevOv,
			CurrOv:       ncf.OvStatus,
			Status:       make(map[string]string),
		},
	}

	n.determineMessage(ncf)
	n.determineSendTo()
	dl.Debug("new notification %d - %s [%s] => %#v", n.p.IdNo, n.p.Unique, n.p.Message, n.p.SendTo)

	n.p.MessageFmted = n.expand(globalDefaults.Message_Fmt, n.p.Message, nil)

	if len(n.p.SendTo) == 0 {
		dl.Debug("nowhere to send! discarding notification")
		return nil
	}

	n.log("system", "created")
	n.Save()
	notechan <- n
	return n
}

func (n *N) determineMessage(ncf *NewConf) {

	st := n.p.OvStatus

	if st == argus.CLEAR {
		if n.cf.MessageUp != "" {
			n.p.Message = n.expand(n.cf.MessageUp, "", nil)
			return
		}
	} else {
		if n.cf.MessageDn != "" {
			n.p.Message = n.expand(n.cf.MessageDn, "", nil)
			return
		}
	}

	bmsg := ""
	switch globalDefaults.Message_Style {
	case "long":
		bmsg = ncf.Unique
	case "short":
		bmsg = ncf.ShortName
	default:
		bmsg = ncf.FriendlyName
	}

	if st == argus.CLEAR {
		n.p.Message = bmsg + " is UP"
	} else {
		n.p.Message = bmsg + " is DOWN/" + st.String()
	}
}

func (n *N) determineSendTo() {

	ns := n.cf.Notify[int(n.p.OvStatus)]
	if ns == nil {
		ns = n.cf.Notify[int(argus.UNKNOWN)]
	}

	dst := []string{}

	// current notify value
	if ns != nil {
		nv := strings.Fields(ns.ResultNow(""))
		if len(nv) > 0 {
			dst = append(dst, nv...)
		}
	}

	// notify also?
	if n.cf.NotifyAlso != "" {
		nv := strings.Fields(n.cf.NotifyAlso)
		dst = append(dst, nv...)
	}

	if len(dst) > 0 {
		n.p.SendTo = []SendDat{{When: 0, Dst: dst}}
	}

	// build escalation table
	// should be: N dst dst ; N dst dst ; ...
	// where N is a timespec [defaults to minutes]
	esc := n.cf.Escalate[int(n.p.OvStatus)]
	if esc == "" {
		esc = n.cf.Escalate[int(argus.UNKNOWN)]
	}

	if esc == "" {
		return
	}

	escl := strings.Split(esc, ";")
	hasErr := false
	for _, e := range escl {
		f := strings.Fields(e)
		if len(f) < 2 {
			hasErr = true
			continue
		}
		t, err := argus.Timespec(f[0], 60)
		if err != nil {
			hasErr = true
			continue
		}

		n.p.SendTo = append(n.p.SendTo, SendDat{When: t, Dst: f[1:]})
	}

	if hasErr {
		dl.Problem("invalid escalate '%s'", esc)
	}
}
