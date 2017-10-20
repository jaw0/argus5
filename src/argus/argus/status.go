// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Aug-31 23:49 (EDT)
// Function:

package argus

import (
	"strings"
)

type Status int

const (
	UNKNOWN  Status = 0
	CLEAR    Status = 1
	WARNING  Status = 2
	MINOR    Status = 3
	MAJOR    Status = 4
	CRITICAL Status = 5
	OVERRIDE Status = 6
	DEPENDS  Status = 7

	MAXSTATUS Status = 7
)

var statusname = []string{
	"unknown", "clear", "warning", "minor", "major", "critical", "override", "depends",
}

func (s Status) String() string {

	if s < UNKNOWN || s > MAXSTATUS {
		return "invalid"
	}
	return statusname[int(s)]
}

func StatusValue(name string) Status {

	switch strings.ToLower(name) {
	case "clear":
		return CLEAR
	case "warning":
		return WARNING
	case "minor":
		return MINOR
	case "major":
		return MAJOR
	case "critical":
		return CRITICAL

	}
	return UNKNOWN
}
