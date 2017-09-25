// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Sep-17 12:11 (EDT)
// Function: tag matching

package argus

func IncludesTag(tags string, tag string, wildcard bool) bool {

	for i := 0; i < len(tags); i++ {
		c := tags[i]
		// skip space
		if c == ' ' || c == '\t' {
			continue
		}
		// compare
		//   find end of tag
		eow := i + 1
		for ; eow < len(tags) && tags[eow] != ' ' && tags[eow] != '\t'; eow++ {
		}
		tt := tags[i:eow]

		if tt == tag {
			return true
		}
		if wildcard && (tt == "all" || tt == "*") {
			return true
		}

		// skip ahead to space
		for ; i < len(tags) && tags[i] != ' ' && tags[i] != '\t'; i++ {
		}
	}

	return false
}

func ACLPermitsUser(acl string, creds []string) bool {

	for _, cred := range creds {
		if IncludesTag(acl, cred, false) {
			return true
		}
	}
	return false
}
