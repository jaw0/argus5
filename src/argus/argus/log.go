// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Sep-04 13:55 (EDT)
// Function: startup/config errors

package argus

import (
	"fmt"

	"github.com/jaw0/acdiag"
)

const MAXLOGS = 100

type logMsg struct {
	Tag string
	Msg string
}

var haveErrors = false
var haveWarns = false
var logmsgs []logMsg

func HasErrors() bool {
	return haveErrors
}
func HasWarnings() bool {
	return haveWarns
}
func LogMsgs() []logMsg {
	return logmsgs
}

func Loggit(tag string, msg string, args ...interface{}) {

	diag.Verbose(msg, args...)

	txt := fmt.Sprintf(msg, args...)

	if len(logmsgs) < MAXLOGS {
		logmsgs = append(logmsgs, logMsg{tag, txt})
	}
}

func ConfigError(file string, line int, fmt string, args ...interface{}) {

	haveErrors = true

	msg := "ERROR: in file %s on line %d: " + fmt
	arg := []interface{}{file, line}
	arg = append(arg, args...)
	Loggit("logerror", msg, arg...)

}
func ConfigWarning(file string, line int, fmt string, args ...interface{}) {

	haveWarns = true

	msg := "WARNING: in file %s on line %d: " + fmt
	arg := []interface{}{file, line}
	arg = append(arg, args...)
	Loggit("logwarning", msg, arg...)

}
