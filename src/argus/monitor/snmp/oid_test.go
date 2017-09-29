// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Sep-28 15:12 (EDT)
// Function:

package snmp

import (
	"fmt"
	"testing"
)

func testParseOid(t *testing.T, spec string, expoid string, expdesc string) {

	cf, oid, desc := parseOid(spec)

	if oid != expoid || desc != expdesc {
		fmt.Printf("oid %s, desc %s; cf %v\n", oid, desc, cf)
		t.Fail()
	}
}

func TestParseOid(t *testing.T) {

	testParseOid(t, "ifInOctets.123[Serial1/0]", ".1.3.6.1.2.1.2.2.1.10.123", "Serial1/0")

}
