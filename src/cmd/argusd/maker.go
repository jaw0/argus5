// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Sep-15 22:01 (EDT)
// Function: interface for making (eg. via darp)

package main

import (
	"encoding/json"

	"argus/argus"
	"argus/configure"
	"argus/construct"
	"argus/monel"
)

type MakeIt struct{}

// data should be of the form produced by monel/api:getconfig
func (x *MakeIt) Make(m map[string]string) error {

	// find parent
	p := m["Parent"]
	var parent *monel.M
	var pcf *configure.CF

	if p != "" {
		parent = monel.Find(p)
	}
	if parent != nil {
		pcf = parent.ConfCF
	}

	// remove existing
	cm := monel.Find(m["Unique"])
	if cm != nil {
		cm.Recycle(true)
	}

	// build cf
	cf := configure.NewCF(m["Type"], m["Name"], pcf)

	for k, v := range m {
		if k == "" {
			continue
		}

		cfv := &configure.CFV{Value: v}

		if k[0] == '*' {
			// unserialize schedule
			cfv.Value = json2Schedule(v)
			k = k[1:]
		}

		if k[0] != '_' {
			continue
		}

		cf.Param[k[1:]] = cfv

	}

	// make it
	construct.Make(cf, parent)

	return nil
}

func json2Schedule(js string) *argus.Schedule {

	sch := &argus.Schedule{}
	json.Unmarshal([]byte(js), sch)
	return sch
}
