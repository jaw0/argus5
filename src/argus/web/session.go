// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Sep-14 12:46 (EDT)
// Function:

package web

import (
	"crypto/rand"
	"net/http"
	"sync"
	"time"

	"argus.domain/argus/argus"
	"argus.domain/argus/clock"
	"argus.domain/argus/config"
	"argus.domain/argus/users"
)

type Session struct {
	Name      string
	XSRFToken string
	Expires   int64
	Hush      int64
}

const (
	EXPIRES    = 30 * 24 * 3600
	COOKIENAME = "argus_ssid"
)

var lock sync.RWMutex
var sessions = make(map[string]*Session)

// RSN - store in sql, with lru

func load() {

	dl.Debug("loading sessions")
	cf := config.Cf()
	if cf.Datadir == "" {
		dl.Debug("datadir not configured. cannot load sessions")
		return
	}
	file := cf.Datadir + "/session"

	err := argus.Load(file, &sessions)

	if err != nil {
		dl.Verbose("cannot load users data: %v", err)
	}

	cleanup()
}

func save() {

	dl.Debug("saving sessions")
	cf := config.Cf()
	if cf.Datadir == "" {
		dl.Debug("datadir not configured. cannot save sessions")
		return
	}
	file := cf.Datadir + "/session"

	cleanup()
	err := argus.Save(file, sessions)
	if err != nil {
		dl.Problem("cannot save session data: %v", err)
	}
}

func cleanup() {

	now := clock.Unix()

	// expire old sessions
	for ck, sess := range sessions {
		if sess.Expires < now {
			delete(sessions, ck)
		}
	}
}

// ################################################################

func (ctx *Context) NewSession(name string) {

	ckdat := make([]byte, 16)
	rand.Read(ckdat)
	token := make([]byte, 8)
	rand.Read(token)

	ck := argus.Encode64Url(string(ckdat))

	sess := &Session{
		Name:      name,
		XSRFToken: argus.Encode64Url(string(token)),
		Expires:   clock.Unix() + EXPIRES,
	}

	dl.Debug("new session user %s: %s", name, ck)
	cookie := http.Cookie{Name: COOKIENAME, Value: ck, Expires: time.Now().Add(EXPIRES * time.Second)}
	http.SetCookie(ctx.W, &cookie)

	lock.Lock()
	defer lock.Unlock()

	if len(sessions) == 0 {
		load()
	}
	sessions[ck] = sess
	save()

	// RSN - darp?
}

func (ctx *Context) GetSession() {

	cookie, _ := ctx.R.Cookie(COOKIENAME)
	if cookie == nil {
		return
	}

	lock.RLock()
	defer lock.RUnlock()

	sess := sessions[cookie.Value]
	if sess == nil {
		return
	}

	if sess.Expires < clock.Unix() {
		return
	}

	user := users.Get(sess.Name)
	if user == nil {
		return
	}

	dl.Debug("found sess %s", sess.Name)
	ctx.User = user
	ctx.XSRFToken = sess.XSRFToken
	ctx.Hush = sess.Hush
}

func DelSession(ctx *Context) {

	cookie, _ := ctx.R.Cookie(COOKIENAME)
	if cookie == nil {
		return
	}

	lock.Lock()
	defer lock.Unlock()

	delete(sessions, cookie.Value)
	save()

}

func Hush(ctx *Context) {

	cookie, _ := ctx.R.Cookie(COOKIENAME)
	if cookie == nil {
		return
	}

	lock.Lock()
	defer lock.Unlock()

	sess := sessions[cookie.Value]
	if sess == nil {
		return
	}

	sess.Hush = clock.Nano()
	save()
}
