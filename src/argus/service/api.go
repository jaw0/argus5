// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Sep-15 22:38 (EDT)
// Function:

package service

import (
	"sort"

	"argus/api"
	"argus/darp"
	"argus/monel"
)

func init() {
	api.Add(true, "darp_list", apiDarpList)
}

func apiDarpList(ctx *api.Context) {

	all := make(map[string]bool)
	tag := ctx.Args["tag"]

	// find all services that match the tag
	lock.RLock()
	for uid, s := range allService {
		if darp.IncludesTag(s.Cf.DARP_Tags, tag) {
			all[uid] = true
			gatherParents(all, s.mon)
		}
	}
	lock.RUnlock()

	// flatten + sort
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
