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
	Value interface{} // string | *Schedule
	Line  int
	Used  bool
}

type CF struct {
	parent *CF
	Type   string // service, group, alias
	Name   string // group name, monitor name
	Extra  string
	File   string // start of the block
	Line   int
	Param  map[string]*CFV
	cache  map[string]*CFV
}

var dl = diag.Logger("configure")

func NewCF(t string, n string, pcf *CF) *CF {

	return &CF{
		Type:   t,
		Name:   n,
		parent: pcf,
		Param:  make(map[string]*CFV),
		cache:  make(map[string]*CFV),
	}
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

		cf.setValue(v, conv, name)
	}
}

// configure thing.severity -> thing[sev]
func (cf *CF) initDotSev(name string, v reflect.Value) {

	dl.Debug("dot sev %s", name)
	for sev := argus.CLEAR; sev <= argus.CRITICAL; sev++ {

		if int(sev) >= v.Len() {
			break
		}

		fname := name + "." + sev.String()

		cf.setValue(v.Index(int(sev)), "", fname)
	}

	cf.setValue(v.Index(int(argus.UNKNOWN)), "", name)
}

func (cf *CF) setValue(v reflect.Value, conv string, name string) {

	// get configured value
	c := cf.Get(name)
	if c == nil {
		// nothing configured - keep default
		return
	}

	cval := c.Value

	pkind := v.Type().String()

	dl.Debug("name '%s' conv '%s' type '%s' = %s", name, conv, pkind, cval)

	switch cval := cval.(type) {
	case string:
		switch pkind {
		case "int", "int64", "int32":
			if conv == "timespec" {
				ts, err := argus.Timespec(cval, 1)
				if err != nil {
					// the error will report the file/line of the service when used
					// not where it actually is.
					// QQQ - should we spend the memory to track?
					cf.Error("invalid timespec '%s'", cval)
					return
				}
				v.SetInt(ts)
			} else {
				i, _ := strconv.ParseInt(cval, 10, 0)
				v.SetInt(i)
			}
		case "bool":
			v.SetBool(argus.CheckBool(cval))
		case "float", "float64", "float32":
			f, _ := strconv.ParseFloat(cval, 64)
			v.SetFloat(f)
		case "string":
			v.SetString(cval)
		case "argus.Status":
			v.SetInt(int64(statusValue(cval)))
		case "darp.Gravity", "argus.Gravity":
			v.SetInt(int64(gravityValue(cval)))
		case "*argus.Schedule":
			// convert string to always schedule
			v.Set(reflect.ValueOf(argus.ScheduleAlways(cval)))

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
	argus.ConfigError(cf.File, cf.Line, e, args...)
}
func (cf *CF) Warning(e string, args ...interface{}) {
	argus.ConfigWarning(cf.File, cf.Line, e, args...)
}

func (cf *CF) Get(name string) *CFV {
	return cf.iGet(name, true)
}
func (cf *CF) iGet(name string, useInherit bool) *CFV {

	if useInherit {
		// 'param!: value' - only used where declared, will not be inherited by sub-blocks
		cv := cf.Param[name+"!"]
		if cv != nil {
			cv.Used = true
			return cv
		}
	}

	cv := cf.Param[name]
	if cv != nil {
		cv.Used = true
		return cv
	}

	cc, ok := cf.cache[name]
	if ok {
		return cc
	}

	if cf.parent != nil {
		v := cf.parent.iGet(name, false)
		if cf.cache != nil {
			cf.cache[name] = v
		}
		return v
	}

	return nil
}

func (cf *CF) DrainCache() {
	cf.cache = nil
}

func (cf *CF) CheckTypos() {

	for key, cfv := range cf.Param {
		if cfv.Used {
			continue
		}
		argus.ConfigWarning(cf.File, cfv.Line, "unused parameter '%s' - typo?", key)
	}
}
