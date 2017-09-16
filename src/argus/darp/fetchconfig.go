// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Sep-16 12:30 (EDT)
// Function: fetch config from remote

package darp

func (c *Client) fetchConfig() {

	dl.Debug("fetching config")
	for {
		err := c.tryFetchConfig()

		if err == nil {
			return
		}

		dl.Verbose("error fetching config: %v", err)
		c.Reconnect(0)
	}
}

func (c *Client) tryFetchConfig() error {

	// get list of our jawns
	resp, err := c.conn.Get("darp_list", map[string]string{"tag": MyId}, TIMEOUT)
	if err != nil {
		return err
	}

	// get the configs
	for _, obj := range resp.Lines {
		dl.Debug("get config %s", obj)

		rcf, err := c.conn.Get("getconfig", map[string]string{"obj": obj}, TIMEOUT)
		if err != nil {
			return err
		}

		// create

		objMaker.Make(rcf.Map())
	}

	return nil
}
