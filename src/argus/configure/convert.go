// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Sep-03 19:02 (EDT)
// Function:

package configure

import (
	"strings"

	"argus/argus"
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

func gravityValue(name string) argus.Gravity {

	switch strings.ToLower(name) {
	case "up":
		return argus.GRAV_UP
	case "down":
		return argus.GRAV_DN
	case "vote":
		return argus.GRAV_VOTE
	case "ietf":
		return argus.GRAV_IETF
	case "self":
		return argus.GRAV_SELF
	}

	return argus.GRAV_UP
}
