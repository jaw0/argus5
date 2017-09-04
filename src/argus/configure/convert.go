// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Sep-03 19:02 (EDT)
// Function:

package configure

import (
	"strings"

	"argus/argus"
	"argus/darp"
)

func statusValue(name string) argus.Status {

	switch strings.ToLower(name) {
	case "clear":
		return argus.CLEAR
	case "warning":
		return argus.WARNING
	case "minor":
		return argus.MINOR
	case "major":
		return argus.MAJOR
	case "critical":
		return argus.CRITICAL

	}
	return argus.UNKNOWN
}

func gravityValue(name string) darp.Gravity {

	switch strings.ToLower(name) {
	case "up":
		return darp.GRAV_UP
	case "down":
		return darp.GRAV_DN
	case "vote":
		return darp.GRAV_VOTE
	case "ietf":
		return darp.GRAV_IETF
	case "self":
		return darp.GRAV_SELF
	}

	return darp.GRAV_UP
}
