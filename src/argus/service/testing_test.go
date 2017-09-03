// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Sep-02 10:15 (EDT)
// Function:

package service

import (
	"fmt"
	"testing"
)

func pluckTest(t *testing.T, reg string, val string, exp string) {

	got := pluck(reg, val)
	if got != exp {
		fmt.Printf("re %s + %s -> '%s' != '%s'\n", reg, val, got, exp)
		t.Fail()
	}
}

func TestPluck(t *testing.T) {

	pluckTest(t, ": (.*)", "key: value", "value")
	pluckTest(t, "foo: (.*)", "key: value", "")
	pluckTest(t, `(\d+)`, "key: 1234", "1234")

}

// ################################################################

func jsonTest(t *testing.T, path string, val string, exp string) {

	got, _ := jsonPath(path, val)
	if got != exp {
		fmt.Printf("path %s + %s -> '%s' != '%s'\n", path, val, got, exp)
		t.Fail()
	}
}

func TestJson(t *testing.T) {

	jsonTest(t, "$.key", `{"key": "value"}`, "value")
	jsonTest(t, "$.key.foo[1]", `{"key": {"bar": 1, "foo": [1, 2, 3]}}`, "2")

}

// ################################################################

func exprTest(t *testing.T, expr string, val float64, exp float64) {

	got, _ := doExpr(expr, val)
	if got != exp {
		fmt.Printf("expr %s + %f -> %f != %f\n", expr, val, got, exp)
		t.Fail()
	}
}

func TestExpr(t *testing.T) {

	exprTest(t, "x * 2 + 1", 1, 3)
	exprTest(t, "7 * x + 2", 2, 16)
}
