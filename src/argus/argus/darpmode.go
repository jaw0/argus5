// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Sep-06 16:28 (EDT)
// Function:

package argus

import (
	"strings"
)

type DarpMode int

const (
	MODE_NONE        DarpMode = 0
	MODE_FAILOVER    DarpMode = 1
	MODE_DISTRIBUTED DarpMode = 2
	MODE_REDUNDANT   DarpMode = 3
)

var darpModeName = []string{
	"none", "failover", "distributed", "redundant",
}

func (d DarpMode) String() string {
	if d < MODE_NONE || d > MODE_REDUNDANT {
		return "invalid"
	}
	return darpModeName[int(d)]
}

func DarpModeValue(name string) DarpMode {

	switch strings.ToLower(name) {
	case "none":
		return MODE_NONE
	case "failover":
		return MODE_FAILOVER
	case "distributed", "distrib":
		return MODE_DISTRIBUTED
	case "redundant", "redund":
		return MODE_REDUNDANT
	}

	return MODE_NONE
}
