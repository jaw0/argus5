// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Aug-28 21:57 (EDT)
// Function: diagnostic logging

package diag

import (
	"fmt"
	"os"
	"sync"
)

type Config struct {
	Debug map[string]bool
}

type Diag struct {
	section string
}

var debugall = false
var lock sync.RWMutex
var config = &Config{}

func (d *Diag) Debug(format string, args ...interface{}) {

	cf := getConfig()

	if !debugall && !cf.Debug[d.section] && !cf.Debug["all"] {
		return
	}

	diag(format, args)
}

func Verbose(format string, args ...interface{}) {
	diag(format, args)
}

func Log(format string, args ...interface{}) {
	diag(format, args)
}

func Problem(format string, args ...interface{}) {
	diag(format, args)
}

func Fatal(format string, args ...interface{}) {
	diag(format, args)
	os.Exit(-1)
}

// ################################################################

func Logger(sect string) *Diag {
	return &Diag{section: sect}
}

func (d *Diag) Logger(sect string) *Diag {
	return &Diag{section: sect}
}

func Init() {

}

func SetConfig(cf *Config) {
	lock.Lock()
	defer lock.Unlock()
	config = cf
}

func getConfig() *Config {
	lock.RLock()
	defer lock.RUnlock()
	return config
}

func diag(format string, args []interface{}) {

	fmt.Printf(format+"\n", args...)
}
