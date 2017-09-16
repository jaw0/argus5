// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Sep-15 20:33 (EDT)
// Function: darp client (slave -> master)

package darp

import (
	"fmt"
	"time"

	"argus/api/client"
	"argus/clock"
	"argus/resolv"
)

const (
	TIMEOUT = 15 * time.Second
)

type Client struct {
	Name string
	Host string
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
		Host: d.Hostname,
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
		now := clock.Unix()
		if timeout != 0 && t0+timeout < now {
			return
		}

		// keep trying until we connect
		addr := c.ipAddr()
		conn, err := client.New("tcp", fmt.Sprintf("%s:%d", addr, c.Port), TIMEOUT)
		if err != nil {
			dl.Debug("connect failed to '%s': %v", c.Name, err)
			time.Sleep(5 * time.Second)
			continue
		}

		c.conn = conn
		ok := c.auth()

		if ok {
			break
		}

		time.Sleep(10 * time.Second)
		conn.Close()
		c.conn = nil
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
			dl.Verbose("cannot resolve hostname '%s'", c.Host)
			time.Sleep(60 * time.Second)
		}
		time.Sleep(time.Second)
	}
}

func (c *Client) auth() bool {

	resp, err := c.conn.Get("auth", nil, TIMEOUT)
	if err != nil {
		dl.Debug("error: %v", err)
		return false
	}

	res := resp.Map()
	nonce := res["nonce"]
	dl.Debug("recvd nonce %s", nonce)

	digest := authDigest(MyDarp.Pass, nonce)

	resp, err = c.conn.Get("auth", map[string]string{
		"name":   MyDarp.Name,
		"digest": digest,
	}, TIMEOUT)

	if err != nil {
		dl.Debug("error: %v", err)
		return false
	}

	if resp.Code != 200 {
		dl.Verbose("authentication with '%s' failed '%d %s'", c.Name, resp.Code, resp.Msg)
		return false
	}
	return true
}
