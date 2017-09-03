// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Aug-30 23:46 (EDT)
// Function: lfsr

package lfsr

func Next16(x int) int {

	if x == 0 {
		x++
	}

	// lfsr: 16, 14, 13, 11
	bit := ((x >> 15) ^ (x >> 13) ^ (x >> 12) ^ (x >> 10)) & 1
	x = (x << 1) | bit
	x = x & 0xFFFF

	return x
}
