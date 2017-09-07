// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Sep-05 19:48 (EDT)
// Function: read/parse schedules

package construct

import (
	"strconv"
	"strings"

	"argus/argus"
	"argus/configure"
)

var dayNo = map[string]int{"sun": 0, "mon": 1, "tue": 2, "wed": 3, "thu": 4, "fri": 5, "sat": 6}

// schedule:? name {

func readSchedule(f *Files, cf *configure.CF, spec string) bool {

	sched := &argus.Schedule{}

	// parse spec line
	delim := strings.IndexAny(spec, " \t")
	if delim == -1 {
		cf.Error("cannot parse schedule")
		dl.Fatal("sched %s %X", spec, spec)
		eatConf(f, spec)
		return true
	}
	end := strings.LastIndexAny(spec, " \t")
	if end == delim {
		end = len(spec) - 1
	}
	name := "schedule_" + strings.Trim(spec[delim:end], " \t")

	for {
		l, ok := f.NextLine()
		if !ok {
			cf.Error("end-of-file while reading schedule")
			return false
		}

		if l == "}" {
			break
		}

		// parse sched
		ok = parseSchedLine(cf, sched, l)

		if !ok {
			cf.Error("cannot parse schedule")
			return true
		}
	}

	setParam(f, cf, name, sched)

	return true
}

// dow [start] [- end] => value
func parseSchedLine(cf *configure.CF, sched *argus.Schedule, line string) bool {

	fld := strings.Fields(line)
	var dow, res string
	var idow, start, end int

	if len(fld) < 3 {
		return false
	}
	if fld[len(fld)-2] != "=>" {
		return false
	}

	res = fld[len(fld)-1]
	dow = fld[0]

	if len(fld) > 3 {
		start, _ = strconv.Atoi(fld[1])
	}
	if len(fld) > 5 {
		end, _ = strconv.Atoi(fld[3])
	}

	var ok bool

	if dow == "*" || dow == "all" {
		idow = -1
	} else if idow, ok = dayNo[dow]; !ok {
		cf.Error("invalid day of week '%s' in schedule", dow)
		return false
	}

	dl.Debug("sched: %d %d %d -> %s", idow, start, end, res)
	sched.Append(idow, start, end, res)
	return true
}
