// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Jun-19 12:10 (EDT)
// Function: load small initial config file

package config

import (
	"fmt"

	"accfg"

	"argus/diag"
)

type Config struct {
	Error_mailto   string
	Error_mailfrom string
	TLS_cert       string // our cert - .crt
	TLS_key        string // our private key - .key
	TLS_root       string // root cert - .crt
	Mon_maxrun     int
	Resolv_maxrun  int
	Port_http      int
	Port_https     int
	Port_darp      int
	Port_test      int
	Nameserver     []string
	DNS_search     []string
	Debug          map[string]bool

	// RSN - various files + directories ...
}

var cf *Config = &Config{}

func Load(file string) {

	err := read_config(file)
	if err != nil {
		diag.Fatal("%s", err)
	}
}

func Cf() *Config {
	return cf
}

func read_config(file string) error {

	newcf := &Config{
		Nameserver: []string{},
		DNS_search: []string{},
		Debug:      make(map[string]bool),
	}

	err := accfg.Read(file, newcf)

	if err != nil {
		return fmt.Errorf("cannot read config '%s': %v", file, err)
	}

	diag.SetConfig(&diag.Config{
		Debug: newcf.Debug,
	})

	cf = newcf
	return nil
}
