// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Sep-28 22:36 (EDT)
// Function:

package resolv

import (
	"testing"
	"time"

	"argus/diag"
)

func TestResolv(t *testing.T) {

	diag.SetConfig(&diag.Config{Debug: map[string]bool{"resolv": true}})
	Init()
	defer Stop()

	ip := New("www.google.com.")

	for {
		addr, _, fail := ip.Addr()
		if addr != "" {
			return
		}
		if fail {
			break
		}
		time.Sleep(time.Second)
	}

	t.Fail()
}
