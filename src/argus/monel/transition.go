// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Sep-06 19:19 (EDT)
// Function: status transition

package monel

import (
	"argus/argus"
	"argus/clock"
)

// m is a service updating status
func (m *M) Update(status argus.Status, result string, reason string) {

	m.Lock.Lock()
	defer m.Lock.Unlock()
	prev := m.P.OvStatus
	changed := m.updateStatus(status, result, reason)

	if !changed {
		return
	}

	m.commonUpdate(prev)
}

func (m *M) updateStatus(status argus.Status, result string, reason string) bool {

	m.P.Result = result
	if status == m.P.Status {
		return false
	}
	m.P.Status = status
	m.P.Reason = reason

	return m.determineStatus()
}

// update group status + ovstatus
func (m *M) UpUpdate(by *M) {

	m.Lock.Lock()
	defer m.Lock.Unlock()
	prev := m.P.OvStatus
	changed := m.determineStatus()

	if !changed {
		return
	}
	m.P.Reason = by.Cf.Uname

	m.commonUpdate(prev)
}

func (m *M) commonUpdate(prevOv argus.Status) {

	// ov status summary
	m.setAlarm()
	m.loggitL("TRANSITION", m.P.Reason)
	m.statsTransition(prevOv)
	m.maybeNotify(prevOv)

	m.andUpwards()
}

func (m *M) andUpwards() {

	// or push to a channel?
	// propagate upwards!
	for _, parent := range m.Parent {
		go parent.UpUpdate(m)
	}

	// and anything depending on me?
}

// ################################################################

func (m *M) maybeNotify(prevOv argus.Status) {

	// anc_in_ov
}

func (m *M) setAlarm() {

	m.P.TransTime = clock.Nano()
	a := m.P.Alarm

	if m.P.OvStatus > argus.CLEAR && m.P.OvStatus <= argus.CRITICAL {
		a = true
	}

	if a != m.P.Alarm {
		m.P.Alarm = a
		m.P.SirenTime = m.P.TransTime
	}
}

// ################################################################

// determine status + ovstatus
// lock should already be held
func (m *M) determineStatus() bool {

	prevo := m.P.OvStatus

	m.determineAggrStatus()
	m.checkDepends()
	m.checkOverride()

	return m.P.OvStatus == prevo
}

func (m *M) checkOverride() {

	if m.P.Status == argus.CLEAR || m.P.Status == argus.UNKNOWN {
		// do we need to remove?
		return
	}
	if m.P.OvStatus == argus.DEPENDS {
		return
	}

	if m.P.Override != nil {
		m.P.OvStatus = argus.OVERRIDE
	}
}

// dtermine our aggregate status
// lock should already be held
func (m *M) determineAggrStatus() {

	if len(m.Children) == 0 {
		return
	}

	nchild := 0
	rsum := [argus.MAXSTATUS + 1]int{}
	osum := [argus.MAXSTATUS + 1]int{}

	for _, child := range m.Children {
		rs, os := child.Status()
		rsum[rs]++
		osum[os]++
		nchild++
	}

	rs := calcAggrStatus(m.Cf.Gravity, nchild, argus.CRITICAL, rsum[:])
	os := calcAggrStatus(m.Cf.Gravity, nchild, argus.MAXSTATUS, osum[:])

	m.P.Status = rs
	m.P.OvStatus = os

}

func calcAggrStatus(grav argus.Gravity, tot int, max argus.Status, statuses []int) argus.Status {

	tot -= statuses[int(argus.UNKNOWN)]

	switch grav {
	case argus.GRAV_DN:
		for sev := max; sev >= argus.CLEAR; sev-- {
			if statuses[int(sev)] > 0 {
				return sev
			}
		}

	case argus.GRAV_UP:
		for sev := argus.CLEAR; sev <= max; sev++ {
			if statuses[int(sev)] > 0 {
				return sev
			}
		}
		return argus.CLEAR
	default:
		lim := tot / 2
		cum := 0
		for sev := argus.CLEAR; sev <= max; sev++ {
			cum += statuses[int(sev)]
			if cum >= lim {
				return sev
			}
		}
	}

	return argus.UNKNOWN
}
