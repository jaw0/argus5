// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Sep-26 21:35 (EDT)
// Function: snmp oids

package snmp

import (
	"strings"

	"argus/configure"
)

type OidConf struct {
	Oid     string
	Idx     string
	UpValue int
	Calc    string
	Watch   string
}

const IFIDX = "ifDescr ifName"

var oids = map[string]*OidConf{
	"sysUptime":          {Oid: ".1.3.6.1.2.1.1.3"},
	"ifNumber":           {Oid: ".1.3.6.1.2.1.2.1"},
	"ifDescr":            {Oid: ".1.3.6.1.2.1.2.2.1.2"},
	"ifName":             {Oid: ".1.3.6.1.2.1.31.1.1.1.1"},
	"ifAlias":            {Oid: ".1.3.6.1.2.1.31.1.1.1.18"},
	"dskpath":            {Oid: ".1.3.6.1.4.1.2021.9.1.2"},
	"dskDevs":            {Oid: ".1.3.6.1.4.1.2021.9.1.3"},
	"ifAdminStatus":      {Oid: ".1.3.6.1.2.1.2.2.1.7", Idx: IFIDX, UpValue: 1},
	"ifOperStatus":       {Oid: ".1.3.6.1.2.1.2.2.1.8", Idx: IFIDX, UpValue: 1},
	"ifInErrors":         {Oid: ".1.3.6.1.2.1.2.2.1.14", Idx: IFIDX},
	"ifOutErrors":        {Oid: ".1.3.6.1.2.1.2.2.1.20", Idx: IFIDX},
	"ifInOctets":         {Oid: ".1.3.6.1.2.1.2.2.1.10", Idx: IFIDX, Calc: "ave rate bits"},
	"ifOutOctets":        {Oid: ".1.3.6.1.2.1.2.2.1.16", Idx: IFIDX, Calc: "ave rate bits"},
	"BGPPeerState":       {Oid: ".1.3.6.1.2.1.15.3.1.2", UpValue: 6}, // NB - .peerip
	"dskPercent":         {Oid: ".1.3.6.1.4.1.2021.9.1.9", Idx: "dskPath dskDevs"},
	"isdnLapdOperStatus": {Oid: ".1.3.6.1.2.1.10.20.1.3.4.1.2", Idx: IFIDX, UpValue: 3},
}

// # yeah, it is much shorter to type 'ciscoEnvMonTemperatureStatusValue' ...
// ciscoEnvMonTemperatureStatusValue => { oid => '.1.3.6.1.4.1.9.9.13.1.3.1.3' },

func NewOID(conf *configure.CF) error {

	c := &OidConf{}
	conf.InitFromConfig(c, "snmpoid", "")

	oids[conf.Name] = c
	conf.CheckTypos()
	return nil
}

func LoadMibFile(file string) {

}

// words.numbers[descr]
// => Conf, (full oid, "") or (base oid, descr)
func parseOid(spec string) (*OidConf, string, string) {

	if spec == "" {
		return nil, "", ""
	}

	// parse out [words][numbers][descr]
	word := ""
	numb := ""
	desc := ""

	bk := strings.IndexByte(spec, '[')
	if bk != -1 {
		desc = spec[bk+1 : len(spec)-1]
		spec = spec[:bk]
	}

	if spec[0] >= 'a' && spec[0] <= 'z' || spec[0] >= 'A' && spec[0] <= 'Z' {
		dot := strings.IndexByte(spec, '.')
		word = spec[:dot] // ifThing
		numb = spec[dot:] // .4
	} else {
		numb = spec
		return nil, numb, desc
	}

	ocf := oids[word]
	if ocf == nil {
		// not found
		return nil, "", ""
	}

	oid := ocf.Oid + numb

	return ocf, oid, desc
}

func (cf *OidConf) idxOids() []string {

	o := strings.Fields(cf.Idx)
	var r []string

	for _, id := range o {
		_, oid, _ := parseOid(id)
		if oid != "" {
			r = append(r, oid)
		}
	}

	return r
}
