// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Sep-18 22:16 (EDT)
// Function: debugging

package api

import (
	"fmt"
	"reflect"
)

const MAXLEN = 200

func (ctx *Context) DumpStruct(obj interface{}, prefix string) {

	var val = reflect.ValueOf(obj)

	switch val.Kind() {
	case reflect.Interface, reflect.Ptr:
		val = val.Elem()
	}

	for i := 0; i < val.NumField(); i++ {

		t := val.Type().Field(i)
		v := val.Field(i)

		vs := fmt.Sprintf("%v", v)

		if len(vs) > MAXLEN {
			vs = fmt.Sprintf("<large object, type %s>", t.Type.String())
		}

		name := prefix + t.Name

		ctx.SendKVP(name, vs)
	}
}
