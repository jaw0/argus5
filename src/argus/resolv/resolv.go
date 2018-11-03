// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Aug-29 18:32 (EDT)
// Function: async dns resolver

package resolv

import (
	"expvar"
	"math/rand"
	"net"
	"sync"
	"time"

	"github.com/miekg/dns"

	"argus/clock"
	"argus/config"
	"github.com/jaw0/acgo/diag"
	"argus/lfsr"
)

const (
	NUMWORKERS   = 5
	MAXQUERIES   = 256
	NANOSECONDS  = 1000000000
	QUERYTIMEOUT = int64(4 * NANOSECONDS)
	SERVERDEAD   = int64(10 * NANOSECONDS)
	SENDDELAY    = 1 * time.Millisecond
	TTL_MIN      = 60
	TTL_MAX      = 14 * 24 * 3600
	TTL_ERR      = 120
	SOONER       = 2 // seconds
	TRIES        = 3
	XDK          = 40
)

type queryReq struct {
	name  string
	prefm int
}

var lock sync.RWMutex
var todo = make(chan queryReq, 1000)
var stop = make(chan struct{})
var done sync.WaitGroup
var dl = diag.Logger("resolv")

var ResolvQueue = expvar.NewInt("resolvqueue")
var ResolvIdle = expvar.NewInt("resolvidle")
var ResolvQuery = expvar.NewInt("resolvqueries")
var ResolvDrops = expvar.NewInt("resolvdrops")
var ResolvTouts = expvar.NewInt("resolvtimeouts")
var idlelock sync.Mutex
var nIdle = 0

func Init() {

	resolvConfigure()

	cf := config.Cf()
	nwork := cf.Resolv_maxrun

	if nwork == 0 {
		nwork = NUMWORKERS
	}
	if nwork < 1 {
		nwork = 1
	}

	for i := 0; i < nwork; i++ {
		done.Add(1)
		go worker()
	}

	done.Add(1)
	go janitor()

	diag.Verbose("resolv running %d workers", nwork)
}

func Stop() {
	close(stop)
	done.Wait()
}

// ################################################################

func query(name string, prefm int) bool {

	select {
	case todo <- queryReq{name, prefm}:
		return true
	default:
		ResolvDrops.Add(1)
	}
	return false
}

func janitor() {

	defer done.Done()

	tock := time.NewTicker(5 * time.Second)
	defer tock.Stop()

	for {
		select {
		case <-stop:
			return
		case <-tock.C:
			cacheMaint()
			ResolvQueue.Set(int64(len(todo)))
			ResolvIdle.Set(int64(numIdle()))
		}
	}
}

//################################################################

type result struct {
	buf []byte
}

type pendQ struct {
	name      string // original name
	zname     string // current fqdn
	start     int64
	tries     int
	prefm     int
	underway4 int
	underway6 int
	res       *cacheResult
}

type workstate struct {
	nqueries   int     // number currently underway
	maxqueries float32 // max underway
	xdelay     float32 // average response delay
	nsn        int     // current nameserver index
	qid        int
	white      int
	lastrcv    int64
	pending    map[int]*pendQ
	sock       *net.UDPConn
}

func worker() {

	defer done.Done()

	// open socket
	sock, err := net.ListenUDP("udp", nil)
	if err != nil {
		diag.Fatal("could not open udp socket: %v", err)
	}
	defer sock.Close()

	tock := time.NewTicker(time.Second)
	defer tock.Stop()

	reschan := make(chan *result, 2*MAXQUERIES)
	w := &workstate{
		maxqueries: 1,
		qid:        rand.Intn(65535),
		white:      rand.Intn(65535),
		pending:    make(map[int]*pendQ),
		lastrcv:    clock.Nano(),
		sock:       sock,
	}

	go receiver(sock, reschan)

	xdelay := clock.Nano()

	for {
		amIdle(true)
		now := clock.Nano()

		if w.nqueries < int(w.maxqueries) && now >= xdelay {
			select {
			case <-stop:
				return
			case res := <-reschan:
				amIdle(false)
				w.processResult(res)
			case <-tock.C:
				amIdle(false)
				w.timeouts()
			case req := <-todo:
				amIdle(false)
				w.makeQueries(req.name, req.prefm)
				d := w.xdelay
				xdelay = now + int64(d)
			}
		} else {
			// too many outstanding, don't process new queries
			select {
			case <-stop:
				return
			case res := <-reschan:
				amIdle(false)
				w.processResult(res)
			case <-tock.C:
				amIdle(false)
				w.timeouts()
			}
		}
	}
}

func (w *workstate) makeQueries(name string, prefm int) {

	dl.Debug("query: %d name %s", prefm, name)

	now := clock.Nano()

	search := []string{""}

	if name[len(name)-1] != '.' {
		search = cfSearch
	}

	for _, s := range search {
		zone := name + s

		p := &pendQ{name: name, zname: zone, start: now, tries: TRIES, prefm: prefm}
		w.sendQueryZone(p)
	}
}

func (w *workstate) sendQueryZone(pq *pendQ) {

	if pq.prefm&F_IP4 != 0 {
		q1 := w.nextqid()
		pq.underway4 = q1
		dl.Debug("sending ->NS[%d] %s: %d A", w.nsn, pq.zname, q1)
		w.sendQuery(pq, pq.zname, dns.TypeA, q1, w.nsn)
	}
	if pq.prefm&F_IP6 != 0 {
		q2 := w.nextqid()
		pq.underway4 = q2
		dl.Debug("sending ->NS[%d] %s: %d AAAA", w.nsn, pq.zname, q2)
		w.sendQuery(pq, pq.zname, dns.TypeAAAA, q2, w.nsn)
	}

	pq.tries--
}

func (w *workstate) sendQuery(pq *pendQ, zone string, typ uint16, qid int, nsn int) {

	dst := cfServer[nsn]

	w.pending[qid] = pq
	w.sock.WriteToUDP(encode(zone, typ, qid), dst)
	time.Sleep(SENDDELAY)
	w.nqueries++
	ResolvQuery.Add(1)
}

func receiver(sock *net.UDPConn, rc chan *result) {

	for {
		buf := make([]byte, 1024)
		size, addr, err := sock.ReadFromUDP(buf)

		if err != nil {
			dl.Debug("recv err %v", err)
			return
		}

		dl.Debug("recv %d from %s", size, addr)
		rc <- &result{buf: buf[:size]}
	}
}

func (w *workstate) processResult(res *result) {

	msg := &dns.Msg{}
	msg.Unpack(res.buf)

	qid := int(msg.Id)

	dl.Debug("rcv qid %d", qid)

	if !msg.Response {
		// invalid response
		return
	}

	pq := w.pending[qid]
	if pq == nil {
		// dupe response?
		return
	}

	// verify this is answering the expected question
	if len(msg.Question) == 0 || msg.Question[0].Name != pq.zname {
		return
	}

	now := clock.Nano()
	dt := now - pq.start
	w.addDelay(dt / int64(w.maxqueries))

	w.lastrcv = now
	delete(w.pending, qid)
	w.nqueries--
	w.addMax(w.maxqueries + 1)

	if pq.underway4 == qid {
		pq.underway4 = 0
	}
	if pq.underway6 == qid {
		pq.underway6 = 0
	}

	for _, ans := range msg.Answer {
		h := ans.Header()
		ttl := int(h.Ttl)
		dl.Debug("rcv [%.1f msec]> %s", float32(dt)/1000000, ans.String())

		if pq.res == nil {
			pq.res = &cacheResult{fqdn: pq.zname, ttl: ttl}
		}
		switch ans := ans.(type) {
		case *dns.A:
			pq.res.addrv4 = append(pq.res.addrv4, ans.A.String())
		case *dns.AAAA:
			pq.res.addrv6 = append(pq.res.addrv6, ans.AAAA.String())
		}
	}

	if pq.underway4 == 0 && pq.underway6 == 0 {
		pq.done()
	}
}

func (w *workstate) timeouts() {

	now := clock.Nano()
	nto := 0

	dl.Debug("nq %d, max %.2f; xd %.2f", w.nqueries, w.maxqueries, w.xdelay)

	for qid, p := range w.pending {
		if p.start+QUERYTIMEOUT < now {
			dl.Debug("no response: %d, %s", qid, p.zname)

			delete(w.pending, qid)
			w.nqueries--
			nto++
			ResolvTouts.Add(1)
			w.addDelay(QUERYTIMEOUT)
			w.addMax(w.maxqueries / 2)

			if p.tries > 0 {
				w.sendQueryZone(p)
			} else {
				p.underway4 = 0
				p.underway6 = 0
				p.done()
			}
		}
	}

	// switch to another server?
	if nto > 0 && w.lastrcv+SERVERDEAD < now {
		diag.Verbose("nameserver down? switching to next")
		w.nsn = (w.nsn + 1) % len(cfServer)
		w.lastrcv = now
	}
}

func (pq *pendQ) done() {
	dl.Debug("%s done; %v", pq.name, pq)

	if pq.res == nil {
		pq.res = &cacheResult{fqdn: pq.name, ttl: TTL_ERR + rand.Intn(TTL_ERR/10)}
	}

	if pq.res.ttl < TTL_MIN {
		pq.res.ttl = TTL_MIN
	}
	if pq.res.ttl > TTL_MAX {
		pq.res.ttl = TTL_MAX
	}

	// avoid thundering herd
	pq.res.ttl -= rand.Intn(pq.res.ttl / 10)

	pq.res.expires = clock.Nano() + int64(pq.res.ttl*NANOSECONDS)
	Result(pq.name, pq.prefm, pq.res)
}

func (w *workstate) nextqid() int {

	for {
		w.qid = lfsr.Next16(w.qid)

		if w.qid == 1 {
			w.white = rand.Intn(65535)
		}
		id := w.qid ^ w.white
		if id == 0 {
			continue
		}
		if w.pending[id] == nil {
			return id
		}
	}
}

func (w *workstate) addDelay(dt int64) {
	w.xdelay = (XDK*w.xdelay + float32(dt)) / (XDK + 1)
}

func (w *workstate) addMax(n float32) {
	w.maxqueries = (XDK*w.maxqueries + n) / (XDK + 1)
	if w.maxqueries < 1 {
		w.maxqueries = 1
	}
	if w.maxqueries > MAXQUERIES {
		w.maxqueries = MAXQUERIES
	}
}

func encode(zone string, qtype uint16, qid int) []byte {

	req := dns.Msg{
		Question: []dns.Question{dns.Question{zone, qtype, dns.ClassINET}},
	}

	req.Id = uint16(qid)
	req.RecursionDesired = true

	buf, err := req.Pack()
	if err != nil {
		diag.Verbose("dns error: %#v -> %v", &req, err)
		return nil
	}

	return buf
}

func amIdle(y bool) {

	idlelock.Lock()
	defer idlelock.Unlock()
	if y {
		nIdle++
	} else {
		nIdle--
	}
}
func numIdle() int {
	idlelock.Lock()
	defer idlelock.Unlock()
	return nIdle
}
