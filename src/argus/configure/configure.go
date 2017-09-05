// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Sep-02 23:23 (EDT)
// Function: configure thinsg

package configure

import (
	"reflect"
	"strconv"
	"strings"

	"argus/argus"
	"argus/diag"
)

type CFV struct {
	Value   interface{} // string | *Schedule
	Line    int
	Used    bool
	Inherit bool
}
type cfcv struct {
	value interface{}
	ok    bool
}

type CF struct {
	parent *CF
	Type   string // service, group, alias
	Name   string // group name, monitor name
	File   string
	Param  map[string]*CFV
	cache  map[string]cfcv
}

var dl = diag.Logger("configure")

func (cf *CF) SetParent(pcf *CF) {
	cf.parent = pcf
}

func (cf *CF) InitFromConfig(dst interface{}, section string, prefix string) {

	var val = reflect.ValueOf(dst).Elem()

	for i := 0; i < val.NumField(); i++ {

		t := val.Type().Field(i)
		v := val.Field(i)
		tags := t.Tag
		kind := v.Kind() // fundamental type "int64"
		//pkind := t.Type.String() // named type "argus.Status"

		// use lower cased field name
		name := prefix + strings.ToLower(t.Name)

		conv := tags.Get("cfconv")

		// currently the only tags known are cfconv:{dotsev,timespec}

		if conv == "dotsev" && kind == reflect.Array {
			cf.initDotSev(name, v)
			continue
		}

		// get configured value
		cval, ok := cf.Get(name)
		if !ok {
			// nothing configured - keep default
			continue
		}

		cf.setValue(v, cval, conv, name)
	}
}

// configure thing.severity -> thing[sev]
func (cf *CF) initDotSev(name string, v reflect.Value) {

	for sev := argus.CLEAR; sev <= argus.CRITICAL; sev++ {

		if int(sev) >= v.Len() {
			break
		}

		fname := name + "." + sev.String()
		cval, ok := cf.Get(fname)
		if !ok {
			continue
		}

		cf.setValue(v.Index(int(sev)), cval, "", fname)
	}
}

func (cf *CF) setValue(v reflect.Value, cval interface{}, conv string, name string) {

	pkind := v.Type().String()

	dl.Debug("name %s conv %s type %s = %s", name, conv, pkind, cval)

	switch cval := cval.(type) {
	case string:
		switch pkind {
		case "int", "int64", "int32":
			if conv == "timespec" {
				ts, err := argus.Timespec(cval)
				if err != nil {
					cf.Error("invalid timespec")
					return
				}
				v.SetInt(ts)
			} else {
				i, _ := strconv.ParseInt(cval, 10, 0)
				v.SetInt(i)
			}
		case "bool":
			v.SetBool(argus.CheckBool(cval))
		case "float64", "float32":
			f, _ := strconv.ParseFloat(cval, 64)
			v.SetFloat(f)
		case "string":
			v.SetString(cval)
		case "argus.Status":
			v.SetInt(int64(statusValue(cval)))
		case "darp.Gravity":
			v.SetInt(int64(gravityValue(cval)))
		default:
			diag.Problem("BUG? cannot configure field '%s', type '%s'", name, pkind)
		}

	default:
		// same types?
		tv := reflect.ValueOf(cval)
		tt := tv.Type().String()

		if pkind == tt {
			v.Set(tv)
		} else {
			diag.Problem("BUG? cannot configure field '%s', type %s != %s", name, pkind, tv)
		}
	}
}

func (cf *CF) Error(e string, args ...interface{}) {
	// RSN - error log with file,line, set errorflags, ...
	diag.Problem(e)
}
func (cf *CF) Warning(e string, args ...interface{}) {

}

func (cf *CF) Get(name string) (interface{}, bool) {
	return cf.iGet(name, true)
}
func (cf *CF) iGet(name string, ui bool) (interface{}, bool) {

	cv := cf.Param[name]
	if cv != nil {
		if cv.Inherit || ui {
			// 'param!: value' - only used where declared, will not be inherited by sub-blocks
			cv.Used = true
			return cv.Value, true
		}
	}

	cc, ok := cf.cache[name]
	if ok {
		return cc.value, true
	}

	if cf.parent != nil {
		v, ok := cf.parent.iGet(name, false)
		cf.cache[name] = cfcv{v, ok}
		return v, ok
	}

	return "", false
}
