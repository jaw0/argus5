// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Sep-04 13:55 (EDT)
// Function:

package argus

import (
	"argus/diag"
)

var haveErrors = false
var haveWarns = false

func Loggit(msg string, args ...interface{}) {

	// main log
	// diag.verbose

	diag.Verbose(msg, args...)

	// open datadir/log
	// [date] msg
	// close

}

func ConfigError(file string, line int, fmt string, args ...interface{}) {

	haveErrors = true

	msg := "ERROR: in file %s on line %d: " + fmt
	arg := []interface{}{file, line}
	arg = append(arg, args...)
	Loggit(msg, arg...)

}
func ConfigWarning(file string, line int, fmt string, args ...interface{}) {

	haveWarns = true

	msg := "UH OH: in file %s on line %d: " + fmt
	arg := []interface{}{file, line}
	arg = append(arg, args...)
	Loggit(msg, arg...)

}
