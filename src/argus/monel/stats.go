// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Sep-06 19:32 (EDT)
// Function: logs + stats

package monel

import (
	"time"

	"argus/argus"
	"argus/clock"
)

const SECSNANO = 1000000000

type statDat struct {
	Start   int64 // nanos
	Elapsed int   // secs
	Ndown   int
	TUp     int // secs
	TDn     int // secs
}
type Stats struct {
	Lastt   int64
	Status  argus.Status
	Daily   []statDat
	Monthly []statDat
	Yearly  []statDat
}

type Log struct {
	When     int64
	OvStatus argus.Status
	Tag      string
	Msg      string
}

const (
	MAXLOG  = 500
	MAXSTAT = 14
)

func (m *M) appendToLog(tag, msg string) {

	m.P.Log = append(m.P.Log, &Log{clock.Nano(), m.P.OvStatus, tag, msg})

	if len(m.P.Log) > MAXLOG {
		s := len(m.P.Log) - MAXLOG
		m.P.Log = m.P.Log[s:]
	}
}

// ################################################################

func (m *M) statsInit() {

	s := &m.P.Stats
	now := clock.Nano()

	if s.Lastt == 0 {
		s.Lastt = now / SECSNANO
	}

	if len(s.Daily) == 0 {
		s.Daily = []statDat{{Start: now}}
	}
	if len(s.Monthly) == 0 {
		s.Monthly = []statDat{{Start: now}}
	}
	if len(s.Yearly) == 0 {
		s.Yearly = []statDat{{Start: now}}
	}
}

func (m *M) statsTransition(prev argus.Status) {

	m.statsUpdateMaybeRoll()

	if m.P.OvStatus == argus.UNKNOWN || prev == argus.UNKNOWN {
		return
	}

	s := &m.P.Stats
	if prev == argus.CLEAR {
		s.Daily[0].Ndown++
		s.Monthly[0].Ndown++
		s.Yearly[0].Ndown++
	}

	s.Status = m.P.OvStatus
}

func (m *M) statsUpdate(t int64) {

	s := &m.P.Stats
	dt := int(t - s.Lastt)
	if dt == 0 {
		return
	}

	s.Daily[0].Elapsed += dt
	s.Monthly[0].Elapsed += dt
	s.Yearly[0].Elapsed += dt

	if s.Status == argus.CLEAR {
		s.Daily[0].TUp += dt
		s.Monthly[0].TUp += dt
		s.Yearly[0].TUp += dt
	} else {
		s.Daily[0].TDn += dt
		s.Monthly[0].TDn += dt
		s.Yearly[0].TDn += dt
	}

	s.Lastt = t
}

func (m *M) statsUpdateMaybeRoll() {

	now := clock.Unix()
	s := &m.P.Stats
	lt := time.Unix(s.Lastt, 0).Local()
	ct := time.Unix(now, 0).Local()

	if lt.Day() != ct.Day() {
		ch, cm, cs := ct.Clock()

		//seconds since midnight:
		ssm := (ch*60+cm)*60 + cs
		midnite := now - int64(ssm)

		// update yesterday's stats
		m.statsUpdate(midnite)
		m.statsRollOver(ct, lt, midnite)
		// A rolling log gathers no moss
	}
	m.statsUpdate(now)
}

func (m *M) statsRollOver(ct time.Time, lt time.Time, midnite int64) {

	m.P.Stats.Daily = statsRollOverStat(m.P.Stats.Daily, midnite)

	cy, cm, _ := ct.Date()
	ly, lm, _ := lt.Date()

	if cm != lm {
		m.P.Stats.Monthly = statsRollOverStat(m.P.Stats.Monthly, midnite)
	}
	if cy != ly {
		m.P.Stats.Yearly = statsRollOverStat(m.P.Stats.Yearly, midnite)
	}
}

func statsRollOverStat(s []statDat, t int64) []statDat {

	s = append(s, statDat{})
	copy(s[1:], s) // slide right
	s[0] = statDat{Start: t * SECSNANO}

	if len(s) > MAXSTAT {
		s = s[:MAXSTAT]
	}
	return s
}

func (m *M) StatsPeriodic() {

	m.Lock.Lock()
	defer m.Lock.Unlock()
	m.statsUpdateMaybeRoll()
}
