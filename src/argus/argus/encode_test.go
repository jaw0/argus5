// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Sep-01 11:09 (EDT)
// Function:

package argus

import (
	"testing"
)

func encodeExpect(t *testing.T, a string, b string) {

	e := Encode(a)
	// fmt.Printf("%s -> %s = %s\n", a, e, b)
	if e != b {
		t.Fail()
	}
}

func TestEncode(t *testing.T) {

	encodeExpect(t, "abc def", "abc%20def")
}
