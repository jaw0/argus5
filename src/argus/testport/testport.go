// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Oct-06 13:59 (EDT)
// Function: remotely testing argus

package testport

import (
	"fmt"
	"net"
	"time"

	"github.com/jaw0/acdiag"
)

func Start(port int) {

	if port == 0 {
		return
	}

	diag.Verbose("testport on tcp/%d", port)
	sock, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		diag.Problem("cannot start test server: %v", err)
		return
	}

	go server(sock)
}

func server(sock net.Listener) {

	for {
		conn, err := sock.Accept()
		if err != nil {
			return
		}
		go sendOK(conn)
	}

}

func sendOK(conn net.Conn) {

	conn.SetDeadline(time.Now().Add(5 * time.Second))
	conn.Write([]byte("Argus running\n"))
	conn.Close()
}

func Stop() {

}
