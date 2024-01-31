// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Aug-31 00:08 (EDT)
// Function: web/darp/control/...

package api

import (
	"bufio"
	"net"

	"github.com/jaw0/acdiag"
)

type APIWriter interface {
	SetStatus(int)
	SetHeader(string, string)
	Write([]byte) (int, error)
}

type Context struct {
	doer   Serverer
	Authed bool
	User   string
	Nonce  string
	Method string // api method called
	Args   map[string]string
	Conn   net.Conn
	bfd    *bufio.Reader
}

type T struct {
	needauth bool
	f        ApiHandlerFunc
}

type ApiHandlerFunc func(*Context)
type Routes map[string]*T

var router = make(Routes)
var dl = diag.Logger("api")

func Add(authreq bool, path string, f ApiHandlerFunc) bool {

	if _, ok := router[path]; ok {
		diag.Fatal("duplicate api path: %s", path)
	}

	router[path] = &T{authreq, f}

	return true
}

// ################################################################

func (ctx *Context) dispatch() bool {

	t := router[ctx.Method]
	if t == nil {
		ctx.SendResponseFinal(404, "Not Found")
		return true
	}

	if t.needauth && !ctx.Authed {
		ctx.SendResponseFinal(403, "Not Authorized")
		return true
	}

	t.f(ctx)

	return true
}
