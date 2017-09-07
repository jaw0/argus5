// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Sep-06 16:28 (EDT)
// Function:

package argus

type Gravity int

const (
	GRAV_UP   Gravity = 0
	GRAV_DN   Gravity = 1
	GRAV_VOTE Gravity = 2
)

var gravityname = []string{
	"up", "down", "vote",
}

func (g Gravity) String() string {
	return gravityname[int(g)]
}
