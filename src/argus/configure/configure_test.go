// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Sep-03 17:21 (EDT)
// Function:

package configure

import (
	"fmt"
	"testing"

	"argus/argus"
	"github.com/jaw0/acgo/diag"
)

type Conf struct {
	Ka string
	Kb int64 `cfconv:"timespec"`
	Kc bool
	Kd float64
	Ke argus.Status
	Kf [8]float64 `cfconv:"dotsev"`
	Kg *argus.Schedule
	Kh [8]*argus.Schedule `cfconv:"dotsev"`
}

func TestConfigure(t *testing.T) {

	conf := &CF{
		Param: map[string]*CFV{
			"ka":       &CFV{Value: "foobar"},
			"kb":       &CFV{Value: "12h3"},
			"kc":       &CFV{Value: "yes"},
			"kd":       &CFV{Value: "1.2345"},
			"ke":       &CFV{Value: "major"},
			"kf.major": &CFV{Value: "123.45"},
			"kg":       &CFV{Value: &argus.Schedule{}},
			"kh":       &CFV{Value: &argus.Schedule{}},
			"kh.major": &CFV{Value: &argus.Schedule{}},
			"kh.minor": &CFV{Value: "yes"},
		},
	}

	diag.SetConfig(&diag.Config{}) // Debug: map[string]bool{"configure": true}})
	cf := &Conf{}

	conf.InitFromConfig(cf, "", "")

	if cf.Ka != "foobar" {
		fmt.Printf("A %#v\n", cf.Ka)
		t.Fail()
	}
	if cf.Kb != 43203 {
		fmt.Printf("B %#v\n", cf.Kb)
		t.Fail()
	}
	if cf.Kc != true {
		fmt.Printf("C %#v\n", cf.Kc)
		t.Fail()
	}
	if cf.Kd != 1.2345 {
		fmt.Printf("D %#v\n", cf.Kd)
		t.Fail()
	}
	if cf.Ke != argus.MAJOR {
		fmt.Printf("E %#v\n", cf.Ke)
		t.Fail()
	}
	if cf.Kf[4] != 123.45 {
		fmt.Printf("F %#v\n", cf.Kf)
		t.Fail()
	}

	if cf.Kg == nil {
		fmt.Printf("G %#v\n", cf.Kg)
		t.Fail()
	}

	fmt.Printf("%#v\n", cf.Kh)
}
