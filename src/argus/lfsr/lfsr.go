// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Aug-30 23:46 (EDT)
// Function: lfsr

package lfsr

func Next16(x int) int {

	if x == 0 {
		x++
	}

	// lfsr: B465
	bit := ((x >> 15) ^ (x >> 13) ^ (x >> 12) ^ (x >> 10) ^ (x >> 6) ^ (x >> 5) ^ (x >> 2) ^ (x)) & 1
	x = (x << 1) | bit
	x = x & 0xFFFF

	return x
}
