// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Sep-13 19:29 (EDT)
// Function: web templates

package web

import (
	"fmt"
	"html/template"
	"net/http"
	"path/filepath"
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
	Login_Notice    string
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
			serveTemplate(loadTemplates(), true, ctx)
		})
	} else {
		t := loadTemplates()
		Add(PUBLIC, HTPATH, func(ctx *Context) {
			serveTemplate(t, false, ctx)
		})
	}
}

func loadTemplates() map[string]*template.Template {

	cf := config.Cf()

	if cf.Htdir == "" {
		return nil
	}

	htdocs, _ := filepath.Glob(cf.Htdir + "/htdocs/*")
	dashes, _ := filepath.Glob(cf.Htdir + "/dash/*")
	htdocs = append(htdocs, dashes...)

	var partials []string
	var pages []string

	for _, f := range htdocs {
		n := filepath.Base(f)

		if n[0] == '.' || n[0] == '#' {
			// skip editor temp files
			continue
		}
		if n[0] == '_' {
			partials = append(partials, f)
		} else {
			pages = append(pages, f)
		}
	}

	tm := make(map[string]*template.Template)

	for _, f := range pages {
		n := filepath.Base(f)
		t := template.New("view").Delims("{[", "]}")
		t.ParseFiles(append(partials, f)...)
		tm[n] = t
	}

	return tm
}

func serveTemplate(tm map[string]*template.Template, devmode bool, ctx *Context) {

	name := strings.TrimPrefix(ctx.R.URL.Path, HTPATH)

	// disallow access to partials (leading underscore)
	if len(name) > 0 && name[0] == '_' {
		http.NotFound(ctx.W, ctx.R)
		return
	}
	// valid? (otherwise, an ugly error)
	t := tm[name]
	if t == nil {
		dl.Debug("no such page %s", name)
		http.NotFound(ctx.W, ctx.R)
		return
	}

	user := ""
	home := ""
	objtitle := name
	query := make(map[string]string)

	// copy query params
	for k, v := range ctx.R.Form {
		if len(v) > 0 {
			query[k] = v[0]
		}
	}

	if ctx.User != nil {
		user = ctx.User.Name
		home = ctx.User.Home
	}

	if obj, ok := query["obj"]; ok {
		objtitle = obj
	}

	// make data available to template
	dat := map[string]interface{}{
		"User":            user,
		"Home":            home,
		"Header":          template.HTML(webConf.Header),
		"Header_Branding": template.HTML(webConf.Header_Branding),
		"Footer":          template.HTML(webConf.Footer),
		"Login_Notice":    template.HTML(webConf.Login_Notice),
		"Host":            ctx.R.Host,
		"Token":           ctx.XSRFToken,
		"DevMode":         devmode,
		"ObjTitle":        objtitle, // object name (if exists), or page name
		"Q":               query,
		"Argus": struct {
			Version string
			Url     string
		}{argus.Version, argus.URL},
	}

	err := t.ExecuteTemplate(ctx.W, "_base", dat)

	if err != nil {
		fmt.Fprintf(ctx.W, "ERROR: page %s: %v", name, err)
		dl.Verbose("template error: page %s: %v", name, err)
	}
}
