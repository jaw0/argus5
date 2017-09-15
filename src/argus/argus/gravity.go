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
	GRAV_IETF Gravity = 3
	GRAV_SELF Gravity = 4
)

var gravityname = []string{
	"up", "down", "vote", "ietf", "self",
}

func (g Gravity) String() string {
	if g < GRAV_UP || g > GRAV_VOTE {
		return "invalid"
	}
	return gravityname[int(g)]
}
