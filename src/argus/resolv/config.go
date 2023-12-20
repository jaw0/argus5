// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Aug-30 22:33 (EDT)
// Function:

package resolv

import (
	"bufio"
	"net"
	"os"
	"strings"

	"argus.domain/argus/config"
	"github.com/jaw0/acgo/diag"
)

const RESOLVCONF = "/etc/resolv.conf"

var cfSearch []string
var cfServer []*net.UDPAddr

func resolvConfigure() {

	cf := config.Cf()

	cfSearch = []string{"."}

	for _, dom := range cf.DNS_search {
		addSearch(dom)
	}

	for _, ns := range cf.DNS_server {
		addServer(ns)
	}

	if len(cfServer) == 0 {
		readResolvConf()
	}

	if len(cfServer) == 0 {
		addServer("127.0.0.1")
	}

}

func readResolvConf() {

	fd, err := os.Open(RESOLVCONF)
	if err != nil {
		return
	}
	defer fd.Close()

	bfd := bufio.NewReader(fd)

	for {
		l, _, err := bfd.ReadLine()
		if err != nil {
			break
		}
		line := string(l)

		dl.Debug("resolv.conf> %s", line)

		// remove comments
		comment := strings.IndexByte(line, '#')
		if comment != -1 {
			line = line[:comment]
		}
		fields := strings.Fields(line)

		if len(fields) == 0 {
			continue
		}

		switch fields[0] {
		case "search", "domain":
			for _, dom := range fields[1:] {
				addSearch(dom)
			}
		case "nameserver":
			for _, ns := range fields[1:] {
				addServer(ns)
			}
		}
	}
}

func addSearch(dom string) {

	// surround with dots => .DOMAIN.
	if dom[0] != '.' {
		dom = "." + dom
	}
	if dom[len(dom)-1] != '.' {
		dom = dom + "."
	}

	cfSearch = append(cfSearch, dom)

}

func addServer(ns string) {

	ua, err := net.ResolveUDPAddr("udp", "["+ns+"]:53")
	if err != nil {
		diag.Fatal("invalid namserver: %s (%v)", ns, err)
	}
	cfServer = append(cfServer, ua)
}
