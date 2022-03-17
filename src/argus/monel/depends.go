// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Sep-06 19:19 (EDT)
// Function: dependancies

package monel

import (
	"strings"

	"argus.domain/argus/argus"
)

func (m *M) resolveDepends() {

	if m.Cf.Depends == "" {
		return
	}
	d := strings.Fields(m.Cf.Depends)
	if len(d) == 0 {
		return
	}

	for _, name := range d {
		if name == m.Cf.Unique {
			m.ConfCF.Error("cannot depend on self")
			continue
		}

		o := Find(name)

		if o == nil {
			m.ConfCF.Error("cannot resolve dependancy '%s'", name)
			continue
		}

		m.Depends = append(m.Depends, name)
	}
}

func (m *M) checkDepends() bool {

	for _, name := range m.Depends {
		o := Find(name)
		if o == nil {
			continue
		}
		_, st := o.Status()
		if st != argus.CLEAR && st != argus.UNKNOWN {
			m.P.OvStatus = argus.DEPENDS
			m.P.Alarm = false
			m.P.Culprit = name
			// and add m to o.dependsOnMe?
			return true
		}
	}
	return false
}
