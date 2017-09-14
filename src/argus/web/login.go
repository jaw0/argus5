// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Sep-13 20:58 (EDT)
// Function: home + login

package web

import (
	"net/http"
)

func init() {
	Add(false, "/", webLogin)
}

func webLogin(ctx *Context) {

	if ctx.User != "" {
		// RSN - user.home, ...
		http.Redirect(ctx.W, ctx.R, "/view/home", 302)
	} else {
		http.Redirect(ctx.W, ctx.R, "/view/login", 302)
	}
}
