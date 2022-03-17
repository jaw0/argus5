// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Sep-14 10:49 (EDT)
// Function: users

package users

import (
	"sync"

	"golang.org/x/crypto/bcrypt"

	"argus.domain/argus/api"
	"argus.domain/argus/argus"
	"argus.domain/argus/config"
	"github.com/jaw0/acgo/diag"
)

type User struct {
	Name    string
	EPasswd string
	Home    string
	Groups  string
}

const BCRYPTCOST = 10

var lock sync.Mutex
var allusers = make(map[string]*User)
var dl = diag.Logger("users")

func init() {
	api.Add(true, "setuser", apiSetUser)
	api.Add(true, "getuser", apiGetUser)
	api.Add(true, "deluser", apiDelUser)
	api.Add(true, "listusers", apiListUsers)
}

func Get(name string) *User {

	lock.Lock()
	defer lock.Unlock()

	if len(allusers) == 0 {
		load()
	}

	return allusers[name]
}

func (user *User) Update() {

	lock.Lock()
	defer lock.Unlock()

	// copy+replace, so we don't need a user.lock
	nuser := User{}
	nuser = *user

	allusers[user.Name] = &nuser
	save()
}

func (user *User) remove() {
	lock.Lock()
	defer lock.Unlock()

	delete(allusers, user.Name)
	save()
}

func List() []string {

	lock.Lock()
	defer lock.Unlock()

	if len(allusers) == 0 {
		load()
	}

	users := make([]string, 0, len(allusers))

	for k := range allusers {
		users = append(users, k)
	}

	return users
}

func (user *User) CheckPasswd(pass string) bool {

	e := bcrypt.CompareHashAndPassword([]byte(user.EPasswd), []byte(pass))

	if e == nil {
		return true
	}
	return false
}

func CheckUserPasswd(name string, pass string) *User {

	u := Get(name)
	if u == nil {
		return nil
	}
	if u.CheckPasswd(pass) {
		return u
	}
	return nil
}

// ################################################################

func load() {

	dl.Debug("loading users")
	cf := config.Cf()
	if cf.Datadir == "" {
		dl.Debug("datadir not configured. cannot load users")
		return
	}
	file := cf.Datadir + "/users"

	err := argus.Load(file, &allusers)

	if err != nil {
		dl.Problem("cannot load users data: %v", err)
	}
}

func save() {

	dl.Debug("saving users")
	cf := config.Cf()
	if cf.Datadir == "" {
		dl.Debug("datadir not configured. cannot save users")
		return
	}
	file := cf.Datadir + "/users"

	err := argus.Save(file, allusers)
	if err != nil {
		dl.Problem("cannot save users data: %v", err)
	}
}

// ################################################################

// argusctl setuser name=NAME [passwd=PLAINTEXT] [home=HOME] [groups=GROUPS]

func apiSetUser(ctx *api.Context) {

	name := ctx.Args["user"]
	if name == "" {
		ctx.SendResponseFinal(500, "must specify user")
		return
	}

	user := Get(name)
	dl.Debug("got %#v", user)

	if user == nil {
		user = &User{Name: name, Home: "Top", Groups: "user"}
	}

	if p := ctx.Args["pass"]; p != "" {
		ep, _ := bcrypt.GenerateFromPassword([]byte(p), BCRYPTCOST)
		user.EPasswd = string(ep)
	}

	if h := ctx.Args["home"]; h != "" {
		user.Home = h
	}

	if g := ctx.Args["groups"]; g != "" {
		user.Groups = g
	}

	user.Update()
	ctx.SendOKFinal()
}

func apiGetUser(ctx *api.Context) {

	name := ctx.Args["user"]
	if name == "" {
		ctx.SendResponseFinal(500, "must specify user")
		return
	}

	user := Get(name)

	if user == nil {
		ctx.Send404()
		return
	}

	ctx.SendOK()
	argus.Dump(ctx, "", user)
	ctx.SendFinal()
}

func apiDelUser(ctx *api.Context) {

	name := ctx.Args["user"]
	if name == "" {
		ctx.SendResponseFinal(500, "must specify user")
		return
	}

	user := Get(name)
	dl.Debug("got %#v", user)

	if user != nil {
		user.remove()
	}

	ctx.SendOKFinal()
}

func apiListUsers(ctx *api.Context) {

	users := List()

	ctx.SendOK()
	for _, u := range users {
		ctx.Send(u)
		ctx.Send("\n")
	}

	ctx.SendFinal()
}
