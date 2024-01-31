// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Sep-28 22:36 (EDT)
// Function:

package resolv

import (
	"bytes"
	"testing"
	"time"

	"github.com/jaw0/acdiag"
)

func xTestResolv(t *testing.T) {

	diag.SetConfig(diag.Config{Debug: map[string]bool{"resolv": true}})
	Init()
	defer Stop()

	ip := New("www.google.com.", []int{4})

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

func xTestReadResolv(t *testing.T) {
	txt := bytes.NewBufferString(`search example.com
nameserver 192.168.200.2
nameserver ::1
`)
	readFromResolvConf(txt)

}

func TestResolv(t *testing.T) {

	diag.SetConfig(diag.Config{Debug: map[string]bool{"resolv": true}})
	Init()
	defer Stop()

	for {
		New("nestor", []int{4})
		New("nestor.tcp4me.com", []int{4, 6})
		ip := New("nestor", []int{4, 6})
		time.Sleep(1 * time.Second)
		a, v, ok := ip.Addr()
		dl.Debug("-> %v -> %v; %v;%v", ip, a, v, ok)
		if a != "" {
			break
		}
	}

}
