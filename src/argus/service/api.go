// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Sep-15 22:38 (EDT)
// Function:

package service

import (
	"fmt"
	"sort"
	"strconv"

	"argus/api"
	"argus/argus"
	"argus/darp"
	"argus/monel"
)

func init() {
	api.Add(true, "darp_list", apiDarpList)
	api.Add(true, "update", apiSetResultFor)
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

// ################################################################

func (s *Service) Dump(ctx *api.Context) {

	ctx.SendKVP("service/Lasttest", fmt.Sprintf("%d", s.Lasttest))
	ctx.SendKVP("service/Started", fmt.Sprintf("%d", s.Started))
	ctx.SendKVP("service/Tries", fmt.Sprintf("%d", s.Tries))
	ctx.DumpStruct(&s.Cf, "service/CF/")
	ctx.DumpStruct(&s.p, "service/")

	cm := s.check.DumpInfo()
	for pre, d := range cm {
		dl.Verbose("pre: %s; %v", pre, d)
		ctx.DumpStruct(d, pre)
	}
}
