// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Aug-31 00:08 (EDT)
// Function: web/darp/control/...

package api

import (
	"net/http"
	"net/url"

	"argus/diag"
)

// QQQ?
type Context struct {
	Authed bool
	User   string
	W      http.ResponseWriter
	Req    *http.Request
	Args   url.Values // aka map[string][]string
}

type T struct {
	needauth bool
	f        ApiHandlerFunc
}

type ApiHandlerFunc func(*Context)
type Routes map[string]T

var router = make(Routes)

func Add(authreq bool, path string, f ApiHandlerFunc) bool {

	if _, ok := router[path]; ok {
		diag.Fatal("duplicate api path: %s", path)
	}

	router[path] = T{authreq, f}

	return true
}

func (c *Context) Param(n string) string {
	return c.Args.Get(n)
}

func (c *Context) Write(p []byte) (int, error) {
	return c.W.Write(p)
}
