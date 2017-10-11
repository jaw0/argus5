// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Sep-02 14:28 (EDT)
// Function:

package service

import (
	"math"

	"argus/clock"
	"argus/configure"
	"argus/monel"
)

type HwabConf struct {
	Alpha  float32
	Beta   float32
	Gamma  float32
	Zeta   float32
	Period int `cf:"timespec"`
}

type HWAB struct {
	cf      HwabConf
	mon     *monel.M
	buckets int
	Created int64
	cstart  int64
	ctotal  float64
	ctota2  float64
	ccount  int
	valid   bool
	yn      float32
	dn      float32
	A       float32
	B       float32
	C       []float32
	D       []float32
}

var hwabdefaults = HwabConf{
	Alpha:  0.005,
	Beta:   0.0005,
	Gamma:  0.1,
	Period: 7 * 24 * 3600,
}

const (
	TWIN       = 300
	PERIOD_MIN = 2
	PERIOD_MAX = 30 * 24 * 3600
)

func (s *Service) HwabConfig(conf *configure.CF) error {

	h := &HWAB{
		Created: clock.Unix(),
		mon:     s.mon,
	}
	h.cf = hwabdefaults
	s.p.Hwab = h

	conf.InitFromConfig(&s.Cf, "service", "hwab_")

	if h.cf.Period < PERIOD_MIN {
		h.cf.Period = PERIOD_MIN
	}
	if h.cf.Period > PERIOD_MAX {
		h.cf.Period = PERIOD_MAX
	}

	h.buckets = h.cf.Period / TWIN

	h.C = make([]float32, h.buckets)
	h.D = make([]float32, h.buckets)

	return nil
}

func (h *HWAB) Init() error {

	return nil
}

func (h *HWAB) Add(val float64) {

	// resample from service.Freq -> TWIN

	h.mon.Debug("hwab add")
	now := clock.Unix()
	if h.cstart == 0 {
		h.cstart = now
	}
	h.ccount++
	h.ctotal += val
	h.ctota2 += val * val

	if h.cstart+TWIN > now {
		return
	}

	ave := h.ctotal / float64(h.ccount)
	sdv := math.Sqrt(h.ctota2/float64(h.ccount) - ave*ave)

	for h.cstart+TWIN <= now {
		h.add(float32(ave), float32(sdv), now)

		h.cstart += TWIN
		h.ccount = 0
		h.ctotal = 0
		h.ctota2 = 0
	}
}

func (h *HWAB) Deviation(val float64) (float64, bool) {

	if !h.valid || h.dn == 0 {
		return 0, false
	}

	dev := (h.yn - float32(val)) / h.dn

	return float64(dev), true
}

// ################################################################

func (h *HWAB) add(ave float32, sdv float32, now int64) {

	age := now - h.Created
	si := int(h.cstart/TWIN) % h.buckets

	if age < int64(h.cf.Period) {
		h.bootstrap1(ave, sdv, si)
		return
	}
	if age < int64(2*h.cf.Period) {
		h.bootstrap2(ave, sdv, si)
		return
	}

	h.hw(ave, sdv, si)
}

// very first period - initialize
func (h *HWAB) bootstrap1(ave float32, sdv float32, si int) {

	c := h.C[si]
	d := h.D[si]

	if c == 0 {
		c = ave
		d = sdv
	}

	at := h.cf.Alpha*ave + (1-h.cf.Alpha)*c
	dx := sdv + fabs(ave-c)
	dt := h.cf.Alpha*dx + (1-h.cf.Alpha)*d

	si = (si + 1) % h.buckets
	h.A = 0
	h.B = 0
	h.C[si] = at
	h.D[si] = dt
	h.yn = at
	h.dn = dx

	h.mon.Debug("hwab/b1 a %f, d %f => %f", at, dt, h.yn)
}

// 2nd period
func (h *HWAB) bootstrap2(ave float32, sdv float32, si int) {

	sp := (si + h.buckets - 1) % h.buckets
	c := h.C[si]
	d := h.D[si]
	y := h.A + h.B + c

	cp := h.C[sp]
	dp := h.D[sp]
	if c == 0 && cp != 0 {
		// missing data - something stopped during boot phase, patch hole
		c = cp
		d = dp
	}

	at := h.cf.Alpha*(ave-c) + (1-h.cf.Alpha)*(h.A)
	ct := h.cf.Gamma*(ave-at) + (1-h.cf.Gamma)*c
	dt := h.cf.Gamma*(fabs(ave-y)+sdv) + (1-h.cf.Gamma)*d

	si = (si + 1) % h.buckets
	cn := h.C[si]
	h.yn = at + cn
	h.dn = dt

	h.A = at
	h.B = 0
	h.C[si] = ct
	h.D[si] = dt

	h.mon.Debug("hwab/b2 a %f, c %f, d %f => %f", at, ct, dt, h.yn)

}

func (h *HWAB) hw(ave float32, sdv float32, si int) {

	sp := (si + h.buckets - 1) % h.buckets
	c := h.C[si]
	d := h.D[si]
	y := h.A + h.B + c

	cp := h.C[sp]
	dp := h.D[sp]
	if c == 0 && cp != 0 {
		// missing data - something stopped during boot phase, patch hole
		c = cp
		d = dp
	}

	at := h.cf.Alpha*(ave-c) + (1-h.cf.Alpha)*(h.A+h.B)
	bt := h.cf.Beta*(at-h.A) + (1-h.cf.Beta)*h.B
	ct := h.cf.Gamma*(ave-at) + (1-h.cf.Gamma)*c
	dt := h.cf.Gamma*(fabs(ave-y)+sdv) + (1-h.cf.Gamma)*d

	si = (si + 1) % h.buckets
	cn := h.C[si]
	h.yn = at + bt + cn
	h.dn = dt
	h.valid = true

	h.A = at
	h.B = bt
	h.C[si] = ct
	h.D[si] = dt

	h.smooth(si)
	if si == 0 {
		h.normalize()
	}

	h.mon.Debug("hwab a %f, b %f, c %f, d %f => %f", at, bt, ct, dt, h.yn)
}

func (h *HWAB) smooth(si int) {

	var ctot float32
	var dtot float32

	for i := -2; i <= 2; i++ {
		sj := (si + i + h.buckets) % h.buckets
		ctot += h.C[sj]
		dtot += h.D[sj]
	}

	h.C[si] = ctot / 5
	h.D[si] = dtot / 5
}

func (h *HWAB) normalize() {

	// move a towards zero-mean
	for i := 0; i < h.buckets; i++ {
		h.C[i] -= h.A
	}

	h.A = 0
}

func fabs(x float32) float32 {
	return float32(math.Abs(float64(x)))
}
