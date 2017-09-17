// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Sep-13 19:29 (EDT)
// Function: web templates

package web

import (
	"fmt"
	"html/template"
	"net/http"
	"strings"

	"argus/argus"
	"argus/config"
	"argus/configure"
)

const (
	HTPATH = "/view/"
)

// user configurable params (from config file)
type WebConf struct {
	Header          string
	Header_Branding string
	Footer          string
}

var webConf = WebConf{}

// load config from config file
func Configure(cf *configure.CF) {
	cf.InitFromConfig(&webConf, "web", "")
}

// called after the config has fully loaded
func Configured() {

	cf := config.Cf()

	// in dev mode, reload the templates on every page view
	if cf.DevMode {
		Add(PUBLIC, HTPATH, func(ctx *Context) {
			serveTemplate(loadTemplates(), ctx)
		})
	} else {
		t := loadTemplates()
		Add(PUBLIC, HTPATH, func(ctx *Context) {
			serveTemplate(t, ctx)
		})
	}
}

func loadTemplates() *template.Template {

	cf := config.Cf()

	if cf.Htdir == "" {
		return nil
	}

	// change the delimiters to avoid clash with js templates
	t := template.New("view").Delims("{[", "]}")
	t.ParseGlob(cf.Htdir + "/htdocs/*")
	t.ParseGlob(cf.Htdir + "/dash/*")

	return t
}

func serveTemplate(t *template.Template, ctx *Context) {

	name := strings.TrimPrefix(ctx.R.URL.Path, HTPATH)

	// disallow access to partials (leading underscore)
	if len(name) > 0 && name[0] == '_' {
		http.NotFound(ctx.W, ctx.R)
		return
	}
	// valid? (otherwise, an ugly error)
	if x := t.Lookup(name); x == nil {
		dl.Debug("no such page %s", name)
		http.NotFound(ctx.W, ctx.R)
		return
	}

	user := ""
	home := ""
	if ctx.User != nil {
		user = ctx.User.Name
		home = ctx.User.Home
	}

	// make data available to template
	query := make(map[string]string)
	dat := map[string]interface{}{
		"User":            user,
		"Home":            home,
		"Header":          template.HTML(webConf.Header),
		"Header_Branding": template.HTML(webConf.Header_Branding),
		"Footer":          template.HTML(webConf.Footer),
		"Host":            ctx.R.Host,
		"Q":               query,
		"Argus": struct {
			Version string
			Url     string
		}{argus.Version, argus.URL},
	}

	// copy query params
	for k, v := range ctx.R.Form {
		if len(v) > 0 {
			query[k] = v[0]
		}
	}

	err := t.ExecuteTemplate(ctx.W, name, dat)

	if err != nil {
		fmt.Fprintf(ctx.W, "ERROR: page %s: %v", name, err)
	}
}
