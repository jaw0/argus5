// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Sep-12 20:08 (EDT)
// Function: control argus

package main

import (
	"flag"
	"fmt"
	"strings"
	"time"

	"argus/api/client"
)

const TIMEOUT = 15 * time.Second

func main() {

	var controlsock string
	flag.StringVar(&controlsock, "c", "/tmp/argus.ctl", "control socket")
	flag.Parse()

	c, err := client.New("unix", controlsock, TIMEOUT)
	if err != nil {
		fmt.Printf("cannot connect to argus: %v", err)
		return
	}

	method := flag.Arg(0)
	args := make(map[string]string)

	for i := 1; i < flag.NArg(); i++ {
		kvp := strings.SplitN(flag.Arg(i), "=", 2)
		if len(kvp) == 2 {
			args[kvp[0]] = kvp[1]
		} else {
			args[kvp[0]] = ""
		}
	}

	resp, err := c.Get(method, args, TIMEOUT)
	if err != nil {
		fmt.Printf("error: %v", err)
		return
	}

	fmt.Printf("%d %s\n", resp.Code, resp.Msg)

	for _, l := range resp.Lines {
		fmt.Printf("%s\n", l)
	}
}
