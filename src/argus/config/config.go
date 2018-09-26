// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Jun-19 12:10 (EDT)
// Function: load small initial config file

package config

import (
	"fmt"

	"argus/accfg"
	"argus/diag"
)

type Config struct {
	TLS_cert        string // our cert - .crt
	TLS_key         string // our private key - .key
	Errors_MailTo   string
	Errors_MailFrom string
	Syslog          string // syslog facility (eg. local4)
	Mon_maxrun      int
	Resolv_maxrun   int
	Ping_maxrun     int
	Port_http       int
	Port_https      int
	Port_test       int
	DARP_Name       string
	DARP_root       string
	DARP_key        string
	DARP_cert       string
	Datadir         string
	Htdir           string
	Monitor_config  string
	Control_Socket  string
	User            string
	Group           string
	DNS_server      []string
	DNS_search      []string
	DevMode         bool
	Agent_Mode      bool
	Auto_Reload     bool
	Agent_Port      int
	Debug           map[string]bool
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
		DNS_server: []string{},
		DNS_search: []string{},
		Debug:      make(map[string]bool),
	}

	err := accfg.Read(file, newcf)

	if err != nil {
		return fmt.Errorf("cannot read config '%s': %v", file, err)
	}

	diag.SetConfig(&diag.Config{
		Debug:    newcf.Debug,
		Facility: newcf.Syslog,
		Mailto:   newcf.Errors_MailTo,
		Mailfrom: newcf.Errors_MailFrom,
	})

	cf = newcf
	return nil
}
