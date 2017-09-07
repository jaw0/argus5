// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Aug-31 23:49 (EDT)
// Function:

package argus

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

type colors struct {
	fg string
	bg string
	bk string
}

var conf = []colors{
	CLEAR:    {"22AA22", "33DD33", "88ff88"}, // green
	WARNING:  {"0088DD", "88DDFF", ""},       // blue
	MINOR:    {"CCCC00", "FFFF00", ""},       // yellow
	MAJOR:    {"DD9900", "FFBB44", ""},       // orange
	CRITICAL: {"CC0000", "FF4444", "ff8888"}, // red
	OVERRIDE: {"888888", "DDDDDD", ""},       // gray
	DEPENDS:  {"DD9900", "FFCC44", ""},       // orange
	UNKNOWN:  {"BB44EE", "DD99FF", ""},       // purple
}
var statusname = []string{
	"unknown", "clear", "warning", "minor", "major", "critical", "override", "depends",
}

func (s Status) String() string {
	return statusname[s]
}

func (s Status) ColorFG() string {
	return conf[s].fg
}

func (s Status) ColorBG() string {
	return conf[s].bg
}

func (s Status) ColorBulk() string {
	c := conf[s].bk
	if c != "" {
		return c
	}
	return conf[s].bg
}
