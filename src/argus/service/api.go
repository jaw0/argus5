// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Sep-15 22:38 (EDT)
// Function:

package service

import (
	"fmt"
	"reflect"
	"sort"

	"argus/api"
	"argus/darp"
	"argus/monel"
)

func init() {
	api.Add(true, "darp_list", apiDarpList)
	api.Add(true, "dump", apiDump)
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

// debugging dump
func apiDump(ctx *api.Context) {

	uid := ctx.Args["obj"]
	s := Find(uid)

	if s == nil {
		dl.Debug("not found: %s", uid)
		ctx.Send404()
		return
	}

	ctx.SendOK()
	s.mon.Lock.RLock()

	ctx.SendKVP("monel/Filename", s.mon.Filename)
	ctx.SendKVP("monel/DirName", s.mon.DirName)
	ctx.SendKVP("monel/Label", s.mon.Label)
	ctx.SendKVP("monel/Friendlyname", s.mon.Friendlyname)
	dumpStruct(ctx, s.mon.Cf, "monel/CF/")
	dumpStruct(ctx, s.mon.P, "monel/")

	ctx.SendKVP("service/Lasttest", fmt.Sprintf("%d", s.Lasttest))
	ctx.SendKVP("service/Started", fmt.Sprintf("%d", s.Started))
	ctx.SendKVP("service/Tries", fmt.Sprintf("%d", s.Tries))
	dumpStruct(ctx, s.Cf, "service/CF/")
	dumpStruct(ctx, s.p, "service/")

	cm := s.check.DumpInfo()
	for pre, d := range cm {
		dumpStruct(ctx, d, pre)
	}

	s.mon.Lock.RUnlock()
	ctx.SendFinal()

}

func dumpStruct(ctx *api.Context, obj interface{}, prefix string) {

	var val = reflect.ValueOf(obj)

	for i := 0; i < val.NumField(); i++ {

		t := val.Type().Field(i)
		v := val.Field(i)

		if t.Name == "Stats" || t.Name == "Log" || t.Name == "HWAB" {
			// too big, skip
			continue
		}

		name := prefix + t.Name

		ctx.SendKVP(name, fmt.Sprintf("%v", v))
	}
}
