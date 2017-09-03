// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Sep-01 21:05 (EDT)
// Function: schedules

package argus

import (
	"time"
)

type scheduleItem struct {
	dow   int // -1 is all
	start int // hhmm
	end   int // hhmm
	val   string
}

type Schedule struct {
	sched []scheduleItem
}

func (s *Schedule) ResultNow() string {

	now := time.Now()
	dow := int(now.Weekday())
	hrs, min, _ := now.Clock()
	tim := hrs*100 + min

	for i, _ := range s.sched {
		r := &s.sched[i]

		if r.dow != -1 && r.dow != dow {
			continue
		}

		if tim < r.start {
			continue
		}
		if tim >= r.end {
			continue
		}

		return r.val
	}

	return ""
}

func (s *Schedule) PermitNow() bool {
	return CheckBool(s.ResultNow())
}

/*
 day start - end => value

*/
