// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Sep-01 11:09 (EDT)
// Function:

package argus

import (
	"encoding/base64"
	"encoding/hex"
	"net/url"
)

func Encode(s string) string {
	return encode(s, '~')
}

func UrlEncode(s string) string {
	return encode(s, '%')
}

func UrlDecode(s string) string {

	r, err := url.QueryUnescape(s)
	if err != nil {
		return ""
	}
	return r
}

func HexStr(s string) string {
	return hex.EncodeToString([]byte(s))
}

func Encode64(s string) string {
	return base64.StdEncoding.EncodeToString([]byte(s))
}

func Decode64(s string) string {

	r, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return ""
	}
	return string(r)
}

func Encode64Url(s string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(s))
}

// ################################################################

// modeled after net/url escape, but simpler
func shouldEscape(c byte) bool {

	switch c {
	case ' ', '"', '%':
		return true
	}

	if c <= ' ' || c >= 127 {
		return true
	}

	return false
}

func encode(s string, pct byte) string {

	hexCount := 0
	slen := len(s)

	for i := 0; i < slen; i++ {
		if shouldEscape(s[i]) {
			hexCount++
		}
	}

	if hexCount == 0 {
		return s
	}

	t := make([]byte, len(s)+2*hexCount)
	j := 0

	for i := 0; i < slen; i++ {
		c := s[i]
		if shouldEscape(c) {
			t[j] = pct
			t[j+1] = "0123456789ABCDEF"[c>>4]
			t[j+2] = "0123456789ABCDEF"[c&15]
			j += 3
		} else {
			t[j] = c
			j++
		}
	}
	return string(t)
}
