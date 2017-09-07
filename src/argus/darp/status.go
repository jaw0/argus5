// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Sep-01 00:35 (EDT)
// Function:

package darp

import (
	"argus/argus"
)

type Gravity int

const (
	GRAV_UP   Gravity = 0
	GRAV_DN   Gravity = 1
	GRAV_VOTE Gravity = 2
	GRAV_IETF Gravity = 3
	GRAV_SELF Gravity = 4
)

func AggrStatus(gravity Gravity, mystatus argus.Status, statuses map[string]argus.Status) argus.Status {

	if gravity == GRAV_SELF { // XXX - || i_am_slave || i_have_no_slaves
		return mystatus
	}

	var darps map[string]bool // XXX

	return CalcAggrStatus(gravity, mystatus, statuses, darps)
}

func CalcAggrStatus(gravity Gravity, mystatus argus.Status,
	statuses map[string]argus.Status, darps map[string]bool) argus.Status {

	var count [argus.MAXSTATUS + 1]int
	nstatus := 0

	for host, up := range darps {

		if up {
			count[statuses[host]]++
			nstatus++
		} else if gravity == GRAV_IETF {
			count[argus.UNKNOWN]++
		} else {
			// use most recent status
			count[statuses[host]]++
			nstatus++
		}
	}

	count[argus.UNKNOWN] += count[argus.OVERRIDE] + count[argus.DEPENDS]

	switch gravity {
	case GRAV_SELF:
		return mystatus
	case GRAV_UP:
		for i := argus.CLEAR; i <= argus.CRITICAL; i++ {
			if count[i] > 0 {
				return i
			}
		}
		return argus.CLEAR

	case GRAV_DN:
		for i := argus.CRITICAL; i >= argus.CLEAR; i-- {
			if count[i] > 0 {
				return i
			}
		}
		return argus.CLEAR

	case GRAV_VOTE, GRAV_IETF:
		lim := nstatus / 2
		cum := 0

		for i := argus.CLEAR; i <= argus.CRITICAL; i++ {
			cum += count[i]
			if cum >= lim {
				return i
			}
		}
		return argus.CLEAR
	}
	return argus.UNKNOWN
}
