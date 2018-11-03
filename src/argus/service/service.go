// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Aug-31 23:44 (EDT)
// Function: monitoring service

package service

import (
	"expvar"
	"math"
	"sync"
	"time"

	"argus/argus"
	"argus/clock"
	"argus/configure"
	"argus/darp"
	"github.com/jaw0/acgo/diag"
	"argus/monel"
	"argus/sched"
)

type Monitor interface {
	PreConfig(*configure.CF, *Service) error
	Config(*configure.CF, *Service) error
	Init() error
	Recycle()
	Start(*Service)
	Abort()
	DoneConfig()
	DumpInfo() map[string]interface{}
	WebJson(map[string]interface{})
	Hostname() string
	Priority() bool
}

type Conf struct {
	Frequency    int
	Retries      int
	Retrydelay   int `cfconv:"timespec"`
	Timeout      int `cfconv:"timespec"`
	Showreason   bool
	Showresult   bool
	Severity     argus.Status
	DARP_Gravity argus.Gravity
	DARP_Tags    string
	Calc         string
	Alpha        float64
	Scale        float64
	Pluck        string
	Unpack       string
	Expr         string
	JPath        string
	XPath        string
	Gr_what      string // only 'elapsed' is supported
	Testing      *argus.Schedule
	Checking     *argus.Schedule
	Expect       [argus.CRITICAL + 1]string  `cfconv:"dotsev"`
	Nexpect      [argus.CRITICAL + 1]string  `cfconv:"dotsev"`
	Minvalue     [argus.CRITICAL + 1]float64 `cfconv:"dotsev"` // NaN if not set
	Maxvalue     [argus.CRITICAL + 1]float64 `cfconv:"dotsev"`
	Eqvalue      [argus.CRITICAL + 1]float64 `cfconv:"dotsev"`
	Nevalue      [argus.CRITICAL + 1]float64 `cfconv:"dotsev"`
	Maxdeviation [argus.CRITICAL + 1]float64 `cfconv:"dotsev"`
	// graph,

}

var defaults = Conf{
	Frequency:    60,
	Retries:      2,
	Retrydelay:   60,
	Timeout:      15,
	DARP_Gravity: argus.GRAV_IETF,
	Severity:     argus.CRITICAL,
	Alpha:        1,
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
	Lastgraph int64
	Statuses  map[string]argus.Status
	Result    string
	Results   map[string]string
	Reason    string
	Calc      calc
	Hwab      *HWAB
}

type Service struct {
	mon      *monel.M
	check    Monitor
	Cf       Conf
	p        Persist
	running  bool
	ready    bool
	sched    *sched.D
	Lasttest int64
	Tries    int
	Started  int64
	Elapsed  int64
	alsoRun  []*Service
	calcmask uint32
	expr     []string
}

var dl = diag.Logger("service")
var lock sync.RWMutex
var allService = make(map[string]*Service)
var NService = expvar.NewInt("services")

/*
typical use:

  Start() JoinMulti()
  CheckValue(x) SetResult(st, v, r)
  Done()

*/

func (s *Service) Debug(fmt string, args ...interface{}) {
	s.mon.Debug(fmt, args...)
}
func (s *Service) Loggit(tag, msg string) {
	s.mon.Loggit(tag, msg)
}

func (s *Service) CFError(fmt string, args ...interface{}) {
	s.mon.ConfCF.Error(fmt, args...)
}

func (s *Service) SetNames(uname string, label string, friendly string) {
	s.mon.SetNames(uname, label, friendly)
}

func (s *Service) Hostname() string {
	return s.check.Hostname()
}
func Find(id string) *Service {
	lock.RLock()
	defer lock.RUnlock()
	return allService[id]
}

func (s *Service) CheckNow() {
	s.sched.ASAP()
}

func (s *Service) IsReady() bool {

	s.mon.Lock.RLock()
	defer s.mon.Lock.RUnlock()
	return s.ready
}

func (s *Service) Unique() string {
	return s.mon.Cf.Unique
}

func (s *Service) Start() {

	if !s.tasRunning() {
		if s.running && clock.Nano()-s.Started > int64(900*time.Second) {
			diag.Problem("%s - running too long. trying to abort", s.mon.Unique(), s.Started)
			s.check.Abort()

		}

		s.mon.Debug("not starting")
		s.reschedule(0)
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
	s.reschedule(0)
	return true
}

func (s *Service) Done() {

	now := clock.Nano()
	s.mon.Lock.Lock()
	s.reschedule(int(s.Elapsed))
	s.running = false
	s.Lasttest = now

	alsorun := s.alsoRun
	s.mon.Lock.Unlock()
	s.mon.Debug("done")

	for _, also := range alsorun {
		also.CheckNow()
	}
}

func (s *Service) Pass() {
	s.ready = true
	s.SetResult(argus.CLEAR, "", "")
}
func (s *Service) FailReady(reason string) {
	s.ready = true
	s.SetResult(s.Cf.Severity, "", reason)
}
func (s *Service) Fail(reason string) {
	s.ready = false
	s.SetResult(s.Cf.Severity, "", reason)
}
func (s *Service) FailNow(reason string) {
	s.ready = false
	s.Tries = s.Cf.Retries + 1
	s.Fail(reason)
}
func (s *Service) Children() []*monel.M {
	return nil
}
func (s *Service) Self() *monel.M {
	return s.mon
}

func (s *Service) AddAlsoRun(c *Service) {
	s.mon.Lock.Lock()
	s.alsoRun = append(s.alsoRun, c)
	s.mon.Lock.Unlock()
}

func (s *Service) RemoveAlsoRun(c *Service) {

	s.mon.Lock.Lock()
	defer s.mon.Lock.Unlock()
	i := -1

	for n, cs := range s.alsoRun {
		if cs == c {
			i = n
			break
		}
	}

	if i == -1 {
		return
	}

	copy(s.alsoRun[i:], s.alsoRun[i+1:])
	s.alsoRun = s.alsoRun[:len(s.alsoRun)-1]
}

// ################################################################

func (s *Service) SetResult(status argus.Status, result string, reason string) {

	if s.Cf.Checking != nil && !s.Cf.Checking.PermitNow("yes") {
		s.mon.Debug("checking bypassed by schedule")
		status = argus.CLEAR
	}

	// handle retries
	if status == argus.CLEAR {
		s.Tries = 0
		reason = ""
	} else {
		if s.Tries <= s.Cf.Retries {
			s.mon.Debug("retrying (%d)", s.Tries)
			s.Tries++
			return
		}
	}

	if status != s.p.Statuses[darp.MyId] {
		// send darp update to masters
		darp.SendUpdate(s.mon.Unique(), status, result, reason)
	}

	// RSN - archive
	s.p.Result = result
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

	return darp.AggrStatus(s.Cf.DARP_Gravity, status, s.p.Statuses)
}

// ################################################################

func (s *Service) reschedule(dt int) {

	if s.Tries != 0 && s.Cf.Retrydelay != 0 {
		s.sched.ReSchedule(s.Cf.Retrydelay, dt/1000000)
	} else {
		s.sched.ReSchedule(s.Cf.Frequency, dt/1000000)
	}
}

func (s *Service) tasRunning() bool {

	s.mon.Lock.Lock()
	defer s.mon.Lock.Unlock()

	if s.running {
		return false
	}

	// RSN - disabled?
	if s.Cf.Testing != nil && !s.Cf.Testing.PermitNow("yes") {
		return false
	}

	s.Started = clock.Nano()
	s.running = true

	return true
}
