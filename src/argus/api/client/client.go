// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Sep-12 22:40 (EDT)
// Function: argus client

package client

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"argus/argus"
)

type Conn struct {
	C   net.Conn
	bfd *bufio.Reader
}

type Response struct {
	Code  int
	Msg   string
	Lines []string
	param map[string]string
}

const (
	PROTOCOL = "ARGUS/5.0"
)

func New(dom string, addr string, timeout time.Duration) (*Conn, error) {

	c, err := net.DialTimeout(dom, addr, timeout)
	if err != nil {
		return nil, err
	}

	bfd := bufio.NewReader(c)

	return &Conn{C: c, bfd: bfd}, nil

}

func NewTLS(addr string, timeout time.Duration, tlscf *tls.Config) (*Conn, error) {

	c, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return nil, err
	}

	tc := tls.Client(c, tlscf)

	bfd := bufio.NewReader(tc)

	return &Conn{C: tc, bfd: bfd}, nil

}
func (c *Conn) Close() {
	c.C.Close()

}

func (c *Conn) GetMap(method string, args map[string]string, timeout time.Duration) (*Response, error) {

	resp, err := c.Get(method, args, timeout)
	if err != nil {
		return resp, err
	}
	resp.Map()
	return resp, err
}

func (c *Conn) Get(method string, args map[string]string, timeout time.Duration) (*Response, error) {

	// send request line
	fmt.Fprintf(c.C, "GET %s %s\n", method, PROTOCOL)

	// send header lines
	for k, v := range args {
		fmt.Fprintf(c.C, "%s: %s\n", k, argus.UrlEncode(v))
	}
	fmt.Fprintf(c.C, "\n")

	// get response line
	respline, _, err := c.bfd.ReadLine()
	if err != nil {
		return nil, err
	}
	// parse response "argus/5.0 200 ok"
	flds := strings.SplitN(string(respline), " ", 3)
	if len(flds) != 3 {
		return nil, fmt.Errorf("protocol botched")
	}

	code, _ := strconv.Atoi(flds[1])
	resp := &Response{
		Code: code,
		Msg:  flds[2],
	}

	// get content lines
	for {
		line, _, _ := c.bfd.ReadLine()
		if len(line) == 0 {
			break
		}
		resp.Lines = append(resp.Lines, string(line))
	}

	return resp, nil
}

func (resp *Response) Map() map[string]string {

	if resp.param != nil {
		return resp.param
	}

	resp.param = make(map[string]string)

	for _, line := range resp.Lines {
		kvp := strings.SplitN(line, ": ", 2)
		if len(kvp) == 2 {
			resp.param[kvp[0]] = argus.UrlDecode(strings.TrimSpace(kvp[1]))
		} else {
			resp.param[kvp[0]] = ""
		}
	}

	return resp.param
}
