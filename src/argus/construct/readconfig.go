// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Sep-04 14:14 (EDT)
// Function: read argus config

package construct

import (
	"strings"

	"argus/argus"
	"argus/configure"
	"argus/diag"
	"argus/monel"
	"argus/notify"
	"argus/service"
	"argus/web"
)

type readConf struct {
	narg   int
	onel   bool
	level  int
	isInfo bool
	permit map[string]bool
}

var dl = diag.Logger("dozer")

var confconf = map[string]*readConf{
	"top":     &readConf{narg: 1, level: 2, permit: map[string]bool{"method": true, "snmpoid": true, "group": true, "host": true, "darp": true, "agent": true}},
	"group":   &readConf{narg: 1, level: 2, permit: map[string]bool{"group": true, "host": true, "service": true, "alias": true}},
	"host":    &readConf{narg: 1, level: 2, permit: map[string]bool{"group": true, "host": true, "service": true, "alias": true}},
	"alias":   &readConf{narg: 2, onel: true, level: 2},
	"service": &readConf{narg: 1, onel: true, level: 2},
	"method":  &readConf{narg: 1, onel: true, level: 1, isInfo: true},
	"snmpoid": &readConf{onel: true, level: 1, isInfo: true},
	"darp":    &readConf{narg: 0, level: 1, isInfo: true},
	"agent":   &readConf{narg: 2, level: 1, isInfo: true},
	"resolv":  &readConf{},
}

func ReadConfig(file string) []string {

	f := NewReader(file)
	cf := configure.NewCF("group", "Top", nil)
	cf.File = f.CurrFile()
	cf.Line = 1

	readKVP(f, cf)
	top := Make(cf, nil)
	readConfigs(f, top, cf, "top")
	dl.Debug("done %v", f)

	notify.Configure(cf)
	web.Configure(cf)
	service.GraphConfig(cf)
	// other.Configure(cf)
	top.DoneConfig()

	return f.allfiles
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
			readSchedule(f, cf, l)
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

		if wrcf == nil {
			dl.Debug("wtf %s", word)
		}
		f.UnGetLine(l)
		return true
	}
	return true
}

func readConfigs(f *Files, parent *monel.M, pcf *configure.CF, ptype string) bool {

	opt := confconf[ptype]
	level := 0

	for {
		l, ok := f.NextLine()
		if !ok {
			dl.Debug("eof")
			return false
		}

		if l == "}" {
			dl.Debug("close} %s", ptype)
			return true
		}

		word := firstWord(l)
		wrcf := wordConf(word)

		if opt.permit[word] && wrcf != nil && wrcf.level >= level {
			level = wrcf.level

			readConfig(f, parent, pcf, l, wrcf)
			continue
		}

		errorMessage(f, pcf, word, wrcf, l, level)
		eatConf(f, l)

	}
}

func errorMessage(f *Files, pcf *configure.CF, word string, wrcf *readConf, line string, level int) {

	if wrcf == nil {
		argus.ConfigError(f.CurrFile(), f.CurrLine(), "I do not understand '%s'", word)
		return
	}

	if wrcf.level == 1 {
		argus.ConfigError(f.CurrFile(), f.CurrLine(), "%s block must appear before any Groups or Services", word)
		return
	}

	if strings.IndexByte(line, ':') != -1 {
		argus.ConfigError(f.CurrFile(), f.CurrLine(), "additional data not permitted here")
		return
	}

	argus.ConfigError(f.CurrFile(), f.CurrLine(), "invalid entry in config file '%s'", word)
}

func readConfig(f *Files, parent *monel.M, pcf *configure.CF, spec string, wrcf *readConf) bool {

	cf := parseSpec(f, pcf, wrcf, spec)

	if spec[len(spec)-1] == '{' {
		readKVP(f, cf)
		nob := Make(cf, parent)
		ok := readConfigs(f, nob, cf, cf.Type)
		return ok
	}

	Make(cf, parent)
	return true
}

func parseSpec(f *Files, pcf *configure.CF, wrcf *readConf, spec string) *configure.CF {

	if spec[len(spec)-1] == '{' {
		// remove final {
		spec = spec[:len(spec)-1]
	}

	// remove first word
	delim := strings.IndexAny(spec, " \t:")
	word := strings.ToLower(spec[:delim])
	spec = strings.TrimSpace(spec[delim+1:])

	var arg1, arg2 string

	if spec != "" {
		if wrcf.narg == 1 {
			arg1 = unquote(spec)
		} else {
			if spec[0] == '"' {
				e := strings.IndexByte(spec[1:], '"') + 1
				if e != 0 {
					arg1 = unquote(spec[0 : e+1])

					if e < len(spec)-1 {
						arg2 = unquote(spec[e+1:])
					}
				}
			} else {
				e := strings.Index(spec, " \t")

				if e != -1 {
					arg1 = strings.TrimSpace(spec[:e])
					arg2 = strings.TrimSpace(spec[e:])
				} else {
					arg1 = strings.TrimSpace(spec)
				}
			}
		}
	}

	cf := configure.NewCF(word, arg1, pcf)
	cf.File = f.CurrFile()
	cf.Line = f.CurrLine()
	cf.Extra = arg2

	return cf
}

func unquote(s string) string {

	if s == "" {
		return s
	}

	s = strings.TrimSpace(s)

	if s[0] == '"' {
		s = s[1:]
	}
	l := len(s)
	if s[l-1] == '"' {
		s = s[:l-1]
	}

	return s
}

func addParam(f *Files, cf *configure.CF, l string) bool {

	colon := strings.IndexByte(l, ':')
	if colon == -1 {
		return false
	}
	key := strings.TrimSpace(l[:colon])
	val := strings.TrimSpace(l[colon+1:])

	return setParam(f, cf, key, val)

}

func setParam(f *Files, cf *configure.CF, key string, val interface{}) bool {

	if _, have := cf.Param[key]; have {
		argus.ConfigWarning(f.CurrFile(), f.CurrLine(), "redefinition of parameter '%s'", key)
	}

	cf.Param[key] = &configure.CFV{
		Value: val,
		Line:  f.CurrLine(),
	}

	return true
}

func firstWord(l string) string {

	delim := strings.IndexAny(l, " \t:")

	if delim == -1 {
		return ""
	}

	return strings.ToLower(l[:delim])
}

func wordConf(word string) *readConf {

	wcf := confconf[word]
	return wcf
}

func eatConf(f *Files, l string) {

	if l[len(l)-1] == '{' {
		eatBlock(f)
	}
}

func eatBlock(f *Files) {

	for {
		l, ok := f.NextLine()
		if !ok {
			return
		}
		if l == "}" {
			return
		}
		if l[len(l)-1] == '{' {
			eatBlock(f)
		}
	}
}
