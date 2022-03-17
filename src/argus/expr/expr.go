// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Sep-27 21:37 (EDT)
// Function: expression calculations - for service testing + compute

package expr

import (
	"errors"
	"fmt"
	"math"
	"math/rand"
	"strconv"
	"strings"

	"argus.domain/argus/argus"
	"argus.domain/argus/clock"
	"argus.domain/argus/monel"
)

type OP struct {
	prec int
	rry  bool // true => all services must be ready
	fop  func(*exprStack) (float64, bool)
	fagg func(*exprStack, bool) (float64, string, bool)
}

var ops = map[string]OP{
	"time":      {50, true, fop_time, nil}, // unix time, seconds
	"rand":      {50, true, fop_rand, nil}, // [0..1)
	"SUM":       {20, true, nil, fop_sum},  // group aggregate ops
	"AVE":       {20, true, nil, fop_ave},  //   SUM(Top:Foo:Bar)
	"AVG":       {20, true, nil, fop_ave},  // test will wait until all services are ready
	"MIN":       {20, true, nil, fop_min},
	"MAX":       {20, true, nil, fop_max},
	"COUNT":     {20, true, nil, fop_count},
	"NSUM":      {20, false, nil, fop_sum}, // group aggregate ops
	"NAVE":      {20, false, nil, fop_ave}, // tests will run ignoring any services not ready
	"NAVG":      {20, false, nil, fop_ave},
	"NMIN":      {20, false, nil, fop_min},
	"NMAX":      {20, false, nil, fop_max},
	"NCOUNT":    {20, false, nil, fop_count},
	"NUP":       {20, false, nil, fop_up},   // count of services that are up
	"NDOWN":     {20, false, nil, fop_down}, // ... or down
	"NOVERRIDE": {20, false, nil, fop_over}, // ... or overridden
	"ceil":      {20, true, fop_ceil, nil},  // standard math functions
	"floor":     {20, true, fop_floor, nil},
	"abs":       {20, true, fop_abs, nil},
	"sin":       {20, true, fop_sin, nil},
	"tan":       {20, true, fop_sin, nil},
	"cos":       {20, true, fop_cos, nil},
	"log":       {20, true, fop_log, nil}, // natural log
	"exp":       {20, true, fop_exp, nil}, // e ^ x
	"sqrt":      {20, true, fop_sqrt, nil},
	"^":         {16, true, fop_pow, nil}, // basic arithmetic
	"*":         {15, true, fop_mul, nil},
	"/":         {14, true, fop_div, nil},
	"%":         {13, true, fop_mod, nil},
	"+":         {12, true, fop_add, nil},
	"-":         {12, true, fop_sub, nil},
}

// calculate value of expression
func Calc(expr string, vars map[string]string) (float64, string, error) {

	pt, _, err := Parse(expr)
	if err != nil {
		return 0, "", err
	}

	res, nrdy := RunExpr(pt, vars)
	if nrdy != "" {
		return 0, nrdy, nil
	}
	v, err := strconv.ParseFloat(res, 64)
	return v, "", err
}

// run pre-compiled expr - return float
func RunExprF(pt []string, vars map[string]string) (float64, string, error) {

	res, nrdy := RunExpr(pt, vars)
	if nrdy != "" {
		return 0, nrdy, nil
	}
	v, err := strconv.ParseFloat(res, 64)
	return v, "", err
}

func Parse(expr string) ([]string, map[string]bool, error) {

	t, err := tokenize(expr)
	if err != nil {
		return nil, nil, err
	}
	return parse(t)
}

func tokenize(expr string) ([]string, error) {

	var tok []string

	expr = strings.TrimSpace(expr)

	for expr != "" {
		if expr[0] == '{' {
			// {Top:Foo+Bar}
			e := strings.IndexByte(expr, '}')
			if e == -1 {
				// unbalanced {}
				return nil, errors.New("syntax error: unbalanced {}")
			}
			tok = append(tok, strings.TrimSpace(expr[1:e]))
			expr = strings.TrimSpace(expr[e+1:])
			continue
		}
		if expr[0] == '"' || expr[0] == '\'' {
			// "Top:Foo+Bar"
			e := strings.IndexByte(expr[1:], expr[0])
			if e == -1 {
				return nil, errors.New("syntax error: unbalanced quotes")
			}
			tok = append(tok, strings.TrimSpace(expr[1:e+1]))
			expr = strings.TrimSpace(expr[e+2:])
			continue
		}
		e := strings.IndexAny(expr, "()+-*/%^ \t")
		if e == -1 {
			// slurp up remaining
			tok = append(tok, strings.TrimSpace(expr))
			break
		}
		if e == 0 {
			// single char op
			tok = append(tok, expr[0:1])
			expr = strings.TrimSpace(expr[e+1:])
			continue
		}

		tok = append(tok, strings.TrimSpace(expr[:e])) // word
		if expr[e] != ' ' && expr[e] != '\t' {
			tok = append(tok, expr[e:e+1]) // op
		}
		expr = strings.TrimSpace(expr[e+1:])
	}

	return tok, nil
}

// Dijkstra
func parse(tok []string) ([]string, map[string]bool, error) {

	var oq []string
	var op []string
	objs := make(map[string]bool)

	//while there are tokens to be read:
	//	read a token.
	for len(tok) != 0 {
		t := tok[0]
		tok = tok[1:]

		if t == "(" {
			// if the token is a left bracket (i.e. "("), then:
			//   push it onto the operator stack.
			op = append(op, t)
			continue
		}
		if t == ")" {
			//if the token is a right bracket (i.e. ")"), then:
			//	while the operator at the top of the operator stack is not a left bracket:
			//		pop operators from the operator stack onto the output queue.
			//	pop the left bracket from the stack.
			//	/* if the stack runs out without finding a left bracket, then there are
			//	mismatched parentheses. */

			matched := false
			for len(op) != 0 {
				nt := op[len(op)-1]
				op = op[:len(op)-1]

				if nt == "(" {
					matched = true
					break
				}
				oq = append(oq, nt)
			}
			if !matched {
				return nil, nil, errors.New("syntax error: mismatched ()")
			}
			continue
		}

		opp, ok := ops[t]

		// if the token is a number, then push it to the output queue.
		if !ok {
			oq = append(oq, t)
			if strings.HasPrefix(t, "Top") {
				objs[t] = true
			}
			continue
		}

		//if the token is an operator, then:
		//	while there is an operator at the top of the operator stack with
		//		greater than or equal to precedence and the operator is left associative:
		//			pop operators from the operator stack, onto the output queue.
		//	push the read operator onto the operator stack.

		for len(op) != 0 {
			o := op[len(op)-1]
			top := ops[o]
			if top.prec < opp.prec {
				break
			}
			oq = append(oq, o)
			op = op[:len(op)-1]
		}
		op = append(op, t)
	}

	//if there are no more tokens to read:
	//	while there are still operator tokens on the stack:
	//		/* if the operator token on the top of the stack is a bracket, then
	//		there are mismatched parentheses. */
	//		pop the operator onto the output queue.

	for len(op) != 0 {
		o := op[len(op)-1]
		op = op[:len(op)-1]
		oq = append(oq, o)
	}

	return oq, objs, nil
}

// ################################################################

type exprStack struct {
	prog []string
	s    []string
	vars map[string]string
}

// run compiled expr
func RunExpr(pt []string, vars map[string]string) (string, string) {

	es := &exprStack{prog: pt, vars: vars}

	for len(es.prog) != 0 {
		// pop next op off program stack
		op := es.prog[0]
		es.prog = es.prog[1:]
		opp, ok := ops[op]

		if !ok {
			// value? copy to value stack
			es.push(op)
			continue
		}

		var res float64
		var nrdy string

		if opp.fagg != nil {
			res, nrdy, ok = opp.fagg(es, opp.rry)
		} else {
			res, ok = opp.fop(es)
		}

		if nrdy != "" {
			return "", nrdy
		}

		if !ok {
			return "", ""
		}

		es.pushf(res)
	}

	return es.pop(), ""
}

func (es *exprStack) push(x string) {
	es.s = append(es.s, x)
}
func (es *exprStack) pushf(x float64) {
	es.push(fmt.Sprintf("%f", x))
}

func (es *exprStack) pop() string {
	if len(es.s) == 0 {
		return ""
	}
	x := es.s[len(es.s)-1]
	es.s = es.s[:len(es.s)-1]
	return x
}

func (es *exprStack) popf() (float64, bool) {

	x := es.pop()

	if strings.HasPrefix(x, "Top") {
		// find value
		m := monel.Find(x)

		if m == nil {
			return 0, false
		}

		x = m.GetResult()
	}
	// look up var
	if v, ok := es.vars[x]; ok {
		x = v
	}

	// convert constant
	v, err := strconv.ParseFloat(x, 64)
	if err != nil {
		return 0, false
	}

	return v, true
}

// ****************************************************************

func resultList(obj string) ([]float64, string) {

	m := monel.Find(obj)
	if m == nil {
		return nil, ""
	}

	do := []*monel.M{m}
	res := []float64{}
	nrdy := ""

	for len(do) != 0 {
		x := do[0]
		do = do[1:]

		children := x.Me.Children()

		if len(children) == 0 {
			v, err := strconv.ParseFloat(x.GetResult(), 64)
			if err == nil {
				res = append(res, v)
			}
		} else {
			do = append(do, children...)
		}
	}

	return res, nrdy
}

func statusList(obj string) []argus.Status {

	m := monel.Find(obj)
	if m == nil {
		return nil
	}

	do := []*monel.M{m}
	res := []argus.Status{}

	for len(do) != 0 {
		x := do[0]
		do = do[1:]

		children := x.Me.Children()

		if len(children) == 0 {
			_, v := x.Status() // => status, ovstatus
			res = append(res, v)
		} else {
			do = append(do, children...)
		}
	}

	return res
}

func fop_sum(es *exprStack, rry bool) (float64, string, bool) {

	obj := es.pop()
	rl, nrdy := resultList(obj)

	if rry && nrdy != "" {
		return 0, nrdy, true
	}

	if len(rl) == 0 {
		return 0, "", false
	}
	sum := 0.0
	for _, v := range rl {
		sum += v
	}
	return sum, "", true
}

func fop_count(es *exprStack, rry bool) (float64, string, bool) {

	obj := es.pop()
	rl, nrdy := resultList(obj)

	if rry && nrdy != "" {
		return 0, nrdy, true
	}

	return float64(len(rl)), "", true
}

func fop_ave(es *exprStack, rry bool) (float64, string, bool) {

	obj := es.pop()
	rl, nrdy := resultList(obj)

	if rry && nrdy != "" {
		return 0, nrdy, true
	}

	if len(rl) == 0 {
		return 0, "", false
	}
	sum := 0.0
	for _, v := range rl {
		sum += v
	}
	return sum / float64(len(rl)), "", true
}

func fop_min(es *exprStack, rry bool) (float64, string, bool) {

	obj := es.pop()
	rl, nrdy := resultList(obj)

	if rry && nrdy != "" {
		return 0, nrdy, true
	}

	if len(rl) == 0 {
		return 0, "", false
	}

	min := rl[0]
	for _, v := range rl {
		if v < min {
			min = v
		}

	}
	return min, "", true
}

func fop_max(es *exprStack, rry bool) (float64, string, bool) {

	obj := es.pop()
	rl, nrdy := resultList(obj)

	if rry && nrdy != "" {
		return 0, nrdy, true
	}

	if len(rl) == 0 {
		return 0, "", false
	}

	max := rl[0]
	for _, v := range rl {
		if v > max {
			max = v
		}

	}
	return max, "", true
}

func fop_up(es *exprStack, rry bool) (float64, string, bool) {

	obj := es.pop()
	sl := statusList(obj)
	var n float64

	for _, v := range sl {
		switch v {
		case argus.CLEAR, argus.OVERRIDE:
			n++
		}
	}

	return n, "", true
}

func fop_down(es *exprStack, rry bool) (float64, string, bool) {

	obj := es.pop()
	sl := statusList(obj)
	var n float64

	for _, v := range sl {
		switch v {
		case argus.CLEAR, argus.OVERRIDE, argus.UNKNOWN:
			break
		default:
			n++
		}
	}

	return n, "", true
}

func fop_over(es *exprStack, rry bool) (float64, string, bool) {

	obj := es.pop()
	sl := statusList(obj)
	var n float64

	for _, v := range sl {
		switch v {
		case argus.OVERRIDE:
			n++
		}
	}

	return n, "", true
}

// ****************************************************************

func fop_time(es *exprStack) (float64, bool) {

	now := clock.Unix()
	return float64(now), true
}

func fop_rand(es *exprStack) (float64, bool) {

	return rand.Float64(), true
}

func fop_sin(es *exprStack) (float64, bool) {
	a, ok := es.popf()
	if !ok {
		return 0, false
	}
	return math.Sin(a), true

}

func fop_cos(es *exprStack) (float64, bool) {

	a, ok := es.popf()
	if !ok {
		return 0, false
	}
	return math.Cos(a), true

}
func fop_tan(es *exprStack) (float64, bool) {

	a, ok := es.popf()
	if !ok {
		return 0, false
	}
	return math.Tan(a), true

}

func fop_ceil(es *exprStack) (float64, bool) {

	a, ok := es.popf()
	if !ok {
		return 0, false
	}
	return math.Ceil(a), true

}
func fop_floor(es *exprStack) (float64, bool) {

	a, ok := es.popf()
	if !ok {
		return 0, false
	}
	return math.Floor(a), true

}
func fop_abs(es *exprStack) (float64, bool) {

	a, ok := es.popf()
	if !ok {
		return 0, false
	}
	return math.Abs(a), true

}
func fop_exp(es *exprStack) (float64, bool) {

	a, ok := es.popf()
	if !ok {
		return 0, false
	}
	return math.Exp(a), true

}
func fop_log(es *exprStack) (float64, bool) {

	a, ok := es.popf()
	if !ok || a == 0 {
		return 0, false
	}
	return math.Log(a), true

}
func fop_sqrt(es *exprStack) (float64, bool) {

	a, ok := es.popf()
	if !ok {
		return 0, false
	}
	return math.Sqrt(a), true

}

// ****************************************************************

func fop_add(es *exprStack) (float64, bool) {

	a, ok := es.popf()
	if !ok {
		return 0, false
	}
	b, ok := es.popf()

	return b + a, ok
}

func fop_sub(es *exprStack) (float64, bool) {

	a, ok := es.popf()
	if !ok {
		return 0, false
	}
	b, ok := es.popf()

	return b - a, ok
}

func fop_pow(es *exprStack) (float64, bool) {

	a, ok := es.popf()
	if !ok {
		return 0, false
	}
	b, ok := es.popf()

	return math.Pow(b, a), ok
}

func fop_mul(es *exprStack) (float64, bool) {

	a, ok := es.popf()
	if !ok {
		return 0, false
	}
	b, ok := es.popf()

	return b * a, ok
}

func fop_div(es *exprStack) (float64, bool) {

	a, ok := es.popf()
	if !ok {
		return 0, false
	}
	b, ok := es.popf()

	if a == 0 {
		return 0, ok
	}
	return b / a, ok
}

func fop_mod(es *exprStack) (float64, bool) {

	a, ok := es.popf()
	if !ok {
		return 0, false
	}
	b, ok := es.popf()

	ia := int64(a)
	if ia == 0 {
		return 0, ok
	}
	return float64(int64(b) % ia), ok
}
