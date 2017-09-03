// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Sep-01 19:03 (EDT)
// Function:

package service

import (
	"encoding/json"
	"fmt"
	"math"
	"regexp"

	"github.com/uknth/jsonpath"
	"github.com/zdebeer99/goexpression"

	"argus/argus"
	"argus/clock"
	"argus/diag"
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
	s.mon.Debug("rawvalue: %s", val)
	val, fval, valtype = s.getValue(val, valtype)

	if valtype == "skip" {
		return
	}

	status, reason := s.testAndCompare(val, fval, valtype)

	if valtype == "" {
		val = fmt.Sprintf("%f", fval)

		if s.graph { // XXX
			s.recordGraphData(fval)
		}
	}

	s.mon.Debug("value '%s' -> status %s (%s)", val, status, reason)
	s.SetResult(status, val, reason)
}

func (s *Service) testAndCompare(val string, fval float64, valtype string) (argus.Status, string) {

	if valtype == "" {
		val = fmt.Sprintf("%f", fval)
	} else {
		fmt.Sscan(val, &fval)
	}

	for sev := argus.CRITICAL; sev > argus.CLEAR; sev-- {

		if s.cf.Expect[sev] != "" {
			if !testMatch(s.cf.Expect[sev], val) {
				return sev, "TEST did not match expected regex"
			}
		}
		if s.cf.Nexpect[sev] != "" {
			if testMatch(s.cf.Nexpect[sev], val) {
				return sev, "TEST did matched unexpected regex"
			}
		}
		if !math.IsNaN(s.cf.Minvalue[sev]) {
			if fval < s.cf.Minvalue[sev] {
				return sev, "TEST less than min"
			}
		}
		if !math.IsNaN(s.cf.Maxvalue[sev]) {
			if fval > s.cf.Maxvalue[sev] {
				return sev, "TEST more than max"
			}
		}
		if !math.IsNaN(s.cf.Eqvalue[sev]) {
			if fval != s.cf.Eqvalue[sev] {
				return sev, "TEST not equal"
			}
		}
		if !math.IsNaN(s.cf.Nevalue[sev]) {
			if fval == s.cf.Nevalue[sev] {
				return sev, "TEST equal"
			}
		}
		if s.p.Hwab != nil && !math.IsNaN(s.cf.Maxdeviation[sev]) {
			dev, ok := s.p.Hwab.Deviation(fval)
			if ok && dev > s.cf.Maxdeviation[sev] {
				return sev, "TEST outside of predicted range"
			}
		}
	}

	return argus.CLEAR, "OK"
}

func (s *Service) getValue(val string, valtype string) (string, float64, string) {

	var fval float64
	now := clock.Nano()

	if s.cf.Pluck != "" {
		val = pluck(s.cf.Pluck, val)
		valtype = "string"
	}

	if s.cf.JPath != "" && valtype == "json" {
		var err error
		val, err = jsonPath(s.cf.JPath, val)
		if err != nil {
			diag.Problem("invalid json/jsonpath '%s': %v", s.cf.JPath, err)
		}
		valtype = "string"
	}

	if s.cf.Unpack != "" {
		ival, ok := argus.Unpack(s.cf.Unpack, []byte(val))
		if !ok {
			diag.Problem("invalid unpack '%s'")
		}
		fval = float64(ival)
		valtype = ""
	}

	if s.cf.Scale != 0 || s.cf.calcmask != 0 || s.cf.Expr != "" || s.p.Hwab != nil {
		if valtype != "" {
			// convert string -> float
			fmt.Sscan(val, &fval)
			valtype = ""
		}
	}

	if s.cf.Scale != 0 {
		fval /= s.cf.Scale
	}

	if s.cf.calcmask&CALC_ONE != 0 {
		fval = 1
	}
	if s.cf.calcmask&CALC_ELAPSED != 0 {
		fval = float64(now-s.Started) / 1e9
	}
	if s.cf.calcmask&(CALC_RATE|CALC_DELTA) != 0 {
		var ok bool
		fval, ok = s.rateCalc(s.cf.calcmask, fval)
		if !ok {
			return "", 0, "skip"
		}
	}
	if s.cf.calcmask&(CALC_AVE|CALC_JITTER) != 0 {
		dt := float64(now-s.p.Calc.Lastta) / 1e9

		if s.p.Calc.Lastta == 0 || dt > float64(s.cf.Frequency)*s.cf.Alpha*3 {
			// initial value
			s.p.Calc.Ave = fval
		}

		fval = (s.cf.Alpha*s.p.Calc.Ave + fval) / (s.cf.Alpha + 1)
		pave := s.p.Calc.Ave
		s.p.Calc.Ave = fval

		if s.cf.calcmask&CALC_JITTER != 0 {
			fval = math.Abs(pave - fval)
		}
		s.p.Calc.Lastta = now
	}
	if s.cf.calcmask&CALC_BITS != 0 {
		fval *= 8
	}

	if s.cf.Expr != "" {
		var err error
		fval, err = doExpr(s.cf.Expr, fval)
		if err != nil {
			diag.Problem("invalid expr '%s': %v", s.cf.Expr, err)
		}
	}

	if s.p.Hwab != nil {
		s.p.Hwab.Add(fval)
	}

	return val, fval, valtype
}

func (s *Service) rateCalc(calcmask uint32, fval float64) (float64, bool) {

	now := clock.Nano()
	dt := float64(now-s.p.Calc.Lastt) / 1e9
	c := &s.p.Calc

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

	c.Lastv = fval
	c.Lastt = now

	var dv float64
	if fval < c.Lastv {
		// handle counter issues
		if c.Lastv < float64(0x7fffffff) {
			// assume reboot/reset
			s.mon.Debug("TEST possible reboot detected")
			return 0, false
		} else {
			// overflow/wraparound
			dv = float64(0xFFFFFFFF) - c.Lastv + 1
			s.mon.Debug("TEST counter rollover detected")
		}
	} else {
		dv = fval - c.Lastv
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

func pluck(regex string, val string) string {

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

func jsonPath(path string, val string) (string, error) {

	// RSN - save jdat for multi-service tests?

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

func doExpr(expr string, fval float64) (ret float64, rer error) {

	dat := map[string]interface{}{
		"x": fval,
	}

	defer func() {
		if err := recover(); err != nil {
			rer = fmt.Errorf("%v", err)
			ret = 0
		}
	}()

	// NB - this does not return an error, it panics!
	r := goexpression.Eval(expr, dat)

	return r, nil
}

/*
https://github.com/zdebeer99/goexpression


*/
