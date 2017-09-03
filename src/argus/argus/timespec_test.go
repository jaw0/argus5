// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Sep-01 11:09 (EDT)
// Function:

package argus

import (
	"fmt"
	"testing"
)

func timespecExpect(t *testing.T, a string, b int64) {

	e, _ := Timespec(a)
	if e != b {
		fmt.Printf("%s -> %d != %d\n", a, e, b)
		t.Fail()
	}
}

func TestTimeSpec(t *testing.T) {

	timespecExpect(t, "123", 123)
	timespecExpect(t, "123s", 123)
	timespecExpect(t, "123m", 7380)
	timespecExpect(t, "1m1", 61)
	timespecExpect(t, "1h1m1", 3661)
	timespecExpect(t, "10h10m5", 36605)
	timespecExpect(t, "1M", 30*24*3600)
}
