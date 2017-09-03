// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Sep-01 11:09 (EDT)
// Function:

package argus

func CheckBool(v string) bool {

	switch v {
	case "yes", "YES", "on", "ON", "true", "TRUE", "1":
		return true
	}
	return false
}
