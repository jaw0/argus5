// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Sep-03 17:21 (EDT)
// Function:

package configure

import (
	"fmt"
	"testing"

	"argus/argus"
)

type Conf struct {
	Ka string
	Kb int64 `cfconv:"timespec"`
	Kc bool
	Kd float64
	Ke argus.Status
	Kf [8]float64 `cfconv:"dotsev"`
	Kg *argus.Schedule
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
		},
	}

	cf := &Conf{}

	conf.InitFromConfig(cf, "", "")

	if cf.Ka != "foobar" {
		fmt.Printf("%#v\n", cf)
		t.Fail()
	}
	if cf.Kb != 43203 {
		fmt.Printf("%#v\n", cf)
		t.Fail()
	}
	if cf.Kc != true {
		fmt.Printf("%#v\n", cf)
		t.Fail()
	}
	if cf.Kd != 1.2345 {
		fmt.Printf("%#v\n", cf)
		t.Fail()
	}
	if cf.Ke != argus.MAJOR {
		fmt.Printf("%#v\n", cf)
		t.Fail()
	}
	if cf.Kf[3] != 123.45 {
		fmt.Printf("%#v\n", cf)
		t.Fail()
	}

	if cf.Kg == nil {
		fmt.Printf("%#v\n", cf)
		t.Fail()
	}

	fmt.Printf("%#v\n", cf)
}
