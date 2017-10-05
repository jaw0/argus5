// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Sep-01 19:47 (EDT)
// Function:

package argus

import (
	"fmt"
	"testing"
)

func unpackExpect(t *testing.T, how string, data []byte, ex int32) {

	v, _ := Unpack(how, data)
	if ex != v {
		fmt.Printf("%s %X -> %d != %d\n", how, data, v, ex)
		t.Fail()
	}
}

func TestUnpack(t *testing.T) {

	unpackExpect(t, " c", []byte{254, 0, 0, 0}, -2)
	unpackExpect(t, " x   C ", []byte{254, 17, 1, 2}, 17)
	unpackExpect(t, " x1  C ", []byte{254, 17, 1, 2}, 17)
	unpackExpect(t, " x2  n ", []byte{254, 17, 37, 77, 2, 3}, 9549)
	unpackExpect(t, "x2  N ", []byte{254, 17, 37, 77, 2, 91, 1}, 625803867)

}
