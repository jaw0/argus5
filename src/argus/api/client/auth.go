// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Oct-16 22:58 (EDT)
// Function: authenticate

package client

import (
	"crypto/hmac"
	"crypto/sha256"
	"fmt"
	"time"

	"argus/argus"
)

type Creds struct {
	Name string
	Pass string
}

func (c *Conn) Auth(creds *Creds, timeout time.Duration) error {

	resp, err := c.Get("auth", nil, timeout)
	if err != nil {
		return err
	}

	res := resp.Map()
	nonce := res["nonce"]

	digest := authDigest(creds.Pass, nonce)

	resp, err = c.Get("auth", map[string]string{
		"name":   creds.Name,
		"digest": digest,
	}, timeout)

	if err != nil {
		return err
	}

	if resp.Code != 200 {
		return fmt.Errorf("authentication with '%s' failed '%d %s'", creds.Name, resp.Code, resp.Msg)
	}
	return nil
}

func authDigest(pass string, nonce string) string {

	// same as the previous version of argus, which is based on APOP
	// but now with modern crypto
	h := hmac.New(sha256.New, []byte(pass))
	h.Write([]byte(nonce))
	bin := h.Sum(nil)

	return argus.Encode64Url(string(bin))
}
