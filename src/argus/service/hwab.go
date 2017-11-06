// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Sep-02 14:28 (EDT)
// Function:

package service

import (
	"math"
	"sync"

	"argus/clock"
	"argus/configure"
	"argus/diag"
	"argus/monel"
)

type HwabConf struct {
	Alpha  float32
	Beta   float32
	Gamma  float32
	Period int `cfconv:"timespec"`
}

type HWAB struct {
	cf       HwabConf
	mon      *monel.M
	lock     sync.RWMutex
	Pbuckets int
	buckets  int
	Created  int64
	Count    int
	cstart   int64
	ctotal   float64
	ctota2   float64
	ccount   int
	yn       float32
	dn       float32
	A        float32
	B        float32
	C        []float32
	D        []float32
}

var dh = diag.Logger("hwab")

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

	now := clock.Unix()
	wnow := (now / TWIN) * TWIN

	h := &HWAB{
		Created: wnow,
		cstart:  wnow,
		mon:     s.mon,
	}
	h.cf = hwabdefaults
	s.p.Hwab = h

	conf.InitFromConfig(&h.cf, "service", "hwab_")

	h.make()

	return nil
}

func (h *HWAB) make() {

	if h.cf.Period < PERIOD_MIN {
		h.cf.Period = PERIOD_MIN
	}
	if h.cf.Period > PERIOD_MAX {
		h.cf.Period = PERIOD_MAX
	}

	h.buckets = h.cf.Period / TWIN

	h.C = make([]float32, h.buckets)
	h.D = make([]float32, h.buckets)
}

// after persisted data is loaded
func (h *HWAB) Init(s *Service) error {

	if h.Pbuckets != 0 && h.Pbuckets != h.buckets {
		// period was changed
		h.Reset()
	}

	h.Pbuckets = h.buckets

	if h.buckets < len(h.C) {
		h.C = h.C[:h.buckets]
		h.D = h.D[:h.buckets]
	}

	h.predict()
	return nil
}

func (h *HWAB) Add(val float64) {

	h.mon.Debug("hwab add")
	now := clock.Unix()

	h.AddAt(val, now)
}

func (h *HWAB) AddAt(val float64, now int64) {

	// resample from service.Freq -> TWIN
	h.lock.Lock()
	defer h.lock.Unlock()

	h.ccount++
	h.ctotal += val
	h.ctota2 += val * val

	if h.cstart+TWIN > now {
		return
	}

	ave := h.ctotal / float64(h.ccount)
	sdv := math.Sqrt(h.ctota2/float64(h.ccount) - ave*ave)
	if sdv == 0 {
		sdv = 0.5 * math.Sqrt(math.Abs(ave))
	}

	for h.cstart+TWIN <= now {
		h.add(float32(ave), float32(sdv), h.cstart)

		h.cstart += TWIN
		h.ccount = 0
		h.ctotal = 0
		h.ctota2 = 0
	}
}

func (h *HWAB) Deviation(val float64) (float64, bool) {

	h.lock.RLock()
	defer h.lock.RUnlock()

	if h.dn == 0 {
		return 0, false
	}

	dev := (h.yn - float32(val)) / h.dn

	return float64(dev), true
}

func (h *HWAB) Reset() {

	h.lock.Lock()
	defer h.lock.Unlock()

	h.Created = (clock.Unix() / TWIN) * TWIN

	for i := 0; i < h.buckets; i++ {
		h.C[i] = 0
		h.D[i] = 0
	}
}

// ################################################################

func (h *HWAB) predict() {

	age := clock.Unix() - h.Created
	si := h.idx(int(age / TWIN))

	c := h.C[si]
	d := h.D[si]
	h.yn = h.A + h.B + c
	h.dn = d
}

func (h *HWAB) add(ave float32, sdv float32, now int64) {

	h.Count++
	age := now - h.Created
	si := h.idx(int(age / TWIN))

	if h.Count == h.buckets {
		h.smooth()
	}
	if h.Count == 2*h.buckets {
		h.interpolate()
	}

	//dh.Debug("now %d, ave %f, sdv %f, si %d, age %d", now, ave, sdv, si, age)

	if h.Count < h.buckets {
		h.bootstrap1(ave, sdv, si)
		return
	}
	if h.Count < 2*h.buckets {
		h.bootstrap2(ave, sdv, si)
		return
	}

	if si == 0 {
		h.normalize()
	}

	h.hw(ave, sdv, si)
}

// very first period - initialize C + D
func (h *HWAB) bootstrap1(ave float32, sdv float32, si int) {

	sp := h.idx(si - 1)

	c := h.C[sp]
	d := h.D[sp]

	if d == 0 {
		c = ave
		d = sdv
	}

	at := ave
	dx := sdv + fabs(ave-c)
	dt := (dx + 3*d) / 4

	h.A = 0
	h.B = 0
	h.C[si] = at
	h.D[si] = dt
	h.yn = 2*at - c
	h.dn = dt

	dh.Debug("hwab/b1 %d: a %f std %f; at %f, dx %f, dt %f => %f # %f", si, ave, sdv, at, dx, dt, h.yn, h.dn)
	if h.mon != nil {
		h.mon.Debug("hwab/b1 a %f, d %f => %f", at, dt, h.yn)
	}
}

// 2nd period - start estimating A
func (h *HWAB) bootstrap2(ave float32, sdv float32, si int) {

	sp := h.idx(si - 1)
	sn := h.idx(si + 1)

	c := h.C[si]
	d := h.D[si]

	cp := h.C[sp]
	dp := h.D[sp]

	if d == 0 {
		// missing data - something stopped during boot phase, patch hole
		c = ave
		d = sdv

		if dp != 0 {
			d = sdv + fabs(ave-cp)
		}

		h.C[si] = c
		h.D[si] = d
	}

	y := h.A + h.B + c
	at := h.cf.Alpha*(ave-c) + (1-h.cf.Alpha)*(h.A)
	ct := h.cf.Gamma*(ave-at) + (1-h.cf.Gamma)*c
	dt := h.cf.Gamma*(fabs(ave-y)+sdv) + (1-h.cf.Gamma)*d

	h.A = at
	h.B = 0
	h.C[si] = ct
	h.D[si] = dt

	cn := h.C[sn]
	h.yn = at + cn
	h.dn = h.D[sn]

	dh.Debug("hwab/b2 %d: a %f std %f; at %f, dt %f => %f # %f", si, ave, sdv, at, dt, h.yn, h.dn)
	if h.mon != nil {
		h.mon.Debug("hwab/b2 a %f, c %f, d %f => %f", at, ct, dt, h.yn)
	}
}

func (h *HWAB) hw(ave float32, sdv float32, si int) {

	sn := h.idx(si + 1)

	c := h.C[si]
	d := h.D[si]

	y := h.A + h.B + c
	at := h.cf.Alpha*(ave-c) + (1-h.cf.Alpha)*(h.A+h.B)
	bt := h.cf.Beta*(at-h.A) + (1-h.cf.Beta)*h.B
	ct := h.cf.Gamma*(ave-at) + (1-h.cf.Gamma)*c
	dt := h.cf.Gamma*(fabs(ave-y)+sdv) + (1-h.cf.Gamma)*d

	h.A = at
	h.B = bt
	h.C[si] = ct
	h.D[si] = dt

	cn := h.C[sn]
	h.yn = at + bt + cn
	h.dn = h.D[sn]

	dh.Debug("hwab %d: a %f std %f; at %f, bt %f, dt %f => %f # %f", si, ave, sdv, at, bt, dt, h.yn, h.dn)
	if h.mon != nil {
		h.mon.Debug("hwab a %f, b %f, c %f, d %f => %f", at, bt, ct, dt, h.yn)
	}
}

func (h *HWAB) interpolate() {

	for si := 0; si < h.buckets; si++ {
		if h.D[si] != 0 {
			continue
		}
		l := 0
		r := 0
		for l = 0; l < h.buckets; l++ {
			if h.D[h.idx(si-l)] != 0 {
				break
			}
		}
		for r = 0; r < h.buckets; r++ {
			if h.D[h.idx(si+r)] != 0 {
				break
			}
		}

		h.C[si] = (h.C[h.idx(si-l)]*float32(r) + h.C[h.idx(si+r)]*float32(l)) / float32(l+r)
		h.D[si] = (h.D[h.idx(si-l)]*float32(r) + h.D[h.idx(si+r)]*float32(l)) / float32(l+r)
	}
}

func (h *HWAB) smooth() {

	c := make([]float32, h.buckets)
	d := make([]float32, h.buckets)

	for si := 0; si < h.buckets; si++ {

		var ctot float32
		var dtot float32
		var wtot float32

		for i := -2; i <= 2; i++ {
			sj := h.idx(si + i)
			w := float32(1)
			if i == 0 {
				w = 4
			}
			if h.D[sj] == 0 {
				continue
			}
			ctot += w * h.C[sj]
			dtot += w * h.D[sj]
			wtot += w
		}
		if wtot == 0 {
			continue
		}
		c[si] = ctot / wtot
		d[si] = dtot / wtot
	}

	h.C = c
	h.D = d
}

func (h *HWAB) normalize() {

	// move a towards zero-mean
	for i := 0; i < h.buckets; i++ {
		h.C[i] += h.A
	}

	h.A = 0
}

func fabs(x float32) float32 {
	return float32(math.Abs(float64(x)))
}

func (h *HWAB) idx(i int) int {

	return (i + h.buckets) % h.buckets
}
