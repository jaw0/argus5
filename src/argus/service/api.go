// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Sep-15 22:38 (EDT)
// Function:

package service

import (
	"fmt"
	"sort"
	"strconv"

	"argus.domain/argus/api"
	"argus.domain/argus/argus"
	"argus.domain/argus/darp"
	"argus.domain/argus/monel"
)

func init() {
	api.Add(true, "darp_list", apiDarpList)
	api.Add(true, "update", apiSetResultFor)
	api.Add(true, "hwab_reset", apiHwabReset)
}

// so slave can configure itself
func apiDarpList(ctx *api.Context) {

	all := make(map[string]bool)
	tag := ctx.Args["tag"]

	// find all services that match the tag...
	lock.RLock()
	for uid, s := range allService {
		if darp.IncludesTag(s.Cf.DARP_Tags, tag) {
			all[uid] = true
			// ...and their parents
			gatherParents(all, s.mon)
		}
	}
	lock.RUnlock()

	// flatten + sort
	// sorting puts objects in dfs order
	// - so parents get configured before their children
	i := 0
	list := make([]string, len(all))
	for k := range all {
		list[i] = k
		i++
	}

	sort.Strings(list)

	// send results
	ctx.SendOK()
	for _, uid := range list {
		ctx.Send(uid + "\n")
	}
	ctx.SendFinal()
}

// slave is sending status update
func apiSetResultFor(ctx *api.Context) {

	uid := ctx.Args["obj"]
	sts, _ := strconv.Atoi(ctx.Args["status"])
	status := argus.Status(sts)
	result := ctx.Args["result"]
	reason := ctx.Args["reason"]

	obj := Find(uid)
	if obj == nil {
		ctx.Send404()
		return
	}
	obj.SetResultFor(ctx.User, status, result, reason)
	ctx.SendOKFinal()
}

func gatherParents(all map[string]bool, m *monel.M) {

	if len(m.Parent) == 0 {
		return
	}

	p := m.Parent[0]
	uid := p.Cf.Unique

	if all[uid] {
		// already have it
		return
	}

	all[uid] = true
	// and upwards
	gatherParents(all, p)
}

func apiHwabReset(ctx *api.Context) {

	uid := ctx.Args["obj"]

	obj := Find(uid)
	if obj == nil {
		ctx.Send404()
		return
	}

	if obj.p.Hwab == nil {
		ctx.SendResponseFinal(404, "HWAB not enabled")
		return
	}

	obj.p.Hwab.Reset()
	ctx.SendOKFinal()
}

// ################################################################

func (s *Service) Dump(dx argus.Dumper) {

	dx.Dump("service/Lasttest", fmt.Sprintf("%d", s.Lasttest))
	dx.Dump("service/Started", fmt.Sprintf("%d", s.Started))
	dx.Dump("service/Elapsed", fmt.Sprintf("%.4f", float32(s.Elapsed)/1e9))
	dx.Dump("service/Tries", fmt.Sprintf("%d", s.Tries))
	argus.Dump(dx, "service", &s.p)
	argus.Dump(dx, "service/CF", &s.Cf)

	cm := s.check.DumpInfo()
	for pre, d := range cm {
		argus.Dump(dx, pre, d)
	}
}
