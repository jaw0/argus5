// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Sep-05 19:45 (EDT)
// Function: make things

package construct

import (
	"argus/configure"
	//"argus/diag"
	"argus/monel"
	"argus/service"
)

func Make(cf *configure.CF, parent *monel.M) *monel.M {

	dl.Debug("make %s; %s", cf.Type, cf.Name)

	switch cf.Type {
	case "service":
		s, err := service.New(cf, parent)
		if err != nil {
			dl.Debug(">>%v", err)
			cf.Error("%v", err)
		}
		return s
	case "host":
	case "group":
	case "alias":
	case "method":
	case "snmpoid":

	default:
		dl.Bug("unable to construct object of type '%s'", cf.Type)
	}
	return nil
}
