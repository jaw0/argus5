// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Sep-12 22:10 (EDT)
// Function: api server

package api

import (
	"bufio"
	"crypto/rand"
	"fmt"
	"net"
	"os"
	"strings"

	"argus/argus"
	"argus/config"
)

const (
	CONTROLSOCKET = "/var/run/argus.ctl"
	PROTOCOL      = "ARGUS/5.0"
	NONCELEN      = 64
)

type Serverer interface {
	Auth(string, string, string) bool
	Connected(string)
	Disco(string)
}

type listenSet struct {
	lsock net.Listener
	dom   string
}

var apiListener []*listenSet

func Init() {

	cf := config.Cf()

	ctl := CONTROLSOCKET
	if cf.Control_Socket != "" {
		ctl = cf.Control_Socket
	}

	os.Remove(ctl)
	ServerNew(nil, "api", "unix", ctl)
}

func Stop() {

	for _, l := range apiListener {
		l.lsock.Close()
	}
}

func ServerNew(ob Serverer, who string, dom string, addr string) {

	l, err := net.Listen(dom, addr)

	if err != nil {
		dl.Problem("cannot open socket: %v", err)
		return
	}

	dl.Verbose("%s listening on %s:%s", who, dom, addr)

	apiListener = append(apiListener, &listenSet{l, dom})
	go serverAccept(ob, l, dom)
}

func serverAccept(ob Serverer, l net.Listener, dom string) {

	for {
		c, err := l.Accept()
		if err != nil {
			return
		}
		dl.Verbose("connection from %s/%s", dom, c.RemoteAddr())

		go apiRun(ob, c, dom)
	}
}

func apiRun(ob Serverer, c net.Conn, dom string) {

	bfd := bufio.NewReader(c)
	ctx := Context{doer: ob, Conn: c, bfd: bfd}

	defer func() {
		c.Close()
		if ob != nil {
			ob.Disco(ctx.User)
		}
	}()

	// unix socket connections are trusted,
	// network connections need to authenticate
	if dom == "unix" {
		ctx.Authed = true
	} else {
		nonce := make([]byte, NONCELEN)
		_, err := rand.Read(nonce)

		if err != nil {
			dl.Verbose("cannot read random: %v", err)
			return
		}

		ctx.Nonce = argus.Encode64Url(string(nonce))
	}

	for {
		ok := ctx.readRequest()
		if !ok {
			return
		}
		if ob != nil && ctx.User != "" {
			ob.Connected(ctx.User)
		}

		ok = ctx.dispatch()
		if !ok {
			return
		}
	}

}

// Grammar, which knows how to control even kings.
//        -- Les Femmes savantes. Act ii. Sc. 6.
//           Jean Baptiste Poquelin Moliere.
//
// protocol is looks roughly like http
//
// Protocol:
//    connect
//    client - send request
//    server - send response
//    repeat...
//
// request:
//    request type and version...: GET REQUEST Argus/2.0
//    param: value\n
//    param: value\n
//    ...
//    blank line\n
//
//    value is url_encoded
//    currently request is only GET
//
//    example:
//	GET /echo ARGUS/2.0
//	foobar: 123
//	<blank line>
//
// response:
//    word number text\n
//    optional data\n
//    ...
//    blank line\n
//
// status numbers:
// 2?? - OK
// anything else - error
//
//    example:
//	ARGUS/2.0 200 OK
//	foobar: 123
//	<blank line>

func (ctx *Context) readRequest() bool {

	// read request line
	reqline, _, err := ctx.bfd.ReadLine()
	if err != nil {
		dl.Debug("read error: %v", err)
		return false
	}
	// parse request: "GET /func ARGUS/5.0
	flds := strings.Fields(string(reqline))

	if len(flds) != 3 {
		return false
	}

	if flds[0] != "GET" || flds[2] != PROTOCOL {
		return false
	}

	ctx.Method = argus.UrlDecode(flds[1])
	dl.Debug("request: %s", ctx.Method)

	// read header lines
	ctx.Args = make(map[string]string)
	for {
		line, _, err := ctx.bfd.ReadLine()
		if err != nil {
			dl.Debug("read error: %v", err)
			return false
		}
		if len(line) == 0 {
			break
		}
		fvp := strings.SplitN(string(line), ": ", 2)

		if len(fvp) == 2 {
			ctx.Args[strings.TrimSpace(fvp[0])] = argus.UrlDecode(strings.TrimSpace(fvp[1]))
		} else {
			ctx.Args[strings.TrimSpace(fvp[0])] = ""
		}

	}

	return true
}

// ################################################################

func (ctx *Context) SendOK() {
	ctx.SendResponse(200, "OK")
}
func (ctx *Context) SendOKFinal() {
	ctx.SendResponseFinal(200, "OK")
}
func (ctx *Context) Send404() {
	ctx.SendResponseFinal(404, "Not Found")
}
func (ctx *Context) SendResponseFinal(code int, msg string) {
	ctx.SendResponse(code, msg)
	ctx.SendFinal()
}
func (ctx *Context) SendResponse(code int, msg string) {
	fmt.Fprintf(ctx.Conn, "%s %d %s\n", PROTOCOL, code, msg)
}
func (ctx *Context) SendFinal() {
	ctx.Conn.Write([]byte("\n"))
}
func (ctx *Context) SendKVP(key string, val string) {
	fmt.Fprintf(ctx.Conn, "%s: %s\n", key, argus.UrlEncode(val))
}
func (ctx *Context) Send(txt string) {
	ctx.Conn.Write([]byte(txt))
}

// ################################################################

func init() {

	Add(false, "exit", apiFuncExit)
	Add(false, "auth", apiFuncAuth)
	Add(true, "ping", apiFuncPing)
}

func apiFuncExit(ctx *Context) {
	ctx.SendOKFinal()
	ctx.Conn.Close()
}

func apiFuncPing(ctx *Context) {
	// RSN - send back interesting data...
	// started_time, user sessions, overrides, ...
	ctx.SendOKFinal()
}

func apiFuncAuth(ctx *Context) {

	name := ctx.Args["name"]
	digest := ctx.Args["digest"]

	if name == "" || digest == "" {
		ctx.SendResponse(403, "Authentication Required")
		ctx.SendKVP("nonce", string(ctx.Nonce))
		ctx.SendFinal()
		return
	}

	if ctx.doer != nil {
		ok := ctx.doer.Auth(name, ctx.Nonce, digest)
		if ok {
			dl.Verbose("authentication ok for %s from %s", name, ctx.Conn.RemoteAddr())
			ctx.User = name
			ctx.Authed = true
			ctx.doer.Connected(name)
			ctx.SendOKFinal()
		} else {
			ctx.SendResponseFinal(403, "Authentication Failed")
			dl.Verbose("authentication failed from %s", ctx.Conn.RemoteAddr())
		}
	}
}
