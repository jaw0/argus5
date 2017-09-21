// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Sep-01 00:35 (EDT)
// Function:

package darp

import (
	"argus/argus"
)

func AggrStatus(gravity argus.Gravity, mystatus argus.Status, statuses map[string]argus.Status) argus.Status {

	dl.Debug("grav %s; %s; %v", gravity, mystatus, statuses)
	if gravity == argus.GRAV_SELF || !iHaveSlaves || !IsEnabled {
		return mystatus
	}

	darps := GetStatuses()

	return CalcAggrStatus(gravity, mystatus, statuses, darps)
}

func CalcAggrStatus(gravity argus.Gravity, mystatus argus.Status,
	statuses map[string]argus.Status, darps map[string]bool) argus.Status {

	var count [argus.MAXSTATUS + 1]int
	nstatus := 0

	for host, up := range darps {
		if up {
			count[statuses[host]]++
			nstatus++
		} else if gravity == argus.GRAV_IETF {
			count[argus.UNKNOWN]++
		} else {
			// use most recent status
			count[statuses[host]]++
			nstatus++
		}
	}

	dl.Debug("counts %v", count)
	count[argus.UNKNOWN] += count[argus.OVERRIDE] + count[argus.DEPENDS]

	switch gravity {
	case argus.GRAV_SELF:
		return mystatus
	case argus.GRAV_UP:
		for i := argus.CLEAR; i <= argus.CRITICAL; i++ {
			if count[i] > 0 {
				return i
			}
		}
		return argus.CLEAR

	case argus.GRAV_DN:
		for i := argus.CRITICAL; i >= argus.CLEAR; i-- {
			if count[i] > 0 {
				return i
			}
		}
		return argus.CLEAR

	case argus.GRAV_VOTE, argus.GRAV_IETF:
		lim := (nstatus + 1) / 2
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
