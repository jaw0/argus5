// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Sep-15 20:33 (EDT)
// Function: darp client (slave -> master)

package darp

import (
	"crypto/tls"
	"fmt"
	"time"

	"argus/api/client"
	"argus/clock"
	"argus/resolv"
	"argus/sec"
)

const (
	TIMEOUT = 15 * time.Second
)

type Client struct {
	Name string
	Port int
	ip   *resolv.IP
	conn *client.Conn
}

func (d *DARP) StartClient() {

	c := d.NewClient(0)
	if c == nil {
		return
	}

	// fetch config from master?
	if d.Name == MyDarp.Fetch_Config {
		c.fetchConfig()
	}

	tock := time.NewTicker(10 * time.Second)
	for {
		select {
		case msg := <-d.ch:
			_, err := c.conn.Get(msg.f, msg.m, TIMEOUT)
			if err != nil {
				c.Close()
				c.Reconnect(0)
			}

		case <-tock.C:
			_, err := c.conn.Get("ping", nil, TIMEOUT)
			if err != nil {
				c.Close()
				c.Reconnect(0)
			}
			// RSN - ping response may have interesting data
		}

	}
}

func (d *DARP) NewClient(timeout int64) *Client {

	c := &Client{
		Name: d.Name,
		Port: d.Port,
		ip:   d.ip,
	}

	c.Reconnect(timeout)
	if c.conn == nil {
		return nil
	}

	return c
}

func (c *Client) Close() {
	if c.conn != nil {
		c.conn.Close()
	}
	c.conn = nil
}

func (c *Client) Reconnect(timeout int64) {

	t0 := clock.Unix()

	c.Close()

	for {
		dl.Debug("connecting to %s", c.Name)

		now := clock.Unix()
		if timeout != 0 && t0+timeout < now {
			return
		}

		// keep trying until we connect
		addr := c.ipAddr()
		name, _, _ := c.ip.Addr()

		conn, err := client.NewTLS(fmt.Sprintf("%s:%d", addr, c.Port), TIMEOUT, &tls.Config{
			Certificates: []tls.Certificate{*sec.Cert},
			RootCAs:      sec.Root,
			ServerName:   name, // cert must be configured with this ip addr
		})
		if err != nil {
			dl.Debug("connect failed to '%s': %v", c.Name, err)
			time.Sleep(5 * time.Second)
			continue
		}

		c.conn = conn
		break
	}

	dl.Verbose("darp connected to %s", c.Name)
}

func (c *Client) ipAddr() string {

	// wait for ip
	for {
		addr, fail := c.ip.AddrWB()
		if addr != "" {
			return addr
		}
		if fail {
			dl.Verbose("cannot resolve hostname '%s'", c.ip.Hostname())
			time.Sleep(60 * time.Second)
		}
		time.Sleep(time.Second)
	}
}
