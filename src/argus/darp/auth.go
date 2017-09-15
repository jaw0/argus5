// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Sep-14 23:57 (EDT)
// Function: darp authentication

package darp

import (
	"crypto/hmac"
	"crypto/sha256"

	"argus/argus"
)

func (*DarpServerer) Auth(name string, nonce string, digest string) bool {

	d := allDarp[name]

	if d == nil {
		return false
	}

	expect := authDigest(d.Pass, nonce)

	if expect == digest {
		return true
	}

	return false
}

func authDigest(pass string, nonce string) string {

	// same as the previous version of argus, which is based on APOP
	// but now with modern crypto
	h := hmac.New(sha256.New, []byte(pass))
	h.Write([]byte(nonce))
	bin := h.Sum(nil)

	return argus.Encode64Url(string(bin))
}
