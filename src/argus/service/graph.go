// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Oct-01 23:07 (EDT)
// Function: send/recv graph data over darp

package service

import (
	"expvar"
	"fmt"

	"argus/api"
	"argus/argus"
	"argus/clock"
	"argus/configure"
	"argus/darp"
	"argus/graph"
	"argus/web"
)

const (
	graphMinTime = 120
	graphMaxSend = 128
)

var graphIsLocal = true
var darpGraphChan = make(chan string, 1024)
var darpGraphQueueLen = expvar.NewInt("darpgraphqueue")
var darpGraphQueueDrop = expvar.NewInt("darpgraphdrops")

func init() {
	api.Add(true, "graphdata", apiAddGraphData)
	web.Add(web.PRIVATE, "/api/graphd", webGraphJson)
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

	now := clock.Unix()
	if s.p.Lastgraph+graphMinTime > now {
		return
	}
	s.p.Lastgraph = now

	var yn, dn float64

	if s.p.Hwab != nil {
		yn = float64(s.p.Hwab.yn)
		dn = float64(s.p.Hwab.dn)
	}

	if s.mon.Cf.Graph {
		// darpid = "" for backwards compat
		s.Debug("graph")

		if graphIsLocal {
			graph.Add(s.mon.Pathname("", ""), now, s.mon.P.OvStatus, val, yn, dn)
		} else {
			darpGraphAdd(s.mon.Pathname("", ""), now, s.mon.P.OvStatus, val, yn, dn)
		}
	}
}

// ################################################################

func darpGraphAdd(file string, when int64, status argus.Status, val, yn, dn float64) {

	l := fmt.Sprintf("%s %d %d %f %f %f", file, when, status, val, yn, dn)

	darpGraphQueueLen.Set(int64(len(darpGraphChan)))

	// drop if queue full
	select {
	case darpGraphChan <- l:
	default:
		darpGraphQueueDrop.Add(1)
	}
}

func darpGraphWorker() {

	for {
		select {
		case l := <-darpGraphChan:
			darpGraphGather(l)
		}
	}
}

// bundle up a whole batch and send together
func darpGraphGather(l string) {

	m := make(map[string]string)
	m["g0"] = l
	n := 1

	for {
		select {
		case l := <-darpGraphChan:
			m[fmt.Sprintf("g%d", n)] = l
			n++
		default:
			break
		}
		if n >= graphMaxSend {
			break
		}
	}

	darp.TellMyMasters("graphdata", m)
}

// ################################################################

func apiAddGraphData(ctx *api.Context) {

	n := 0

	for {
		l := ctx.Args[fmt.Sprintf("g%d", n)]
		if l == "" {
			break
		}

		var file string
		var when int64
		var status int
		var val, yn, dn float64
		fmt.Sscan(l, &file, &when, &status, &val, &yn, &dn)

		graph.Add(file, when, argus.Status(status), val, yn, dn)
	}

	ctx.SendOKFinal()
}

// ################################################################

func webGraphJson(ctx *web.Context) {

	// obj, since, which, width

}
