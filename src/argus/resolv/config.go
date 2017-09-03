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

	"argus/config"
	"argus/diag"
)

const RESOLVCONF = "/etc/resolv.conf"

func (w *workstate) configure() {

	cf := config.Cf()

	w.search = []string{"."}

	for _, dom := range cf.DNS_search {
		w.addSearch(dom)
	}

	for _, ns := range cf.Nameserver {
		w.addServer(ns)
	}

	w.readResolvConf()

	if len(w.server) == 0 {
		w.addServer("127.0.0.1")
	}

}

func (w *workstate) readResolvConf() {

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
				w.addSearch(dom)
			}
		case "nameserver":
			for _, ns := range fields[1:] {
				w.addServer(ns)
			}
		}
	}
}

func (w *workstate) addSearch(dom string) {

	// surround with dots => .DOMAIN.
	if dom[0] != '.' {
		dom = "." + dom
	}
	if dom[len(dom)-1] != '.' {
		dom = dom + "."
	}

	w.search = append(w.search, dom)

}

func (w *workstate) addServer(ns string) {

	ua, err := net.ResolveUDPAddr("udp", ns+":53")
	if err != nil {
		diag.Fatal("invalid namserver: %s (%v)", ns, err)
	}
	w.server = append(w.server, ua)
}
