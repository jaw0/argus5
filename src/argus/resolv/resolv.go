// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Aug-29 18:32 (EDT)
// Function: async dns resolver

package resolv

import (
	"expvar"
	"net"
	"sync"
	"time"

	"github.com/miekg/dns"

	"argus/clock"
	"argus/config"
	"argus/diag"
	"argus/lfsr"
)

type cacheRes struct {
	addr string
	ipv  int
}

type cacheE struct {
	lock     sync.RWMutex
	name     string
	fqdn     string
	result   []cacheRes
	accum    []cacheRes // next set of results currently being collected
	expire   int64
	neededby int64
	updated  int64
	created  int64
	underway bool
	failed   bool
	pend     map[int]bool // accessed only by the goro that set underway
}

const (
	MAXQUERIES   = 32
	SOONER       = int64(10 * time.Second)
	QUERYTIMEOUT = int64(2 * time.Second)
	SERVERDEAD   = int64(10 * time.Second)
	TOOLONG      = int64(300 * time.Second)
	SENDDELAY    = time.Millisecond // do not kill friendly nameserver
	TTL_MIN      = 60
	TTL_MAX      = 1209600 // 2 weeks
	TTL_ERR      = int64(60 * time.Second)
	TRIES        = 3
)

var lock sync.RWMutex
var cache = make(map[string]*cacheE)
var todo = make(chan string, 1000)
var stop = make(chan struct{})
var done sync.WaitGroup
var dl = diag.Logger("resolv")

var ResolvQueue = expvar.NewInt("resolvqueue")
var ResolvIdle = expvar.NewInt("resolvidle")
var idlelock sync.Mutex
var nIdle = 0

func Lookup(name string, ipv int) (string, int, bool) {

	e := getCache(name)

	if e == nil {
		lookup(name)
		return "", 0, false
	}

	e.lock.RLock()
	defer e.lock.RUnlock()

	if e.expire < clock.Nano() && !e.underway {
		lookup(name)
	}

	// first matching result
	for i, _ := range e.result {
		r := &e.result[i]

		if ipv == 0 || ipv == r.ipv {
			return r.addr, r.ipv, false
		}
	}

	return "", 0, e.failed
}

func WillNeedIn(name string, secs int) {

	t := clock.Nano() + int64(secs)*int64(time.Second)

	e := getCache(name)

	if e == nil {
		lookup(name)
		return
	}

	e.lock.RLock()
	defer e.lock.RUnlock()

	if t < e.expire {
		return
	}

	if e.neededby == 0 || e.neededby > t {
		e.neededby = t
	}
}

func Init() {

	cf := config.Cf()
	nwork := cf.Resolv_maxrun
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

func lookup(name string) {

	select {
	case todo <- name:
		break
	default:
	}
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
			prefetch()
			ResolvQueue.Set(int64(len(todo)))
			ResolvIdle.Set(int64(numIdle()))
		}
	}
}

func prefetch() {

	now := clock.Nano()
	lock.RLock()
	defer lock.RUnlock()

	for n, c := range cache {
		c.lock.RLock()
		if c.neededby > 0 && c.neededby-SOONER < now && now < c.expire && !c.underway {
			c.neededby = 0
			lookup(n)
		}
		c.lock.RUnlock()
	}
}

// ################################################################

func proceedWith(name string) (bool, string, map[int]bool) {

	now := clock.Nano()

	e := getCache(name)

	if e == nil {
		e = &cacheE{name: name, pend: make(map[int]bool), created: now}
		dl.Debug("new entry: %#v", e)
		setCache(name, e)
	}

	e.lock.Lock()
	defer e.lock.Unlock()

	if e.underway {
		return false, "", nil
	}

	if e.expire > now {
		return false, "", nil
	}

	e.underway = true
	e.accum = nil

	return true, e.fqdn, e.pend
}

func doneWith(name string, qid int) {

	e := getCache(name)

	if e == nil {
		return
	}

	e.lock.Lock()
	defer e.lock.Unlock()

	delete(e.pend, qid)

	if len(e.pend) == 0 {
		now := clock.Nano()
		e.underway = false

		if len(e.accum) == 0 {
			e.expire = now + TTL_ERR
			// keep old results
			e.failed = true
		} else {
			e.result = e.accum
		}
		e.updated = now
	}
}

func haveValid(name string) bool {

	now := clock.Nano()

	e := getCache(name)

	if e == nil {
		return false
	}

	e.lock.RLock()
	defer e.lock.RUnlock()

	if e.failed || e.expire <= now {
		return false
	}

	return true
}

func cacheAnswer(name string, fqdn string, addr string, ipv int, ttl int) {

	dl.Debug("caching answer %s -> %s -> %s", name, fqdn, addr)

	e := getCache(name)

	if e == nil {
		return
	}

	if ttl < TTL_MIN {
		ttl = TTL_MIN
	}
	if ttl > TTL_MAX {
		ttl = TTL_MAX
	}

	e.lock.Lock()
	defer e.lock.Unlock()

	e.expire = clock.Nano() + int64(ttl*1000000)

	if e.expire > e.neededby {
		e.neededby = 0
	}

	if addr == "" {
		e.failed = true
		return
	}

	e.failed = false
	e.fqdn = fqdn

	e.accum = append(e.accum, cacheRes{addr, ipv})

}

// ################################################################

type result struct {
	buf []byte
}

type pendQ struct {
	name  string // original name
	qname string // current name
	zname string // current fqdn
	start int64
	tries int
	pend  map[int]bool
}

type workstate struct {
	nqueries int
	qid      int
	search   []string
	server   []*net.UDPAddr
	nsn      int
	pending  map[int]*pendQ
	lastrcv  int64
	sock     *net.UDPConn
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

	reschan := make(chan *result, MAXQUERIES)
	w := &workstate{
		pending: make(map[int]*pendQ),
		lastrcv: clock.Nano(),
		sock:    sock,
	}
	w.configure()

	go receiver(sock, reschan)

	for {
		amIdle(true)

		if w.nqueries < MAXQUERIES {
			select {
			case <-stop:
				return
			case res := <-reschan:
				amIdle(false)
				w.processResult(res)
			case name := <-todo:
				amIdle(false)
				can, fqdn, m := proceedWith(name)
				if can {
					w.sendQuery(name, fqdn, m)
				}
			case <-tock.C:
				amIdle(false)
				w.timeouts()
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

func (w *workstate) sendQuery(name string, qname string, pend map[int]bool) {

	dl.Debug("query: name %s fqdn %s", name, qname)

	now := clock.Nano()

	search := []string{""}

	if qname == "" {
		qname = name
		if name[len(name)-1] != '.' {
			search = w.search
		}
	} else if qname[len(qname)-1] != '.' {
		qname = qname + "."
	}

	for _, s := range search {
		zone := qname + s

		p := &pendQ{name, qname, zone, now, TRIES, pend}
		w.sendQueryZone(p)
	}
}

func (w *workstate) sendQueryZone(pq *pendQ) {

	dst := w.server[w.nsn]

	q1 := w.nextqid()
	q2 := w.nextqid()
	dl.Debug("sending %d %s A", q1, pq.zname)
	dl.Debug("sending %d %s AAAA", q2, pq.zname)

	w.pending[q1] = pq
	w.pending[q2] = pq
	pq.pend[q1] = true
	pq.pend[q2] = true

	w.sock.WriteToUDP(encode(pq.zname, dns.TypeA, q1), dst)
	time.Sleep(SENDDELAY)
	w.sock.WriteToUDP(encode(pq.zname, dns.TypeAAAA, q2), dst)
	time.Sleep(SENDDELAY)
	w.nqueries += 2
	pq.tries--
}

func (w *workstate) nextqid() int {

	for {
		w.qid = lfsr.Next16(w.qid)

		if w.qid == 0 {
			w.qid++
		}
		if w.pending[w.qid] == nil {
			break
		}
	}

	return w.qid
}

func (w *workstate) timeouts() {

	now := clock.Nano()
	nto := 0

	for qid, p := range w.pending {

		if p.start+QUERYTIMEOUT < now {
			dl.Debug("no response: %d, %s", qid, p.qname)

			delete(w.pending, qid)
			doneWith(p.name, qid)
			w.nqueries--
			nto++

			if p.tries > 0 {
				w.sendQueryZone(p)
			}
		}
	}

	// switch to another server?
	if nto > 0 && w.lastrcv+SERVERDEAD < now {
		diag.Verbose("nameserver down? switching to next")
		w.nsn = (w.nsn + 1) % len(w.server)
		w.lastrcv = now
	}

}

func receiver(sock *net.UDPConn, rc chan *result) {

	for {
		buf := make([]byte, 1024)
		size, addr, err := sock.ReadFromUDP(buf)

		if err != nil {
			dl.Debug("recv err %v", err)
			return // XXX
		}

		dl.Debug("recv %d from %s", size, addr)
		rc <- &result{buf: buf[:size]}
	}
}

func (w *workstate) processResult(res *result) {

	msg := &dns.Msg{}
	msg.Unpack(res.buf)

	qid := int(msg.Id)

	if qid == 0 || !msg.Response {
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
	w.lastrcv = now
	delete(w.pending, qid)
	w.nqueries--

	dt := now - pq.start

	for _, ans := range msg.Answer {
		h := ans.Header()
		dl.Debug("rcv [%.1f msec]> %s", float32(dt)/1000000, ans.String())
		switch ans := ans.(type) {
		case *dns.A:
			cacheAnswer(pq.name, pq.zname, ans.A.String(), 4, int(h.Ttl))
		case *dns.AAAA:
			cacheAnswer(pq.name, pq.zname, ans.AAAA.String(), 6, int(h.Ttl))
		}
	}

	doneWith(pq.name, qid)
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

func getCache(name string) *cacheE {
	lock.RLock()
	defer lock.RUnlock()
	return cache[name]
}
func setCache(name string, e *cacheE) {
	lock.Lock()
	defer lock.Unlock()
	cache[name] = e
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
