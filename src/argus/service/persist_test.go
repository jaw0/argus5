// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Sep-02 19:51 (EDT)
// Function:

package service

import (
	"encoding/json"
	"fmt"
	"testing"

	"argus/argus"
)

func TestPersist(t *testing.T) {

	s := &Service{
		p: Persist{
			Statuses: map[string]argus.Status{
				"qtssjc": argus.MINOR,
			},
			Hwab: &HWAB{A: 12.3, B: 1.23, C: []float32{1, 2, 3, 4, 5}},
			Calc: calc{Ave: 1.234},
		},
	}
	s.Cf = defaults

	pm := make(map[string]interface{})
	s.Persist(pm)

	dat, _ := json.Marshal(pm)
	fmt.Printf("json: %s\n\n", dat)
	pn := make(map[string]interface{})
	err := json.Unmarshal(dat, &pn)
	fmt.Printf("mi: %#v; %v\n\n", pn, err)

	n := &Service{
		p: Persist{
			Statuses: map[string]argus.Status{},
			Hwab:     &HWAB{},
		},
	}
	n.Cf = defaults

	n.Restore(pn)
	//fmt.Printf("status %s, s: %#v; hwab: %#v\n", n.p.Status, n, n.p.Hwab)

	//	if n.p.Status != s.p.Status {
	//		t.Fail()
	//	}
}
