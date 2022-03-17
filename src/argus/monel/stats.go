// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Sep-06 19:32 (EDT)
// Function: logs + stats

package monel

import (
	"fmt"
	"time"

	"argus.domain/argus/argus"
	"argus.domain/argus/clock"
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

type StatsExport struct {
	Label   string
	HiLite  bool
	Start   int64
	Elapsed string
	NDown   int
	PctUp   string
	PctDn   string
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
		s.Status = m.P.Status
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

	// discard time spent while not running
	s.Lastt = now / SECSNANO

	m.statsInitMaybeRoll()
	m.statsTransition()
}

func (m *M) statsTransition() {

	m.statsUpdateMaybeRoll()
	s := &m.P.Stats
	dl.Debug("%s -> %s", s.Status, m.P.Status)

	if m.P.Status == argus.UNKNOWN || s.Status == argus.UNKNOWN {
		s.Status = m.P.Status
		return
	}

	if s.Status == argus.CLEAR && m.P.Status != argus.CLEAR {
		s.Daily[0].Ndown++
		s.Monthly[0].Ndown++
		s.Yearly[0].Ndown++
	}

	s.Status = m.P.Status
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

	dl.Debug("stats update %s + %d", s.Status, dt)

	if s.Status == argus.CLEAR || s.Status == argus.UNKNOWN {
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

		m.WebTime = clock.Nano()
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

func (m *M) statsInitMaybeRoll() {

	now := clock.Unix()
	s := &m.P.Stats
	ct := time.Unix(now, 0).Local()
	ld := time.Unix(s.Daily[0].Start/SECSNANO, 0).Local()
	lm := time.Unix(s.Monthly[0].Start/SECSNANO, 0).Local()
	ly := time.Unix(s.Yearly[0].Start/SECSNANO, 0).Local()

	cty, ctm, ctd := ct.Date()
	ldy, ldm, ldd := ld.Date()
	lmy, lmm, _ := lm.Date()
	lyy, _, _ := ly.Date()

	if ctd != ldd || ctm != ldm || cty != ldy {
		m.P.Stats.Daily = statsRollOverStat(m.P.Stats.Daily, now)
	}
	if ctm != lmm || cty != lmy {
		m.P.Stats.Monthly = statsRollOverStat(m.P.Stats.Monthly, now)
	}
	if cty != lyy {
		m.P.Stats.Yearly = statsRollOverStat(m.P.Stats.Yearly, now)
	}
}

// ################################################################
// stats for web page

func (m *M) ExportStats() []*StatsExport {

	var res []*StatsExport
	res = appendExportStats(res, "Today", m.P.Stats.Daily, 0)
	res = appendExportStats(res, "Yesterday", m.P.Stats.Daily, 1)
	res = appendExportStats(res, "2 Days Ago", m.P.Stats.Daily, 2)
	res = appendExportStats(res, "This Month", m.P.Stats.Monthly, 0)
	res = appendExportStats(res, "Last Month", m.P.Stats.Monthly, 1)
	res = appendExportStats(res, "2 Months Ago", m.P.Stats.Monthly, 2)
	res = appendExportStats(res, "This Year", m.P.Stats.Yearly, 0)
	res = appendExportStats(res, "Last Year", m.P.Stats.Yearly, 1)
	res = appendExportStats(res, "2 Years Ago", m.P.Stats.Yearly, 2)

	return res
}

func appendExportStats(r []*StatsExport, label string, stats []statDat, idx int) []*StatsExport {

	if idx >= len(stats) {
		return r
	}

	s := stats[idx]

	if s.Elapsed == 0 {
		return r
	}

	// RSN - pretty strings
	r = append(r, &StatsExport{
		Label:   label,
		Start:   s.Start,
		Elapsed: argus.Elapsed(int64(s.Elapsed)),
		HiLite:  idx == 0,
		NDown:   s.Ndown,
		PctUp:   percent(float32(s.TUp) / float32(s.Elapsed) * 100),
		PctDn:   percent(float32(s.TDn) / float32(s.Elapsed) * 100),
	})

	return r
}

func percent(f float32) string {

	if f <= 0 {
		return "0.00"
	}
	if f >= 100 {
		return "100.0"
	}
	if f > 99.99 {
		return fmt.Sprintf("%.4f", f)
	}
	return fmt.Sprintf("%.2f", f)
}

func (m *M) StatsPeriodic() {

	m.Lock.Lock()
	defer m.Lock.Unlock()
	m.statsUpdateMaybeRoll()
}
