// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Sep-01 21:09 (EDT)
// Function:

package argus

import (
	"fmt"
	"strconv"
)

// convert friendly time specifiers to seconds
func Timespec(v string, def int) (int64, error) {

	var tot int64

	for v != "" {

		i := 0
		for i = 0; i < len(v); i++ {
			c := v[i]
			if c >= '0' && c <= '9' {
				continue
			}
			break
		}

		if i == len(v) {
			t, _ := strconv.ParseInt(v, 10, 64)
			v = v[i:]
			tot += t * int64(def)
		} else {
			t, _ := strconv.ParseInt(v[:i], 10, 64)

			switch v[i] {
			case 's':
				break
			case 'm':
				t *= 60
			case 'h':
				t *= 3600
			case 'd':
				t *= 24 * 3600
			case 'w':
				t *= 7 * 24 * 3600
			case 'M':
				t *= 30 * 24 * 3600
			case 'y':
				t *= 365 * 24 * 3600
			default:
				return 0, fmt.Errorf("invalid timespec '%s'", v[:i])
			}
			v = v[i+1:]
			tot += t
		}

	}

	return tot, nil
}

func Elapsed(e int64) string {

	sec := e % 60
	e /= 60
	min := e % 60
	e /= 60
	hrs := e % 24
	e /= 24
	day := e % 30 // -ish
	e /= 30
	mon := e

	if mon > 0 {
		return fmt.Sprintf("%dM %dd %0.2d:%0.2d", mon, day, hrs, min)
	}
	if day > 0 {
		return fmt.Sprintf("%dd %0.2d:%0.2d", day, hrs, min)
	}
	return fmt.Sprintf("%0.2d:%0.2d:%0.2d", hrs, min, sec)
}
