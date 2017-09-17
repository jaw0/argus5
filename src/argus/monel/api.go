// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Sep-16 00:20 (EDT)
// Function:

package monel

import (
	"encoding/json"

	"argus/api"
	"argus/argus"
)

func init() {
	api.Add(true, "setdebug", apiDebug)
	api.Add(true, "getconfig", apiGetConfig)
}

func apiDebug(ctx *api.Context) {

	uid := ctx.Args["obj"]
	enable := argus.CheckBool(ctx.Args["enabled"])
	m := Find(uid)

	if m == nil {
		ctx.Send404()
		return
	}

	m.Cf.Debug = enable
	ctx.SendOKFinal()
}

func apiGetConfig(ctx *api.Context) {

	uid := ctx.Args["obj"]
	m := Find(uid)

	if m == nil {
		dl.Debug("not found: %s", uid)
		ctx.Send404()
		return
	}

	cf := m.ConfCF

	ctx.SendOK()
	ctx.SendKVP("Name", cf.Name)
	ctx.SendKVP("Type", cf.Type)
	ctx.SendKVP("Unique", m.Cf.Unique)

	if len(m.Parent) > 0 {
		p := m.Parent[0].Unique()
		ctx.SendKVP("Parent", p)
	}

	for k, v := range cf.Param {
		if !v.Used {
			continue
		}
		prefix := "_"

		if !v.Inherit {
			prefix = "!"
		}

		switch val := v.Value.(type) {
		case string:
			ctx.SendKVP(prefix+k, val)
		case *argus.Schedule:
			ctx.SendKVP("*"+prefix+k, schedule2Json(val))
		default:
			dl.Bug("cannot serialize %v", val)
		}

	}

	ctx.SendFinal()
}

func schedule2Json(sch *argus.Schedule) string {

	js, _ := json.Marshal(sch)
	return string(js)
}
