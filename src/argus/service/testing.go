// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Sep-01 19:03 (EDT)
// Function:

package service

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"regexp"
	"strings"

	"github.com/ChrisTrenkamp/goxpath"
	"github.com/ChrisTrenkamp/goxpath/tree/xmltree"
	"github.com/jaw0/acgo/diag"
	"github.com/oliveagle/jsonpath"

	"argus/argus"
	"argus/clock"
	"argus/expr"
)

const (
	CALC_ONE     uint32 = 1 << 0
	CALC_ELAPSED uint32 = 1 << 1
	CALC_RATE    uint32 = 1 << 2
	CALC_DELTA   uint32 = 1 << 3
	CALC_AVE     uint32 = 1 << 4
	CALC_JITTER  uint32 = 1 << 5
	CALC_BITS    uint32 = 1 << 6
)

type calc struct {
	rawvalue string
	Lastv    float64
	Lastdv   float64
	Lastt    int64
	Lastta   int64
	Ave      float64
}

func (s *Service) CheckValue(val string, valtype string) {

	var fval float64

	s.p.Calc.rawvalue = val
	s.Elapsed = clock.Nano() - s.Started

	s.mon.Debug("rawvalue: %s", limitString(val, 40))
	val, fval, valtype = s.getValue(val, valtype)

	if valtype == "skip" {
		return
	}

	s.ready = true
	status, reason := s.testAndCompare(val, fval, valtype)

	if valtype == "" {
		val = fmt.Sprintf("%f", fval)
	} else {
		fmt.Sscan(val, &fval)
	}

	s.recordMyGraphData(fval)

	s.mon.Debug("value '%s' (%s) -> status %s (%s)", limitString(val, 16), valtype, status, reason)
	s.SetResult(status, val, reason)
}

func limitString(s string, limit int) string {

	if len(s) <= limit {
		return s
	}

	return s[:limit] + "..."
}

func (s *Service) testAndCompare(val string, fval float64, valtype string) (argus.Status, string) {

	if valtype == "" {
		val = fmt.Sprintf("%f", fval)
	} else {
		fmt.Sscan(val, &fval)
	}

	for sev := argus.CRITICAL; sev >= argus.UNKNOWN; sev-- {

		rsev := sev
		if sev == argus.CLEAR {
			continue
		}
		if sev == argus.UNKNOWN {
			rsev = s.Cf.Severity
		}

		if s.Cf.Expect[sev] != "" {
			if !testMatch(s.Cf.Expect[sev], val) {
				return rsev, "TEST did not match expected regex"
			}
		}
		if s.Cf.Nexpect[sev] != "" {
			if testMatch(s.Cf.Nexpect[sev], val) {
				return rsev, "TEST did matched unexpected regex"
			}
		}
		if !math.IsNaN(s.Cf.Minvalue[sev]) {
			if fval < s.Cf.Minvalue[sev] {
				return rsev, "TEST less than min"
			}
		}
		if !math.IsNaN(s.Cf.Maxvalue[sev]) {
			if fval > s.Cf.Maxvalue[sev] {
				return rsev, "TEST more than max"
			}
		}
		if !math.IsNaN(s.Cf.Eqvalue[sev]) {
			if fval != s.Cf.Eqvalue[sev] {
				return rsev, "TEST not equal"
			}
		}
		if !math.IsNaN(s.Cf.Nevalue[sev]) {
			if fval == s.Cf.Nevalue[sev] {
				return rsev, "TEST equal"
			}
		}
		if s.p.Hwab != nil && !math.IsNaN(s.Cf.Maxdeviation[sev]) {
			dev, ok := s.p.Hwab.Deviation(fval)
			if ok && dev > s.Cf.Maxdeviation[sev] {
				return rsev, "TEST outside of predicted range"
			}
		}
	}

	return argus.CLEAR, ""
}

func (s *Service) getValue(val string, valtype string) (string, float64, string) {

	var fval float64
	now := clock.Nano()

	if s.Cf.Pluck != "" {
		val = Pluck(s.Cf.Pluck, val)
		valtype = "string"
		s.Debug("pluck => value %v", val)
	}

	if s.Cf.JPath != "" && valtype != "" {
		var err error
		s.Debug("type %s; jpath %s; val %s", valtype, s.Cf.JPath, val)
		val, err = JsonPath(s.Cf.JPath, val)
		if err != nil {
			diag.Verbose("invalid json/jsonpath '%s': %v", s.Cf.JPath, err)
		}
		valtype = "string"
		s.Debug("json => value %#v", val)
	}

	if s.Cf.XPath != "" && valtype != "" {
		var err error
		val, err = XPath(s.Cf.XPath, val)
		if err != nil {
			diag.Verbose("invalid xml/xpath '%s': %v", s.Cf.XPath, err)
		}
		valtype = "string"
		s.Debug("xml => value %v", val)
	}

	if s.Cf.Unpack != "" {
		ival, ok := argus.Unpack(s.Cf.Unpack, []byte(val))
		if !ok {
			diag.Verbose("invalid unpack '%s'")
		}
		fval = float64(ival)
		valtype = ""
		s.Debug("unpack => value %f", fval)
	}

	if s.Cf.Scale != 0 || s.calcmask != 0 || s.Cf.Expr != "" || s.p.Hwab != nil {
		if valtype != "" {
			// convert string -> float
			fmt.Sscan(val, &fval)
			valtype = ""
		}
	}

	if s.Cf.Scale != 0 {
		fval /= s.Cf.Scale
	}

	if s.calcmask&CALC_ONE != 0 {
		fval = 1
	}
	if s.calcmask&CALC_ELAPSED != 0 {
		fval = float64(s.Elapsed) / 1e9
	}
	if s.calcmask&(CALC_RATE|CALC_DELTA) != 0 {
		var ok bool
		fval, ok = s.rateCalc(s.calcmask, fval)
		if !ok {
			return "", 0, "skip"
		}
	}
	if s.calcmask&(CALC_AVE|CALC_JITTER) != 0 {
		dt := float64(now-s.p.Calc.Lastta) / 1e9

		if s.p.Calc.Lastta == 0 || dt > float64(s.Cf.Frequency)*s.Cf.Alpha*3 {
			// initial value
			s.p.Calc.Ave = fval
		}

		fval = (s.Cf.Alpha*s.p.Calc.Ave + fval) / (s.Cf.Alpha + 1)
		pave := s.p.Calc.Ave
		s.p.Calc.Ave = fval

		if s.calcmask&CALC_JITTER != 0 {
			fval = math.Abs(pave - fval)
		}
		s.p.Calc.Lastta = now
	}
	if s.calcmask&CALC_BITS != 0 {
		fval *= 8
	}

	if len(s.expr) != 0 {
		var err error
		var nrdy string

		fval, nrdy, err = s.doExpr(s.expr, fval)

		if err != nil {
			s.Debug("invalid expr '%s': %v", s.Cf.Expr, err)
			s.FailNow("invalid expr")
			return "", 0, "skip"
		}

		if nrdy != "" {
			// service is not yet ready
			s.Debug("not ready: %s", nrdy)
			return "", 0, "skip"
		}
	}

	if s.p.Hwab != nil {
		s.p.Hwab.Add(fval)
	}

	return val, fval, valtype
}

// if we detect the device has reset
func (s *Service) ResetRateCalc() {
	s.p.Calc.Lastt = 0
	s.p.Calc.Lastv = 0
}

func (s *Service) rateCalc(calcmask uint32, fval float64) (float64, bool) {

	s.mon.Lock.Lock()
	defer s.mon.Lock.Unlock()

	now := clock.Nano()
	c := &s.p.Calc
	dt := float64(now-c.Lastt) / 1e9

	if c.Lastt == 0 {
		// startup transient - skip
		c.Lastv = fval
		c.Lastt = now
		s.mon.Debug("TEST delta startup")
		return 0, false
	}
	if dt < 1 {
		// too soon - skip
		s.mon.Debug("TEST too soon to retest. skipping")
		return 0, false
	}

	lastv := c.Lastv
	c.Lastv = fval
	c.Lastt = now

	var dv float64

	if fval < lastv {
		// handle counter issues
		if lastv < float64(0x7fffffff) {
			// assume reboot/reset
			s.mon.Debug("TEST possible reboot detected")
			return 0, false
		} else {
			// overflow/wraparound
			dv = float64(0xFFFFFFFF) - lastv + 1
			s.mon.Debug("TEST counter rollover detected")
		}
	} else {
		dv = fval - lastv
	}

	c.Lastdv = dv

	if c.Lastdv != 0 && dv > 100*c.Lastdv {
		// unusually large spike, probably a reset/reboot - supress
		s.mon.Debug("TEST supressing transient spike (%s)", dv)
		return 0, false
	}

	if calcmask&CALC_RATE != 0 {
		fval = dv / dt
	} else {
		fval = dv
	}

	return fval, true
}

func testMatch(regex string, val string) bool {

	m, err := regexp.MatchString(regex, val)
	if err != nil {
		diag.Problem("invalid match regexp '%s': %v", regex)
		return false
	}

	return m
}

func Pluck(regex string, val string) string {

	re, err := regexp.Compile(regex)
	if err != nil {
		diag.Problem("invalid pluck regexp '%s': %v", regex)
		return ""
	}

	matches := re.FindStringSubmatch(val)

	if len(matches) < 2 {
		return ""
	}
	return matches[1]
}

func JsonPath(path string, val string) (ret string, rer error) {

	defer func() {
		if x := recover(); x != nil {
			rer = errors.New("invalid jpath")
		}
	}()

	var jdat interface{}
	err := json.Unmarshal([]byte(val), &jdat)
	if err != nil {
		return "", err
	}
	res, err := jsonpath.JsonPathLookup(jdat, path)

	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%v", res), nil
}

func XPath(path string, val string) (ret string, rer error) {

	defer func() {
		if x := recover(); x != nil {
			rer = errors.New("invalid xpath")
		}
	}()

	xTree, err := xmltree.ParseXML(bytes.NewBufferString(val))
	if err != nil {
		return "", err
	}

	xpExec, err := goxpath.Parse(path)
	if err != nil {
		return "", err
	}

	res, err := xpExec.ExecNum(xTree)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%v", res), nil
}

func (s *Service) doExpr(exp []string, fval float64) (ret float64, nrdy string, rer error) {

	defer func() {
		if x := recover(); x != nil {
			rer = fmt.Errorf("invalid expr: %v", x)
		}
	}()

	sv := fmt.Sprintf("%f", fval)

	dat := map[string]string{
		"elapsed": fmt.Sprintf("%f", float32(s.Elapsed)/1e9),
		"lastt":   fmt.Sprintf("%d", s.Lasttest/1e9),
		"freq":    fmt.Sprintf("%d", s.Cf.Frequency),
		"value":   sv,
		"x":       sv,
		"$x":      sv, // backwards compat(ish)
	}

	res, nrdy, err := expr.RunExprF(exp, dat)

	return res, nrdy, err
}

func calcMask(calc string) uint32 {

	f := strings.Fields(strings.Replace(strings.ToLower(calc), "-", " ", -1))
	var mask uint32

	for _, c := range f {
		switch c {
		case "elapsed":
			mask |= CALC_ELAPSED
		case "rate":
			mask |= CALC_RATE
		case "delta":
			mask |= CALC_DELTA
		case "ave":
			mask |= CALC_AVE
		case "JITTER":
			mask |= CALC_JITTER
		case "bits":
			mask |= CALC_BITS
		case "one":
			mask |= CALC_ONE
		}
	}
	return mask
}
