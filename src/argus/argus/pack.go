// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Sep-01 19:13 (EDT)
// Function: just enough of perl's unpack for compat

package argus

// only support: x# followed by one: A# c C n N Z
func Unpack(how string, dat []byte) (int32, bool) {

	if how == "" {
		return 0, false
	}

	for len(how) != 0 && how[0] == ' ' {
		// eat white
		how = how[1:]
	}

	if len(how) != 0 && how[0] == 'x' {
		skip := 0
		how = how[1:]
		for how != "" {
			if how[0] >= '0' && how[0] <= '9' {
				skip *= 10
				skip += int(how[0] - '0')
				how = how[1:]
			} else {
				break
			}
		}
		if skip == 0 {
			skip = 1
		}
		dat = dat[skip:]
	}

	for len(how) != 0 && how[0] == ' ' {
		// eat white
		how = how[1:]
	}

	if how == "" {
		return 0, false
	}

	switch how[0] {
	case 'c': // signed
		if len(dat) < 1 {
			return 0, true
		}
		v := int32(dat[0])
		if v > 127 {
			v = int32(v) - 256
		}
		return v, true
	case 'C': // unsigned
		if len(dat) < 1 {
			return 0, true
		}
		return int32(dat[0]), true
	case 'n':
		if len(dat) < 2 {
			return 0, true
		}
		return int32(dat[0])<<8 | int32(dat[1]), true
	case 'N':
		if len(dat) < 4 {
			return 0, true
		}
		return int32(dat[0])<<24 | int32(dat[1])<<16 | int32(dat[2])<<8 | int32(dat[3]), true
	}

	return 0, false

}
