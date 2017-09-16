// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Sep-01 21:05 (EDT)
// Function: schedules

package argus

import (
	"time"
)

type ScheduleItem struct {
	Dow   int // -1 is all
	Start int // hhmm
	End   int // hhmm
	Val   string
}

type Schedule struct {
	Sched []ScheduleItem
}

var ScheduleAlwaysYes = &Schedule{[]ScheduleItem{{-1, 0, 2400, "yes"}}}

func ScheduleAlways(val string) *Schedule {
	return &Schedule{[]ScheduleItem{{-1, 0, 2400, val}}}
}

func (s *Schedule) ResultNow() string {

	now := time.Now()
	dow := int(now.Weekday())
	hrs, min, _ := now.Clock()
	tim := hrs*100 + min

	for i, _ := range s.Sched {
		r := &s.Sched[i]

		if r.Dow != -1 && r.Dow != dow {
			continue
		}

		if tim < r.Start {
			continue
		}
		if tim >= r.End {
			continue
		}

		return r.Val
	}
	return ""
}

func (s *Schedule) PermitNow() bool {
	return CheckBool(s.ResultNow())
}

func (s *Schedule) Append(dow int, start int, end int, value string) {

	s.Sched = append(s.Sched, ScheduleItem{dow, start, end, value})
}

/*
 day start - end => value

*/
