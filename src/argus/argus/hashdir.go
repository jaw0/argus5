// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Sep-04 10:24 (EDT)
// Function: hashed directory name

package argus

import (
	"fmt"
)

func HashDjb2(s string) int {
	h := 5381

	for i := 0; i < len(s); i++ {
		h = 33*h + int(s[i])
	}
	return h & 0x7fffffff
}

func HashedDirectory(file string) string {
	h := HashDjb2(file) & 0xffffff
	a := (h % 26) + 'A'
	h >>= 5
	b := (h % 26) + 'A'

	return fmt.Sprintf("%c/%c", a, b)
}
