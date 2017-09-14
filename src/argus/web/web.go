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
)

type Context struct {
	User string
	W    http.ResponseWriter
	R    *http.Request
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
		dir := cf.Htdir + "/htdocs"
		dl.Verbose("serving static on %s", dir)
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
func Add(authreq bool, path string, f WebHandlerFunc) {

	http.HandleFunc(path, httpAdapt(authreq, f))
}

func httpAdapt(authreq bool, f WebHandlerFunc) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {

		// RSN -
		// determine user
		// ...
		// check auth
		// ...

		ctx := &Context{W: w, R: r}
		f(ctx)

		// access_log? dl.Verbose?
		// ...
	}
}
