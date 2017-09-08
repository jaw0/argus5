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

var ScheduleAlwaysYes = &Schedule{[]scheduleItem{{-1, 0, 0, "yes"}}}

func ScheduleAlways(val string) *Schedule {
	return &Schedule{[]scheduleItem{{-1, 0, 0, val}}}
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

func (s *Schedule) Append(dow int, start int, end int, value string) {

	s.sched = append(s.sched, scheduleItem{dow, start, end, value})
}

/*
 day start - end => value

*/
