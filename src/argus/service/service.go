// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Aug-31 23:44 (EDT)
// Function: monitoring service

package service

import (
	"math"
	"sync"
	"time"

	"argus/argus"
	"argus/clock"
	"argus/configure"
	"argus/darp"
	"argus/diag"
	"argus/monel"
	"argus/sched"
)

type Monitor interface {
	Config(*configure.CF, *Service) error
	Init() error
	Recycle()
	Start(*Service)
	Abort()
	DoneConfig()
}

type Conf struct {
	Frequency    int
	Retries      int
	Retrydelay   int `cfconv:"timespec"`
	Timeout      int `cfconv:"timespec"`
	Showreason   bool
	Showresult   bool
	DARPGravity  argus.Gravity
	Severity     argus.Status
	DARP_Tags    string
	Calc         string
	calcmask     uint32
	Alpha        float64
	Scale        float64
	Pluck        string
	Unpack       string
	Expr         string
	JPath        string
	Testing      *argus.Schedule
	Checking     *argus.Schedule
	Expect       [argus.CRITICAL + 1]string  `cfconv:"dotsev"`
	Nexpect      [argus.CRITICAL + 1]string  `cfconv:"dotsev"`
	Minvalue     [argus.CRITICAL + 1]float64 `cfconv:"dotsev"` // NaN if not set
	Maxvalue     [argus.CRITICAL + 1]float64 `cfconv:"dotsev"`
	Eqvalue      [argus.CRITICAL + 1]float64 `cfconv:"dotsev"`
	Nevalue      [argus.CRITICAL + 1]float64 `cfconv:"dotsev"`
	Maxdeviation [argus.CRITICAL + 1]float64 `cfconv:"dotsev"`
	// calc, testing, schedule, graph,

}

var defaults = Conf{
	Frequency:   60,
	Retries:     2,
	Retrydelay:  60,
	Timeout:     60,
	DARPGravity: argus.GRAV_IETF,
	Severity:    argus.CRITICAL,
	Alpha:       1,
}

func init() {

	for i := argus.UNKNOWN; i <= argus.CRITICAL; i++ {
		defaults.Minvalue[i] = math.NaN()
		defaults.Maxvalue[i] = math.NaN()
		defaults.Eqvalue[i] = math.NaN()
		defaults.Nevalue[i] = math.NaN()
		defaults.Maxdeviation[i] = math.NaN()
	}
}

type Persist struct {
	Statuses map[string]argus.Status
	Result   string
	Results  map[string]string
	Reason   string
	Calc     calc
	Hwab     *HWAB
}

type Service struct {
	mon      *monel.M
	check    Monitor
	Cf       Conf
	p        Persist
	running  bool
	sched    *sched.D
	Lasttest int64
	Tries    int
	Started  int64
	AlsoRun  []*Service
	graph    bool
}

var dl = diag.Logger("service")
var lock sync.RWMutex
var allService = make(map[string]*Service)

/*
typical use:

  Start() JoinMulti()
  CheckValue(x) SetResult(st, v, r)
  Done()

*/

func (s *Service) Debug(fmt string, args ...interface{}) {
	s.mon.Debug(fmt, args...)
}

func (s *Service) SetNames(uname string, label string, friendly string) {
	s.mon.SetNames(uname, label, friendly)
}

func (s *Service) Start() {

	if !s.tasRunning() {
		if s.running && clock.Nano()-s.Started > int64(900*time.Second) {
			diag.Problem("%s - running too long. trying to abort", s.mon.Unique(), s.Started)
			s.check.Abort()

		}

		s.mon.Debug("not starting")
		s.reschedule()
		return
	}

	s.mon.Debug("service starting")
	s.check.Start(s)
}

func (s *Service) JoinMulti() bool {

	ok := s.tasRunning()
	if !ok {
		return false
	}

	s.mon.Debug("joining multi-start")
	s.reschedule()
	return true
}

func (s *Service) Done() {

	// debug

	s.mon.Lock.Lock()
	defer s.mon.Lock.Unlock()
	s.reschedule()
	s.running = false

	s.mon.Debug("done")
	for _, also := range s.AlsoRun {
		also.Start()
	}
}

func (s *Service) Fail(reason string) {
	s.SetResult(s.Cf.Severity, "", reason)
}

// ################################################################

func (s *Service) SetResult(status argus.Status, result string, reason string) {

	if s.Cf.Checking != nil && !s.Cf.Checking.PermitNow() {
		s.mon.Debug("checking bypassed by schedule")
		status = argus.CLEAR
	}

	// handle retries
	if status == argus.CLEAR {
		s.Tries = 0
	} else {
		if s.Tries <= s.Cf.Retries {
			s.mon.Debug("retrying (%d)", s.Tries)
			// status = s.p.Statuses[s.Cf.myid]
			s.Tries++
			return
		}
	}

	if status != s.p.Statuses[darp.MyId] {
		// send darp update to masters
		darp.SendUpdate(s.mon.Unique(), status, result, reason)
	}

	// RSN - archive
	s.SetResultFor(darp.MyId, status, result, reason)
}

func (s *Service) SetResultFor(id string, status argus.Status, result string, reason string) {

	s.mon.Debug("result[%s] => %s (%s)", id, status, reason)

	status = s.setResultForL(id, status, result, reason)

	// propagate change upwards
	s.mon.Debug("sending update %s", status)
	s.mon.Update(status, result, reason)
}

func (s *Service) setResultForL(id string, status argus.Status, result string, reason string) argus.Status {

	s.mon.Lock.Lock()
	defer s.mon.Lock.Unlock()

	s.p.Results[id] = result
	if id == darp.MyId {
		s.p.Reason = reason
	}

	s.p.Statuses[id] = status

	return darp.AggrStatus(s.Cf.DARPGravity, status, s.p.Statuses)
}

// ################################################################

func (s *Service) reschedule() {

	if s.Tries != 0 && s.Cf.Retrydelay != 0 {
		s.sched.ReSchedule(s.Cf.Retrydelay)
	} else {
		s.sched.ReSchedule(s.Cf.Frequency)
	}
}

func (s *Service) tasRunning() bool {

	s.mon.Lock.Lock()
	defer s.mon.Lock.Unlock()

	if s.running {
		return false
	}

	// RSN - disabled?
	if s.Cf.Testing != nil && !s.Cf.Testing.PermitNow() {
		return false
	}

	s.Started = clock.Nano()
	s.running = true

	return true
}

func (s *Service) recordGraphData(val float64) {

	// RSN - send to graphing channel
	// T, id, status, value, yn, dn, nmax{s,h.d}
}
