// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Sep-26 21:35 (EDT)
// Function: snmp

package snmp

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/soniah/gosnmp"

	"argus/argus"
	"argus/configure"
	"argus/diag"
	"argus/resolv"
	"argus/service"
)

type Conf struct {
	Port         int
	Community    string
	Oid          string // 1.3.6.1.2, ifInOctets.1, ifInOctets[Interface0/0]
	SNMPVersion  string
	SNMPUser     string
	SNMPPass     string
	SNMPPrivPass string
	SNMPAuth     string // none, md5, sha1
	SNMPPriv     string // none, des, aes, [RSN: 3des, aes128, aes192, aes256]
}

type SNMP struct {
	S       *service.Service
	Cf      Conf
	Ip      *resolv.IP
	oid     string   // actual oid to monitor
	baseOid string   // oid of table (ifInOctets)
	idxDesc string   // description "Serial1/0"
	idxOids []string // list of tables to search
	descOid string   // discovered oid to verify desc - Get(descOid) == idxDesc
	uptime  int64    // to detect reboot
	v3sec   *gosnmp.UsmSecurityParameters
}

type snmpResult struct {
	resType gosnmp.Asn1BER
	value   string
}

const (
	sysUptime0 = ".1.3.6.1.2.1.1.3.0"
)

var dl = diag.Logger("snmp")

func init() {
	// register with service factory
	service.Register("SNMP", New)
	service.Register("UDP/SNMP", New)
}

func New(conf *configure.CF, s *service.Service) service.Monitor {
	t := &SNMP{S: s}
	t.Cf.Port = 161
	t.Cf.SNMPVersion = "2c"
	return t
}

func (t *SNMP) PreConfig(conf *configure.CF, s *service.Service) error {

	// parse spec
	//   Service [UDP/]SNMP[2c,3,v2c,v3][/oid]
	name := conf.Name

	if strings.Contains(name, "SNMPv2c") || strings.Contains(name, "SNMP2c") {
		t.Cf.SNMPVersion = "2c"
	}
	if strings.Contains(name, "SNMPv3") || strings.Contains(name, "SNMP3") {
		t.Cf.SNMPVersion = "3"
	}

	// pull oid from UDP/SNMP/oid
	name = strings.TrimPrefix(name, "UDP/")
	slash := strings.IndexByte(name, '/')
	if slash != -1 {
		t.Cf.Oid = name[slash+1:]
	}

	conf.InitFromConfig(&t.Cf, "snmp", "")
	ip, err := resolv.Config(conf)
	if err != nil {
		return err
	}

	t.Ip = ip

	// parse oid spec, configure defaults, etal
	ocf, baseoid, descr := parseOid(t.Cf.Oid)

	if baseoid == "" {
		return errors.New("unrecognized or invalid OID")
	}

	if descr != "" {
		dl.Debug("autodiscover %s in %s", descr, baseoid)
		t.baseOid = baseoid
		t.idxDesc = descr
	} else {
		t.oid = baseoid
	}

	if ocf != nil {
		dl.Debug("setting defaults %s", ocf.Calc)
		if ocf.UpValue != 0 {
			s.Cf.Eqvalue[int(argus.UNKNOWN)] = float64(ocf.UpValue)
		}
		s.Cf.Calc = ocf.Calc

		if descr != "" {
			t.idxOids = ocf.idxOids()
		}
	}

	if descr != "" && len(t.idxOids) == 0 {
		return errors.New("I don't know how to autodiscover oids in this table")
	}

	return nil
}

func (t *SNMP) Config(conf *configure.CF, s *service.Service) error {

	// validate
	if t.Cf.Port == 0 {
		return errors.New("port not specified")
	}
	if t.Cf.Oid == "" {
		return errors.New("oid not specified")
	}

	t.Cf.SNMPAuth = strings.ToLower(t.Cf.SNMPAuth)
	t.Cf.SNMPPriv = strings.ToLower(t.Cf.SNMPPriv)
	t.v3sec = t.snmpV3Security()

	label := t.Cf.Oid
	uname := "SNMP_"
	host := t.Ip.Hostname()

	if t.idxDesc == "" {
		uname += t.oid + "_" + host
	} else {
		uname += t.baseOid + "_" + t.idxDesc + "_" + host
	}

	friendly := t.Cf.Oid + " on " + host

	s.SetNames(uname, label, friendly)

	return nil
}

func (t *SNMP) Init() error {
	return nil
}
func (t *SNMP) Hostname() string {
	return t.Ip.Hostname()
}
func (t *SNMP) Recycle() {
}
func (t *SNMP) Abort() {
}
func (t *SNMP) DoneConfig() {
}
func (t *SNMP) DumpInfo() map[string]interface{} {
	return map[string]interface{}{
		"service/ip/CF":    &t.Ip.Cf,
		"service/snmp/CF/": &t.Cf,
		"service/snmp/": &struct {
			oid     string
			baseOid string
			idxDesc string
			idxOids []string
			descOid string
			uptime  int64
		}{t.oid, t.baseOid, t.idxDesc, t.idxOids, t.descOid, t.uptime},
	}
}
func (t *SNMP) WebJson(md map[string]interface{}) {
	md["SNMP oid"] = t.Cf.Oid
}

// ################################################################

func (t *SNMP) Start(s *service.Service) {

	s.Debug("snmp start")
	defer s.Done()

	client := t.snmpClient()
	if client == nil {
		return
	}

	err := client.Connect()
	if err != nil {
		s.Debug("connect failed: %v", err)
		s.Fail("connect failed")
		return
	}

	if t.oid == "" {
		if !t.autoDiscover(client) {
			return
		}
	}

	oids := []string{t.oid, sysUptime0}
	if t.descOid != "" {
		oids = append(oids, t.descOid)
	}

	resp, err := client.Get(oids)
	dl.Debug("res: %#v; err %v", resp, err)

	if err != nil {
		s.Debug("get failed: %v", err)
		s.Fail("snmp get failed")
		return
	}

	res := t.getResults(resp)
	if !t.verifyResults(res) {
		return
	}

	vres := res[t.oid]
	if vres == nil {
		s.Fail("invalid response - no result")
		return
	}

	s.CheckValue(vres.value, "snmp")
}

//################################################################

func (t *SNMP) autoDiscover(client *gosnmp.GoSNMP) bool {

	for _, oid := range t.idxOids {
		dl.Debug("audodiscovery %s", oid)
		resp, _ := getall(oid, client)
		if t.searchDescr(resp) {
			return true
		}
	}

	return false
}

func getall(oid string, client *gosnmp.GoSNMP) ([]gosnmp.SnmpPDU, error) {

	// first try get bulk
	resp, err := client.BulkWalkAll(oid)
	if err != nil && len(resp) != 0 {
		dl.Debug("bulkwalk: %v", err)
		return resp, nil
	}
	// then try get next
	resp, err = client.WalkAll(oid)
	dl.Debug("walkall: %v", err)
	return resp, err
}

func (t *SNMP) searchDescr(resp []gosnmp.SnmpPDU) bool {

	for _, pdu := range resp {
		val := pduToString(&pdu)
		dl.Debug("resp: %s = %s", pdu.Name, val)

		if val == t.idxDesc {
			t.descOid = pdu.Name
			dot := strings.LastIndexByte(pdu.Name, '.')
			if dot == -1 {
				dl.Bug("returned oid has no dots?! %s", pdu.Name)
				return false
			}
			idx := pdu.Name[dot:] // .12
			t.oid = t.baseOid + idx
			t.S.Debug("discovered %s = %s => oid %s", pdu.Name, val, t.oid)
			return true
		}
	}
	return false
}

func (t *SNMP) verifyResults(rm map[string]*snmpResult) bool {

	// reboot?
	upres := rm[sysUptime0]
	if upres == nil {
		t.S.Fail("invalid response - no sysUptime")
		return false
	}
	uptime := upres.Int()

	logged := false
	if uptime < t.uptime {
		t.S.Loggit("INFO", "device rebooted")
		t.S.ResetRateCalc()
		logged = true
	}
	t.uptime = uptime

	// reconfigure?
	if t.descOid != "" {
		dres := rm[t.descOid]
		if dres == nil {
			t.S.Fail("invalid response - no check oid")
			return false
		}

		if dres.value != t.idxDesc {
			if !logged {
				t.S.Loggit("INFO", "device reconfigured")
				t.S.ResetRateCalc()
				logged = true
			}

			t.oid = ""
			// we will re-autodiscover next time
			return false
		}
	}

	return true
}

func (t *SNMP) snmpClient() *gosnmp.GoSNMP {

	addr, _, fail := t.Ip.Addr()
	if fail {
		t.S.FailNow("cannot resolve hostname")
		return nil
	}
	if addr == "" {
		t.S.Debug("hostname still resolving")
		return nil
	}

	client := &gosnmp.GoSNMP{
		Target:    addr,
		Port:      uint16(t.Cf.Port),
		Community: t.Cf.Community,
		Version:   t.snmpVersion(),
		Timeout:   time.Duration(t.S.Cf.Timeout) * time.Second,
		Retries:   0,
		MaxOids:   gosnmp.MaxOids,
		Logger:    newSnmpLogger(t.S),
	}

	if t.Cf.SNMPVersion == "3" {
		if t.Cf.SNMPPass != "" && t.Cf.SNMPPrivPass != "" {
			client.MsgFlags = gosnmp.AuthPriv
			client.SecurityModel = gosnmp.UserSecurityModel
		} else if t.Cf.SNMPPass != "" {
			client.MsgFlags = gosnmp.AuthNoPriv
			client.SecurityModel = gosnmp.UserSecurityModel
		} else {
			client.MsgFlags = gosnmp.NoAuthNoPriv
			client.SecurityModel = gosnmp.UserSecurityModel
		}
		// NB - NoAuthPriv is not a thing

		if client.SecurityModel == gosnmp.UserSecurityModel {
			client.SecurityParameters = t.v3sec
		}
	}

	t.S.Debug("connecting to udp/%s/%d", addr, t.Cf.Port)

	return client
}

func (t *SNMP) snmpV3Security() *gosnmp.UsmSecurityParameters {

	auth := gosnmp.NoAuth
	switch t.Cf.SNMPAuth {
	case "md5":
		auth = gosnmp.MD5
	case "sha1":
		auth = gosnmp.SHA
	}

	priv := gosnmp.NoPriv
	switch t.Cf.SNMPPriv {
	case "des":
		priv = gosnmp.DES
	case "aes", "aes128":
		priv = gosnmp.AES
	}
	//case "3des":
	//	priv = gosnmp.3DES
	//case "aes192":
	//	priv = gosnmp.AES192
	//case "aes256":
	//	priv = gosnmp.AES256

	return &gosnmp.UsmSecurityParameters{
		UserName:                 t.Cf.SNMPUser,
		AuthenticationProtocol:   auth,
		AuthenticationPassphrase: t.Cf.SNMPPass,
		PrivacyProtocol:          priv,
		PrivacyPassphrase:        t.Cf.SNMPPrivPass,
	}
}

func (t *SNMP) getResults(res *gosnmp.SnmpPacket) map[string]*snmpResult {

	rm := make(map[string]*snmpResult)

	for _, pdu := range res.Variables {
		name := pdu.Name
		value := pduToString(&pdu)
		t.S.Debug("got result: %s -> %s", name, value)

		rm[name] = &snmpResult{pdu.Type, value}
	}

	return rm
}

func pduToString(pdu *gosnmp.SnmpPDU) string {

	switch pdu.Type {
	case gosnmp.OctetString:
		return string(pdu.Value.([]byte))
	default:
		return fmt.Sprintf("%v", pdu.Value)
	}
}

func (t *SNMP) snmpVersion() gosnmp.SnmpVersion {

	switch t.Cf.SNMPVersion {
	case "1":
		return gosnmp.Version1
	case "3":
		return gosnmp.Version3
	default:
		return gosnmp.Version2c
	}
}

func (res *snmpResult) Int() int64 {
	v, _ := strconv.ParseInt(res.value, 10, 64)
	return v
}

//################################################################

type snmpLogger struct {
	s *service.Service
}

func newSnmpLogger(s *service.Service) *snmpLogger {
	return &snmpLogger{s}
}

func (l *snmpLogger) Print(args ...interface{}) {
	l.s.Debug("%v", args)
}
func (l *snmpLogger) Printf(f string, args ...interface{}) {
	l.s.Debug(f, args...)
}
