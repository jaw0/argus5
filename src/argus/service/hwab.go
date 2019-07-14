// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Sep-02 14:28 (EDT)
// Function:

package service

import (
	"fmt"
	"math"
	"sync"

	"argus/clock"
	"argus/configure"
	"argus/monel"
	"github.com/jaw0/acgo/diag"
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
	Lap      int
	cstart   int64
	ctotal   float64
	ctota2   float64
	ccount   int
	alpha    float32
	beta     float32
	gamma    float32
	yn       float32
	dn       float32
	cn       float32
	A        float32
	B        float32
	C        []float32
	D        []float32
}

var dh = diag.Logger("hwab")

var hwabdefaults = HwabConf{
	Alpha:  0.5,
	Beta:   0.5,
	Gamma:  0.25,
	Period: 7 * 24 * 3600,
}

const (
	TWIN       = 300
	PERIOD_MIN = 2
	PERIOD_MAX = 30 * 24 * 3600
	EPSILOND   = 0.0001
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
	h.alpha = h.cf.Alpha / float32(h.buckets)
	h.beta = h.cf.Beta / float32(h.buckets)
	h.gamma = h.cf.Gamma

	h.C = make([]float32, h.buckets)
	h.D = make([]float32, h.buckets)
}

// after persisted data is loaded
func (h *HWAB) Init(s *Service) error {

	if h.Pbuckets != 0 && h.Pbuckets != h.buckets {
		// period was changed
		h.Reset()
	}
	if h.Lap == 0 && h.gap() > 5 {
		// large gap during startup
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

	dh.Debug("reset")
	h.lock.Lock()
	defer h.lock.Unlock()

	h.Created = (clock.Unix() / TWIN) * TWIN
	h.Lap = 0
	h.Count = 0

	for i := 0; i < h.buckets; i++ {
		h.C[i] = 0
		h.D[i] = 0
	}
}

func (h *HWAB) Format(f fmt.State, c rune) {
	fmt.Fprintf(f, "{size: %d, prediction: %v +/- %v, count: %d, lap: %d, a %f, b %f, c: %f}",
		h.buckets, h.yn, h.dn, h.Count, h.Lap, h.A, h.B, h.cn)
}

// ################################################################

func (h *HWAB) predict() {

	si := h.idxOfT(clock.Unix())

	h.cn = h.C[si]
	h.dn = h.D[si]
	h.yn = h.A + h.B + h.cn
}

func (h *HWAB) add(ave float32, sdv float32, now int64) {

	si := h.idxOfT(now)

	if si == 0 && h.Count > 0 {
		h.Lap++
	}

	if h.Lap == 1 && si == 0 {
		if h.checkEnough() {
			h.interpolate()
		} else {
			h.Lap = 0
			h.Count = 0
		}
	}

	if si == 0 && h.Lap > 1 {
		h.smooth()
		h.normalize()
	}

	//dh.Debug("now %d, ave %f, sdv %f, si %d, age %d", now, ave, sdv, si, age)

	switch h.Lap {
	case 0:
		h.bootstrap1(ave, sdv, si)
	case 1:
		h.bootstrap2(ave, sdv, si)
	default:
		h.hw(ave, sdv, si)
	}

	h.Count++
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
	if dx == 0 {
		dx = float32(math.Sqrt(float64(ave)))
	}
	dt := (dx + 3*d) / 4

	h.A = 0
	h.B = 0
	h.C[si] = at
	h.D[si] = dt
	h.yn = 2*at - c
	h.cn = at
	h.dn = dt

	dh.Debug("hwab/b1 %d: a %f std %f; at %f, dx %f, dt %f => %f # %f", si, ave, sdv, at, dx, dt, h.yn, h.dn)
	if h.mon != nil {
		h.mon.Debug("hwab/b1 a %f, d %f => %f", at, dt, h.yn)
	}
}

// 2nd period - start estimating A+B
func (h *HWAB) bootstrap2(ave float32, sdv float32, si int) {

	sp := h.idx(si - 1)
	sn := h.idx(si + 1)

	c := h.C[si]
	d := h.D[si]

	dp := h.D[sp]

	if d == 0 {
		// missing data - something stopped during boot phase, patch hole
		dh.Debug("missing")
		d = dp
		if dp == 0 {
			d = sdv
		}
		c = ave - h.A
	}

	y := h.B*float32(h.buckets) + c
	dx := fabs(ave-y) + sdv
	if dx == 0 {
		dx = float32(math.Sqrt(float64(ave)))
	}

	dh.Debug("sdv %f, dx %f, d %f", sdv, dx, d)
	alpha := h.alpha
	beta := 1 / float32(si+1)
	gamma := float32(0.5)

	at := alpha*(ave-c) + (1-alpha)*(h.A+h.B)
	bt := beta*(ave-c)/float32(h.buckets) + (1-beta)*h.B
	ct := ave - at - bt
	dt := gamma*dx + (1-gamma)*d

	h.A = at
	h.B = bt
	h.C[si] = ct
	h.D[si] = dt

	h.cn = ct
	h.yn = ave + bt
	h.dn = h.D[sn]

	dh.Debug("si %d, ave %f, c %f, dt %f, bt %f; val %f", si, ave, c, dt, bt, (ave-c)/float32(h.buckets))
	//dh.Debug("hwab/b2 %d: a %f std %f; at %f, dt %f => %f # %f", si, ave, sdv, at, dt, h.yn, h.dn)
	if h.mon != nil {
		h.mon.Debug("hwab/b2 a %f, c %f, d %f => %f", at, ct, dt, h.yn)
	}
}

func (h *HWAB) hw(ave float32, sdv float32, si int) {

	sn := h.idx(si + 1)

	c := h.C[si]
	d := h.D[si]

	y := h.A + h.B + c
	dx := fabs(ave-y) + sdv
	if dx == 0 {
		dx = EPSILOND
	}

	dh.Debug("d %f, sdv %f, dx %f", d, sdv, dx)
	at := h.alpha*(ave-c) + (1-h.alpha)*(h.A+h.B)
	bt := h.beta*(at-h.A) + (1-h.beta)*h.B
	ct := h.gamma*(ave-at) + (1-h.gamma)*c
	dt := h.gamma*dx + (1-h.gamma)*d

	h.A = at
	h.B = bt
	h.C[si] = ct
	h.D[si] = dt

	h.cn = h.C[sn]
	h.yn = h.A + h.B + h.cn
	h.dn = h.D[sn]

	dh.Debug("hwab %d: a %f std %f; at %f, bt %f, ct %f, dt %f => %f # %f", si, ave, sdv, at, bt, ct, dt, h.yn, h.dn)
	if h.mon != nil {
		h.mon.Debug("hwab a %f, b %f, c %f, d %f => %f", at, bt, ct, dt, h.yn)
	}
}

func (h *HWAB) interpolate() {

	li := 0

	for si := 0; si < h.buckets; si++ {
		if h.D[si] != 0 {
			li = si
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
		h.D[si] = h.D[h.idx(li)] + h.D[h.idx(si+r)]

		dh.Debug("interp [%d][%d-%d]: %f,%f -> %f", si, l, r, h.C[h.idx(si-l)], h.C[h.idx(si+r)], h.C[si])
	}
}

func (h *HWAB) smooth() {

	dh.Debug("smooth")
	c := make([]float32, h.buckets)
	d := make([]float32, h.buckets)

	wsize := h.buckets / 20
	if wsize < 2 {
		wsize = 2
	}
	sdev := float64(h.buckets) / 100 // std dev = 1%
	sdvk := -sdev * sdev * 2

	for si := 0; si < h.buckets; si++ {

		var ctot float64
		var dtot float64
		var wtot float64

		for i := -wsize; i <= wsize; i++ {
			// gaussian
			w := math.Exp(float64(i*i) / sdvk)

			sj := h.idx(si + i)
			if h.D[sj] == 0 {
				continue
			}
			ctot += w * float64(h.C[sj])
			dtot += w * float64(h.D[sj])
			wtot += w
		}
		if wtot == 0 {
			continue
		}
		c[si] = float32(ctot / wtot)
		d[si] = float32(dtot / wtot)
	}

	h.C = c
	h.D = d
}

func (h *HWAB) normalize() {

	var ctot float32
	for i := 0; i < h.buckets; i++ {
		ctot += h.C[i]
	}

	// make C zero mean
	cm := ctot / float32(h.buckets)
	for i := 0; i < h.buckets; i++ {
		h.C[i] -= cm
	}
	h.A += cm

}

// do we have enough data?
func (h *HWAB) checkEnough() bool {

	dh.Debug("check %f %f", h.D[0], h.D[h.buckets-1])
	if h.D[0] == 0 || h.D[h.buckets-1] == 0 {
		return false
	}
	return true
}

func fabs(x float32) float32 {
	return float32(math.Abs(float64(x)))
}

func (h *HWAB) idxOfT(now int64) int {
	age := now - h.Created
	return h.idx(int(age / TWIN))
}

func (h *HWAB) idx(i int) int {
	return (i + h.buckets) % h.buckets
}

func (h *HWAB) gap() int {
	return h.idxOfT(clock.Unix()) - h.Count
}
