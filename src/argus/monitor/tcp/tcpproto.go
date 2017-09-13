// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Sep-09 22:25 (EDT)
// Function: tcp protocols

package tcp

import (
	"encoding/hex"
	"strings"
)

type tcpProtoConf struct {
	Port    int
	Send    string
	Expect  string
	ReadHow string
	SSL     bool
}

var tcpProtoTab = map[string]*tcpProtoConf{

	// not RFC compliant, but quiets sendmails logs
	"SMTP":       {Port: 25, Send: "MAIL\r\n", Expect: "^220", ReadHow: "banner"},
	"FTP":        {Port: 21, Expect: "^220", ReadHow: "banner"},
	"POP":        {Port: 110, Expect: "^\\+OK", ReadHow: "banner"},
	"NNTP":       {Port: 119, Expect: "^220", ReadHow: "banner"},
	"Telnet":     {Port: 23},
	"Gopher":     {Port: 70, Send: "\r\n", Expect: "\\.\r\n"},
	"HTTP":       {Port: 80, Expect: "HTTP", ReadHow: "toeof"},
	"HTTPS":      {Port: 443, Expect: "HTTP", ReadHow: "toeof", SSL: true},
	"SSH":        {Port: 22, Send: "SSH-1.99-argus\r\n", Expect: "^SSH", ReadHow: "banner"},
	"IMAP":       {Port: 143, Expect: "^\\* OK", ReadHow: "banner"},
	"Whois":      {Port: 43, Send: "\r\n"},
	"Rwhois":     {Port: 4321, Expect: "^%rwhois", ReadHow: "banner"},
	"LPD":        {Port: 515, Send: "\x03lp\n", ReadHow: "toeof"},
	"POPS":       {Port: 995, Expect: "^\\+OK", ReadHow: "banner", SSL: true},
	"IMAPS":      {Port: 993, Expect: "^\\* OK", ReadHow: "banner", SSL: true},
	"SMTPS":      {Port: 465, Send: "MAIL\r\n", Expect: "^220", ReadHow: "banner"},
	"NNTPS":      {Port: 563, Expect: "^220", ReadHow: "banner"},
	"Argus":      {Expect: "running", ReadHow: "banner"},
	"SlimServer": {Port: 9090, Send: "version ?\r\n", Expect: "version", ReadHow: "banner"},
	"Asterisk":   {Port: 9038, ReadHow: "toeof"}, // see also: monitor/asterisk/
	"Freeswitch": {Port: 8021, ReadHow: "toeof"}, // see also: monitor/freeswitch/
	"NFS": {Port: 2049, ReadHow: "once",
		Send: hxd("800000280000304E0000000000000002000186A3000000020000000000000000000000000000000000000000")},
	"NFSv3": {Port: 2049, ReadHow: "once",
		Send: hxd("800000280000304E0000000000000003000186A3000000020000000000000000000000000000000000000000")},

	// NB - http and family are handled elsewhere
}

func hxd(s string) string {
	x, _ := hex.DecodeString(strings.Replace(s, " ", "", -1))
	return string(x)
}
