// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Sep-06 16:28 (EDT)
// Function:

package argus

import (
	"strings"
)

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
	if g < GRAV_UP || g > GRAV_SELF {
		return "invalid"
	}
	return gravityname[int(g)]
}

func GravityValue(name string) Gravity {

	switch strings.ToLower(name) {
	case "up":
		return GRAV_UP
	case "down":
		return GRAV_DN
	case "vote":
		return GRAV_VOTE
	case "ietf":
		return GRAV_IETF
	case "self":
		return GRAV_SELF
	}

	return GRAV_UP
}
