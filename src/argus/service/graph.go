// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Oct-01 23:07 (EDT)
// Function: send/recv graph data over darp

package service

import (
	"fmt"
	"strconv"

	"argus/api"
	"argus/argus"
	"argus/clock"
	"argus/configure"
	"argus/darp"
	"argus/graph"
)

var graphIsLocal = true
var darpGraphChan = make(chan string, 1024)

func init() {
	api.Add(true, "graphdata", apiAddGraphData)
}

func GraphConfig(cf *configure.CF) {

	// configure params
	// darp or not?
	// start worker

	if !graphIsLocal {
		go darpGraphWorker()
	}
}

func (s *Service) recordMyGraphData(val float64) {

	dl.Verbose("record graph")

	var yn, dn float64

	if s.p.Hwab != nil {
		yn = float64(s.p.Hwab.yn)
		dn = float64(s.p.Hwab.dn)
	}

	if s.mon.Cf.Graph {
		// darpid = "" for backwards compat
		s.Debug("graph")

		if graphIsLocal {
			graph.Add(s.mon.Pathname("", ""), clock.Unix(), s.mon.P.OvStatus, val, yn, dn)
		} else {
			darpGraphAdd(s.mon.Pathname("", ""), clock.Unix(), s.mon.P.OvStatus, val, yn, dn)
		}
	}
}

// ################################################################

func darpGraphAdd(file string, when int64, status argus.Status, val, yn, dn float64) {

	l := fmt.Sprintf("%s %d %d %f %f %f", file, when, status, val, yn, dn)

	// drop if queue full
	select {
	case darpGraphChan <- l:
	default:
		break
	}
}

func darpGraphWorker() {

	// XXX
	return
	for {
		darp.TellMyMasters("graphdata", nil)
	}
}

func apiAddGraphData(ctx *api.Context) {

	uid := ctx.Args["obj"]
	obj := Find(uid)
	if obj == nil {
		ctx.Send404()
		return
	}

	// XXX
	sts, _ := strconv.Atoi(ctx.Args["status"])
	status := argus.Status(sts)
	when, _ := strconv.ParseInt(ctx.Args["when"], 10, 64)
	value, _ := strconv.ParseFloat(ctx.Args["value"], 32)
	yn, _ := strconv.ParseFloat(ctx.Args["yn"], 32)
	dn, _ := strconv.ParseFloat(ctx.Args["dn"], 32)

	graph.Add(obj.mon.Pathname(ctx.User+":", ""), when, status, value, yn, dn)
	ctx.SendOKFinal()
}
