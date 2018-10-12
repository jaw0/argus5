// Copyright (c) 2018
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2018-Oct-12 14:06 (EDT)
// Function: debugging dump

package argus

import (
	"fmt"
	"reflect"
)

const MAXLEN = 200

type Dumper interface {
	Dump(string, string)
}

func Dump(dx Dumper, prefix string, obj interface{}) {

	var val = reflect.ValueOf(obj)

	switch val.Kind() {
	case reflect.Interface, reflect.Ptr:
		val = val.Elem()
	}

	switch val.Kind() {
	case reflect.Struct:

		for i := 0; i < val.NumField(); i++ {

			t := val.Type().Field(i)
			v := val.Field(i)

			vs := fmt.Sprintf("%v", v)

			if len(vs) > MAXLEN {
				vs = fmt.Sprintf("<large object, type %s>", t.Type.String())
			}

			if prefix == "" {
				dx.Dump(t.Name, vs)
			} else {
				name := prefix + "/" + t.Name
				dx.Dump(name, vs)
			}
		}
	case reflect.Invalid:
		dx.Dump(prefix, "<nil>")

	default:
		dx.Dump(prefix, fmt.Sprintf("%v", val))
	}
}
