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

	"argus/clock"
	"argus/monel"
)

type OP struct {
	prec int
	f    func(*exprStack) (float64, bool)
}

var ops = map[string]OP{
	"time":  {6, fop_time},
	"SUM":   {5, fop_sum}, // group aggregate ops
	"AVE":   {5, fop_ave},
	"AVG":   {5, fop_ave},
	"MIN":   {5, fop_min},
	"MAX":   {5, fop_max},
	"COUNT": {5, fop_count},
	"rand":  {5, fop_rand}, // standard math functions
	"ceil":  {5, fop_ceil},
	"floor": {5, fop_floor},
	"abs":   {5, fop_abs},
	"sin":   {5, fop_sin},
	"tan":   {5, fop_sin},
	"cos":   {5, fop_cos},
	"log":   {5, fop_log},
	"exp":   {5, fop_exp},
	"sqrt":  {5, fop_sqrt},
	"^":     {4, fop_pow}, // basic arithmetic
	"*":     {3, fop_mul},
	"/":     {3, fop_div},
	"%":     {3, fop_mod},
	"+":     {2, fop_add},
	"-":     {2, fop_sub},
}

// calculate value of expression
func Calc(expr string, vars map[string]string) (float64, error) {

	pt, _, err := Parse(expr)
	if err != nil {
		return 0, err
	}

	res := RunExpr(pt, vars)
	v, err := strconv.ParseFloat(res, 64)
	return v, err
}

// run pre-compiled expr - return float
func RunExprF(pt []string, vars map[string]string) (float64, error) {

	res := RunExpr(pt, vars)
	v, err := strconv.ParseFloat(res, 64)
	return v, err
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
		if expr[0] == '"' {
			// "Top:Foo+Bar"
			e := strings.IndexByte(expr, '"')
			if e == -1 {
				return nil, errors.New("syntax error: unbalanced \"\"")
			}
			tok = append(tok, strings.TrimSpace(expr[1:e]))
			expr = strings.TrimSpace(expr[e+1:])
			continue
		}
		e := strings.IndexAny(expr, "()+-*/%^")
		if e == -1 {
			// slurp up remaining
			tok = append(tok, strings.TrimSpace(expr))
			break
		}
		if e == 0 {
			tok = append(tok, expr[0:1])
			expr = strings.TrimSpace(expr[e+1:])
			continue
		}

		tok = append(tok, strings.TrimSpace(expr[:e])) // word
		tok = append(tok, expr[e:e+1])                 // op
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
func RunExpr(pt []string, vars map[string]string) string {

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

		res, ok := opp.f(es)

		if !ok {
			return ""
		}

		es.pushf(res)
	}

	return es.pop()
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

func resultList(obj string) []float64 {

	m := monel.Find(obj)
	if m == nil {
		return nil
	}

	do := []*monel.M{m}
	res := []float64{}

	for len(do) != 0 {
		x := do[0]
		do = do[1:]

		children := x.Me.Children()

		if len(children) == 0 {
			v, err := strconv.ParseFloat(x.GetResult(), 64)
			if err != nil {
				res = append(res, v)
			}
		} else {
			do = append(do, children...)
		}
	}

	return res
}

func fop_sum(es *exprStack) (float64, bool) {

	obj := es.pop()
	rl := resultList(obj)

	if len(rl) == 0 {
		return 0, false
	}
	sum := 0.0
	for _, v := range rl {
		sum += v
	}
	return sum, true
}

func fop_count(es *exprStack) (float64, bool) {

	obj := es.pop()
	rl := resultList(obj)
	return float64(len(rl)), true
}

func fop_ave(es *exprStack) (float64, bool) {

	obj := es.pop()
	rl := resultList(obj)

	if len(rl) == 0 {
		return 0, false
	}
	sum := 0.0
	for _, v := range rl {
		sum += v
	}
	return sum / float64(len(rl)), true
}

func fop_min(es *exprStack) (float64, bool) {

	obj := es.pop()
	rl := resultList(obj)

	if len(rl) == 0 {
		return 0, false
	}

	min := rl[0]
	for _, v := range rl {
		if v < min {
			min = v
		}

	}
	return min, true
}

func fop_max(es *exprStack) (float64, bool) {

	obj := es.pop()
	rl := resultList(obj)

	if len(rl) == 0 {
		return 0, false
	}

	max := rl[0]
	for _, v := range rl {
		if v > max {
			max = v
		}

	}
	return max, true
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
