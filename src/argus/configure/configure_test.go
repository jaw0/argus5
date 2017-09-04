// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Sep-03 17:21 (EDT)
// Function:

package configure

import (
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
		},
	}

	cf := &Conf{}

	conf.InitFromConfig(cf, "", "")

	if cf.Ka != "foobar" {
		t.Fail()
	}
	if cf.Kb != 43203 {
		t.Fail()
	}
	if cf.Kc != true {
		t.Fail()
	}
	if cf.Kd != 1.2345 {
		t.Fail()
	}
	if cf.Ke != argus.MAJOR {
		t.Fail()
	}
	if cf.Kf[3] != 123.45 {
		t.Fail()
	}

}
