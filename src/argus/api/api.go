// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Aug-31 00:08 (EDT)
// Function: web/darp/control/...

package api

import (
	//"net/http"
	"net/url"

	"argus/diag"
)

const (
	PUBLIC  = 0 // publicly accessible
	AUTHED  = 1 // must be authenticated
	PRIVATE = 2 // only via control, not http
)

type APIWriter interface {
	SetStatus(int)
	SetHeader(string, string)
	Write([]byte) (int, error)
}

type Context struct {
	Authed bool
	User   string
	Method string     // api method called
	Args   url.Values // aka map[string][]string

	W APIWriter
	//W   http.ResponseWriter
	//Req *http.Request

}

type T struct {
	needauth int
	f        ApiHandlerFunc
}

type ApiHandlerFunc func(*Context)
type Routes map[string]T

var router = make(Routes)
var dl = diag.Logger("api")

func Add(authreq int, path string, f ApiHandlerFunc) bool {

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
