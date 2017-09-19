// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Sep-17 12:34 (EDT)
// Function: access control

package monel

import (
	"argus/argus"
)

func ACLPermitsUser(acl string, creds []string) bool {

	for _, cred := range creds {
		if argus.IncludesTag(acl, cred, false) {
			return true
		}
	}
	return false
}
