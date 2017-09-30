// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Sep-26 22:18 (EDT)
// Function:

package expr

import (
	"fmt"
	"testing"
)

func TestCompute(t *testing.T) {

	//fmt.Printf("%v\n", Tokenize(" SUM(Top:X:Y) + Top:Bar:Baz * Top:Foo"))
	//fmt.Printf("%v\n", Tokenize(" SUM(Top:X:Y) + Top:Bar:Baz * AVG(Top:Foo)"))
	//fmt.Printf("%v\n", Tokenize(" SUM(Top:X:Y) + Top:Bar:Baz * Top:Foo -"))
	//fmt.Printf("%v\n", Tokenize(" SUM(Top:X:Y) + Top:Bar:Baz * AVG(Top:Foo) -"))
	//fmt.Printf("%v\n", Tokenize(" SUM(Top:X:Y) + {Top:Bar:B+z} * Top:Foo"))
	//
	//t, _ := Tokenize("Top:Foo + SUM(Top:Bar:S) / COUNT(TOP:Bar:C) * 4")
	//p, _, err := Parse(t)
	//fmt.Printf("p %v; err %v\n", p, err)

	res, _ := Calc("4 * x + sqrt(6 * y / 2)", map[string]string{"x": "5", "y": "3"})
	fmt.Printf("res: %v\n", res)

	if res != 23 {
		t.Fail()
	}

}
