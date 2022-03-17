// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Sep-19 20:56 (EDT)
// Function: dns testing

package dns

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/miekg/dns"

	"argus.domain/argus/configure"
	"github.com/jaw0/acgo/diag"
	"argus.domain/argus/lfsr"
	"argus.domain/argus/resolv"
	"argus.domain/argus/service"
)

type Conf struct {
	Port    int
	Zone    string
	Query   string
	Class   string
	Recurse bool
	Test    string
	Proto   string
}

type DNS struct {
	S    *service.Service
	Cf   Conf
	Ip   *resolv.IP
	Addr string // for debugging
	msg  *dns.Msg
}

var dl = diag.Logger("dns")

var defaults = map[string]*Conf{
	"Domain": &Conf{Query: "SOA", Test: "authok", Class: "IN"},
	"DNS":    &Conf{Query: "STAT", Class: "IN"},
	"DNSQ":   &Conf{Query: "NS", Test: "noerror", Recurse: true, Class: "IN", Zone: "."},
	"Serial": &Conf{Query: "SOA", Test: "serial", Recurse: true, Class: "IN"},
	"BIND":   &Conf{Query: "TXT", Class: "CH", Test: "answer", Zone: "version.bind"},
}

func init() {
	// register with service factory
	service.Register("TCP/DNS", New)
	service.Register("UDP/DNS", New)
	service.Register("TCP/Domain", New)
	service.Register("UDP/Domain", New)
	service.Register("DNS", New)
	service.Register("Domain", New)
}

func New(conf *configure.CF, s *service.Service) service.Monitor {

	d := &DNS{}
	d.S = s
	d.Cf.Class = "IN"

	return d
}

func (d *DNS) PreConfig(conf *configure.CF, s *service.Service) error {
	return nil
}
func (d *DNS) Config(conf *configure.CF, s *service.Service) error {

	if err := configDNS(&d.Cf, conf); err != nil {
		return err
	}

	ip, err := resolv.Config(conf)
	if err != nil {
		return err
	}
	d.Ip = ip

	if d.Cf.Port == 0 {
		d.Cf.Port = 53
	}

	d.buildPacket()
	d.determineNames(conf, s)

	return nil
}

// ################################################################

func configDNS(d *Conf, conf *configure.CF) error {

	// parse service descr
	name := conf.Name
	proto := "udp"

	if strings.HasPrefix(name, "TCP") {
		proto = "tcp"
	}

	name = strings.TrimPrefix(name, "TCP/")
	name = strings.TrimPrefix(name, "UDP/")
	ns := strings.SplitN(name, "/", 3)

	dl.Debug("name %s; ns %v", name, ns)
	var n, t, z string
	n = ns[0]
	if len(ns) > 1 {
		t = ns[1]
	}
	if len(ns) > 2 {
		z = ns[3]
	}
	if defaults[t] == nil && z == "" {
		// no QUERY specified: DNS/zone
		z = t
		t = ""
	}
	if t == "" {
		t = n
	}
	dl.Debug("n %s t %s z %s", n, t, z)

	if def := defaults[t]; def != nil {
		// set defaults from table
		*d = *def
		dl.Debug("defaults: %#v", d)
	} else if t != "" {
		// if specified as DNS/QUERY[/zone] set defaults as:
		d.Query = t
		d.Test = "answer"
		d.Recurse = true
		dl.Debug("query: %#v", d)
	}
	// else { query/recurse/test need to be specified explicitly }

	if z != "" {
		d.Zone = z
	}
	d.Proto = proto

	conf.InitFromConfig(d, "DNS", "")

	if d.Zone == "" {
		d.Zone = "."
	}
	if d.Zone[len(d.Zone)-1] != '.' {
		d.Zone += "."
	}

	dl.Debug("config: %#v", d)
	return nil
}

func (d *DNS) determineNames(conf *configure.CF, s *service.Service) {

	dcf := &d.Cf
	label := conf.Name
	label = strings.TrimPrefix(label, "TCP/")
	label = strings.TrimPrefix(label, "UDP/")

	if dcf.Zone != "" && dcf.Zone != "." {
		label = dcf.Zone
	}

	host := d.Ip.Hostname()
	friendly := "DNS for " + dcf.Zone + " on " + host

	uname := ""
	if dcf.Proto == "tcp" {
		uname = "TCPDNS_"
	} else {
		uname = "DNS_"
	}

	if dcf.Zone != "" && dcf.Zone != "." {
		uname += dcf.Zone + "_"
	}
	uname += dcf.Query + "_"
	if dcf.Test != "" && dcf.Test != "none" {
		uname += dcf.Test + "_"
	}

	uname += host

	s.SetNames(uname, label, friendly)
}

func (d *DNS) buildPacket() {

	qtype := dns.StringToType[d.Cf.Query]
	class := dns.StringToClass[d.Cf.Class]

	req := dns.Msg{
		Question: []dns.Question{dns.Question{d.Cf.Zone, qtype, class}},
	}

	req.Id = uint16(rand.Intn(65535))
	req.RecursionDesired = d.Cf.Recurse

	if d.Cf.Query == "STAT" {
		req.Opcode = dns.OpcodeStatus
	}

	d.msg = &req

	dl.Debug("dns request: %s", req.String())
}

func (d *DNS) nextQid() {
	d.msg.Id = uint16(lfsr.Next16(int(d.msg.Id)))
}

func (d *DNS) Init() error {
	return nil
}
func (d *DNS) Priority() bool {
	return false
}

func (d *DNS) Hostname() string {
	return d.Ip.Hostname()
}
func (d *DNS) Recycle() {
}
func (d *DNS) Abort() {
}
func (d *DNS) DoneConfig() {
}

// ################################################################

func (d *DNS) Start(s *service.Service) {

	s.Debug("dns start")
	defer s.Done()

	addr, fail := d.Ip.AddrWB()
	if fail {
		s.FailNow("cannot resolve hostname")
		return
	}
	if addr == "" {
		s.Debug("hostname still resolving")
		return
	}

	d.Ip.WillNeedIn(s.Cf.Frequency)
	addrport := fmt.Sprintf("%s:%d", addr, d.Cf.Port)
	d.Addr = addrport
	s.Debug("connecting to %s/%s", d.Cf.Proto, addrport)

	d.nextQid()
	timeout := time.Duration(d.S.Cf.Timeout) * time.Second

	client := &dns.Client{
		Net:     d.Cf.Proto,
		UDPSize: 1024,
		Timeout: timeout,
	}

	resp, _, err := client.Exchange(d.msg, addrport)
	if resp == nil {
		s.Debug("error: %v", err)
		s.Fail("dns query failed")
		d.Ip.TryAnother()
		return
	}
	if err != nil {
		s.Debug("error: %v", err)
		// STAT responses, truncated responses, etc return errTruncated
		// as long as we have a resp, keep going...
	}

	s.Debug("dns resp %s", resp.String())

	d.testResponse(s, resp)
}

func (d *DNS) testResponse(s *service.Service, resp *dns.Msg) {

	test := d.Cf.Test

	if test == "" || test == "none" {
		// recvd response => up
		s.Pass()
		return
	}

	// everything else checks error field
	ec := resp.Rcode
	if ec != 0 {
		s.Debug("DNS Error %s (RCODE=%d)", dns.RcodeToString[ec], ec)
		s.FailReady("DNS Error")
		return
	}

	if test == "noerror" {
		// no error => up
		s.Pass()
		return
	}

	if test == "authok" {
		// He looked at the Gryphon as if he thought it had some kind of authority over Alice.
		//    -- Alice in Wonderland

		if resp.Authoritative {
			s.Pass()
		} else {
			s.FailReady("DNS Error - Non-Authoratative Response")
		}
		return
	}

	if test == "nanswers" {
		// number of answers returned
		checkIntValue(s, len(resp.Answer))
		return
	}
	if test == "nauthority" {
		// number of ns records returned
		checkIntValue(s, len(resp.Ns))
		return
	}
	if test == "nadditional" {
		// number of additional records returned
		checkIntValue(s, len(resp.Extra))
		return
	}

	if test == "serial" {
		for _, rr := range resp.Answer {
			h := rr.Header()
			if h.Rrtype != dns.TypeSOA {
				continue
			}
			switch soa := rr.(type) {
			case *dns.SOA:
				checkIntValue(s, int(soa.Serial))
			default:
				s.FailReady("Invalid SOA")
			}
			return
		}
		s.FailReady("No SOA")
		return
	}

	// anything else: test the answers
	// we may have multiple answers, concatenate them with \n

	res := ""

	if test == "answer" || test == "aaa" {
		for _, rr := range resp.Answer {
			res += rr.String() + "\n"
		}
	}
	if test == "authority" || test == "aaa" {
		for _, rr := range resp.Ns {
			res += rr.String() + "\n"
		}
	}
	if test == "additional" || test == "aaa" {
		for _, rr := range resp.Extra {
			res += rr.String() + "\n"
		}
	}

	s.CheckValue(res, "dns")
}

func checkIntValue(s *service.Service, n int) {
	s.CheckValue(fmt.Sprintf("%d", n), "dns")
}

func (t *DNS) DumpInfo() map[string]interface{} {
	return map[string]interface{}{
		"service/ip/CF":    &t.Ip.Cf,
		"service/ip/FQDN":  &t.Ip.Fqdn,
		"service/dns/CF":   &t.Cf,
		"service/dns/addr": &t.Addr,
	}
}
func (t *DNS) WebJson(md map[string]interface{}) {
	md["DNS Zone"] = t.Cf.Zone
}
