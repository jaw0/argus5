// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Sep-13 20:58 (EDT)
// Function: home + login

package web

import (
	"net/http"

	"argus.domain/argus/users"
)

func init() {
	Add(PUBLIC, "/", webHome)
	Add(PUBLIC, "/login-1", webLogin1)
	Add(PRIVATE, "/logout", webLogout)
	Add(PRIVATE, "/hush", webHush)
}

func webHome(ctx *Context) {

	if ctx.User != nil {
		http.Redirect(ctx.W, ctx.R, ctx.webHome(), 302)
	} else {
		http.Redirect(ctx.W, ctx.R, "/view/login", 302)
	}
}

func webLogin1(ctx *Context) {

	name := ctx.Get("name")
	pass := ctx.Get("pass")
	next := ctx.Get("next")

	u := users.CheckUserPasswd(name, pass)

	if u != nil {
		ctx.NewSession(name)
		ctx.User = u

		if next == "" {
			next = ctx.webHome()
		}

		dl.Verbose("login success '%s' from %s'", name, ctx.R.RemoteAddr)
		http.Redirect(ctx.W, ctx.R, next, 302)
		return
	}

	// RSN - rate limit on failures
	http.Redirect(ctx.W, ctx.R, "/view/login?fail=1", 302)

	dl.Verbose("login failure '%s' from %s'", name, ctx.R.RemoteAddr)
}

func webLogout(ctx *Context) {

	DelSession(ctx)
	http.Redirect(ctx.W, ctx.R, "/view/login", 302)
}

func webHush(ctx *Context) {
	Hush(ctx)
}

func (ctx *Context) webHome() string {

	home := ctx.User.Home

	if home == "" {
		home = "Top"
	}

	return "/view/home?obj=" + home
}
