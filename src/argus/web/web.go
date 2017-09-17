// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Sep-11 15:39 (EDT)
// Function: web serving

package web

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"argus/config"
	"argus/diag"
	"argus/users"
)

const (
	PUBLIC  = 0
	PRIVATE = 1
	WRITE   = 2
)

type Context struct {
	User      *users.User
	XSRFToken string
	W         http.ResponseWriter
	R         *http.Request
	// user pers, home, ...
}

type Server struct {
	services []*http.Server
	done     sync.WaitGroup
}

type WebHandlerFunc func(*Context)

var dl = diag.Logger("web")
var server *Server

func Init() {
	load() // load sessions
	server = Start()
}
func Stop() {
	server.Shutdown()
}

func Start() *Server {

	cf := config.Cf()
	s := &Server{}

	if cf.Port_http != 0 {
		dl.Verbose("starting http on :%d", cf.Port_http)
		www := s.httpServer(cf.Port_http)
		go func() {
			defer s.done.Done()
			www.ListenAndServe()
		}()
	}

	if cf.Port_https != 0 && cf.TLS_cert != "" && cf.TLS_key != "" {
		dl.Verbose("starting https on :%d", cf.Port_https)
		www := s.httpServer(cf.Port_https)
		go func() {
			defer s.done.Done()
			www.ListenAndServeTLS(cf.TLS_cert, cf.TLS_key)
		}()
	}

	// QQQ - different dir?
	if cf.Htdir != "" {
		// server static assets
		dir := cf.Htdir + "/static"
		dl.Debug("serving static on %s", dir)
		http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir(dir))))

	}

	return s
}

func (s *Server) httpServer(port int) *http.Server {
	www := &http.Server{
		Addr: fmt.Sprintf(":%d", port),
	}
	s.services = append(s.services, www)
	s.done.Add(1)
	return www
}

// see also: net/http Shutdown()
func (s *Server) Shutdown() {

	var wg sync.WaitGroup
	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)

	for _, ss := range s.services {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ss.Shutdown(ctx)
		}()
	}

	wg.Wait()
	s.done.Wait()
}

// ################################################################

// add routes
func Add(authreq int, path string, f WebHandlerFunc) {

	http.HandleFunc(path, httpAdapt(authreq, f))
}

// ################################################################

func (ctx *Context) Get(name string) string {
	return ctx.R.Form.Get(name)
}

// ################################################################

func httpAdapt(authreq int, f WebHandlerFunc) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {

		r.ParseForm()

		rw := &responseWriter{w: w, status: 200}
		ctx := &Context{W: rw, R: r}
		ctx.GetSession()

		defer func() {
			user := "[nouser]"
			if ctx.User != nil {
				user = ctx.User.Name
			}
			// NB: files in /static do not pass through here, and do not get logged
			dl.Verbose("ACCESS: %s %s %d %d %s",
				user, r.RemoteAddr, rw.status, rw.size, r.RequestURI)

		}()

		// check authorization
		switch authreq {
		case PUBLIC:
			break
		case PRIVATE, WRITE:
			if ctx.User == nil {
				http.Error(ctx.W, "Not Authorized", 403)
				return
			}
		}
		if authreq == WRITE {
			if ctx.Get("xtok") != ctx.XSRFToken {
				http.Error(ctx.W, "Not Authorized", 403)
				return
			}
		}

		// do it!
		f(ctx)
	}
}

// ################################################################

type responseWriter struct {
	w      http.ResponseWriter
	size   int64
	status int
}

func (w *responseWriter) Header() http.Header {
	return w.w.Header()
}

func (w *responseWriter) Write(b []byte) (int, error) {
	w.size += int64(len(b))
	return w.w.Write(b)
}

func (w *responseWriter) WriteHeader(s int) {
	w.status = s
	w.w.WriteHeader(s)
}
