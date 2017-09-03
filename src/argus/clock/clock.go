// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Aug-30 15:01 (EDT)
// Function: argus time

package clock

import (
	"time"
)

func Unix() int64 {
	return Now().Unix()
}

func Milli() int64 {
	return Nano() / 1000000
}
func Micro() int64 {
	return Nano() / 1000
}

func Nano() int64 {
	return Now().UnixNano()
}

func Now() time.Time {
	t := time.Now()

	return t
}
