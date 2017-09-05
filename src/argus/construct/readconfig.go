// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Sep-04 14:14 (EDT)
// Function: read argus config

package construct

import (
	"strings"

	"argus/configure"
	"argus/diag"
	"argus/monel"
)

type readConf struct {
	quot  int
	onel  bool
	level int
	//cfunc func(f *Files, cf *configure.CF, m *monel.M) bool
	subs string //map[string]bool
}

var dl = diag.Logger("dozer")

var confconf = map[string]readConf{
	"group":   readConf{quot: 1, level: 2, subs: "group host service alias schedule"},
	"host":    readConf{quot: 1, level: 2, subs: "group host service alias schedule"},
	"alias":   readConf{quot: 2, onel: true, level: 2},
	"service": readConf{onel: true, level: 2, subs: "schedule"},

	"method":  readConf{quot: 1, onel: true, level: 1, subs: "schedule"},
	"snmpoid": readConf{onel: true, level: 1},
}

func ReadConfig(file string) {

	f := NewReader(file)
	cf := &configure.CF{
		Type: "top",
		Name: "Top",
		File: f.CurrFile(),
	}

	readBlock(f, cf, nil)
}

func readKVP(f *Files, cf *configure.CF) bool {

	for {
		l, ok := f.NextLine()
		if !ok {
			return false
		}

		if l == "}" {
			f.UnGetLine(l)
			return true
		}

		word := firstWord(l)
		wrcf := wordConf(word)

		if word == "schedule" {
			readSchedule(f, cf)
			continue
		}

		if wrcf != nil {
			f.UnGetLine(l)
			return true
		}

		if strings.IndexByte(l, ':') != -1 {
			addParam(f, cf, l)
			continue
		}

		cf.Error("invalid entry in config file, %s not permitted in $class: '%s'", word, cf.Type)
		return false
	}
	return true
}

func readBlock(f *Files, cf *configure.CF, parent *monel.M) bool {

	var nob *monel.M
	opt := confconf[cf.Type]
	level := 0

	for {
		l, ok := f.NextLine()
		if !ok {
			return false
		}

		if l == "}" {
			return true
		}

		word := firstWord(l)
		wrcf := wordConf(word)

		if word == "schedule" {
			// read schedule
			continue
		}

		if opt != nil && wrcf != nil { // && permitted in opt
			cf.UnGetLine(l)

			if wrcf.level < level {
				if wrcf.level == 2 {
					cf.Error("%s block must appear before any Groups or Services", word)
				} else if wrcf.level == 3 {
					cf.Error("%s block must appear after any Groups or Services", word)
				}
				return false
			}

			if nob == nil { // and opt.isMonEl
				// build new object
			}

			level = wrcf.level

			readConfig(f, cf, nob)
			continue
		}

		if strings.IndexByte(l, ':') != -1 {
			addParam(f, cf, l)
			continue
		}

		cf.Error("invalid entry in config file, %s not permitted in $class: '%s'", word, cf.Type)
		return false
	}

	return true
}

func readConfig(f *Files, cf *configure.CF, m *monel.M) bool {

	l = f.Readline()
	// parse line

	cf := &configure.CF{
		File: f.CurrFile(),
	}

	// readBlock

	return true
}

func addParam(f *Files, cf *configure.CF, l string) bool {

	colon := strings.IndexByte(l, ':')
	if colon == -1 {
		return false
	}
	key := strings.Trim(l[:colon], " \t")
	val := strings.Trim(l[colon+1:], " \t")
	inh := true

	// key!: value -> non-inherited
	if key[len(key)-1] == '!' {
		key = key[:len(key)-1]
		inh = false
	}

	if _, have := cf.Param[key]; have {
		cf.Warning("redefinition of parameter '%s'", key)
	}

	cf.Param[key] = &configure.CFV{
		Value:   val,
		Line:    f.CurrLine(),
		Inherit: inh,
	}

	return true
}

func readSchedule(f *Files, cf *configure.CF) bool {

}

func firstWord(l string) string {

	delim := strings.IndexAny(l, " \t:")

	if delim == -1 {
		return nil
	}

	return strings.ToLower(l[:delim])
}

func wordConf(word string) {

	wcf := confconf[word]
	return wcf
}

/*
readkvp(f,cf) - read kvp+sched into cf

readInfoConfigs(f) - read methods+snmpoids
readMonConfigs(f, parent, groupsonly) - read groups/alias/service into parent


*/
