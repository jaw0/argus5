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

// in unix time, seconds
type statDat struct {
	start   int64
	elapsed int
	ndown   int
	statt   [argus.MAXSTATUS + 1]int
}
type Stats struct {
	lastt   int64
	status  argus.Status
	daily   []statDat
	monthly []statDat
	yearly  []statDat
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

}

func (m *M) statsTransition(prev argus.Status) {

	m.statsUpdateMaybeRoll()

	if m.P.OvStatus == argus.UNKNOWN || prev == argus.UNKNOWN {
		return
	}

	s := m.P.Stats
	if prev == argus.CLEAR {
		s.daily[0].ndown++
		s.monthly[0].ndown++
		s.yearly[0].ndown++
	}

	s.status = m.P.OvStatus
}

func (m *M) statsUpdate(t int64) {

	s := m.P.Stats
	dt := int(t - s.lastt)
	if dt == 0 {
		return
	}

	s.daily[0].elapsed += dt
	s.daily[0].statt[s.status] += dt
	s.monthly[0].elapsed += dt
	s.monthly[0].statt[s.status] += dt
	s.yearly[0].elapsed += dt
	s.yearly[0].statt[s.status] += dt

	s.lastt = t
}

func (m *M) statsUpdateMaybeRoll() {

	now := clock.Unix()
	s := m.P.Stats
	lt := time.Unix(s.lastt, 0).Local()
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

	m.P.Stats.daily = statsRollOverStat(m.P.Stats.daily, midnite)

	cy, cm, _ := ct.Date()
	ly, lm, _ := lt.Date()

	if cm != lm {
		m.P.Stats.monthly = statsRollOverStat(m.P.Stats.monthly, midnite)
	}
	if cy != ly {
		m.P.Stats.yearly = statsRollOverStat(m.P.Stats.yearly, midnite)
	}
}

func statsRollOverStat(s []statDat, t int64) []statDat {

	s = append(s, statDat{})
	copy(s[1:], s) // slide right
	s[0] = statDat{start: t}

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
