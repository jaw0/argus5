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

	"argus.domain/argus/api/client"
	"argus.domain/argus/argus"
)

const TIMEOUT = 15 * time.Second

type KVP struct {
	k string
	v string
}

func main() {

	var rawoutput bool
	var controlsock string
	flag.StringVar(&controlsock, "s", argus.ControlSocket, "control socket")
	flag.BoolVar(&rawoutput, "r", false, "raw output")
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

	var kvp []KVP
	maxlen := 0

	if rawoutput {
		for _, l := range resp.Lines {
			fmt.Printf("%s\n", l)
		}
		return
	}

	for _, l := range resp.Lines {
		kvs := strings.SplitN(l, ": ", 2)
		k := kvs[0]
		v := ""
		if len(k) > maxlen {
			maxlen = len(k)
		}
		if len(kvs) > 1 {
			v = argus.UrlDecode(strings.TrimSpace(kvs[1]))
		}
		kvp = append(kvp, KVP{k, v})
	}

	for _, kv := range kvp {
		fmt.Printf("%-*s  %s\n", maxlen+1, kv.k+":", kv.v)
	}
}
