// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Sep-09 17:00 (EDT)
// Function: monitor web

package url

import (
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"argus/configure"
	"argus/diag"
	"argus/monitor/tcp"
	"argus/service"
)

type Conf struct {
	URL         string
	Browser     string
	Referer     string
	HTTP_Accept string
}

type Url struct {
	tcp.TCP
	UCf  Conf
	File string
	Host string
}

const MAXREDIRECT = 16

var dl = diag.Logger("tcp")

func init() {
	// register with service factory
	service.Register("TCP/URL", New)
}

func New(conf *configure.CF, s *service.Service) service.Monitor {

	u := &Url{}
	// set defaults
	u.UCf.Browser = "Argus"
	u.TCP.InitNew(conf, s)
	u.TCP.Cf.Port = 80
	u.TCP.Cf.ReadHow = "toeof"
	return u
}

func (d *Url) Config(conf *configure.CF, s *service.Service) error {

	conf.InitFromConfig(&d.UCf, "URL", "")

	if d.UCf.URL == "" {
		return errors.New("URL not specified")
	}

	// parse url, set defaults
	purl, err := url.Parse(d.UCf.URL)
	if err != nil {
		return fmt.Errorf("cannot parse url '%s': %v", d.UCf.URL, err)
	}

	if purl.Scheme == "https" {
		d.Cf.SSL = true
	}

	d.Cf.Hostname = purl.Hostname()
	d.Host = purl.Hostname()
	d.File = purl.RequestURI()

	if d.File == "" {
		d.File = "/"
	}

	if purl.Port() == "" {
		if purl.Scheme == "https" {
			d.Cf.Port = 443
		} else {
			d.Cf.Port = 80
		}

	} else {
		d.Cf.Port, _ = strconv.Atoi(purl.Port())
	}

	// set tcp config
	d.TCP.Config(conf, s)

	// determine names
	uname := fmt.Sprintf("URL_%s:%d%s", d.Host, d.Cf.Port, d.File)
	s.SetNames(uname, "URL", "URL "+d.UCf.URL)

	return nil
}

func (d *Url) Start(s *service.Service) {

	s.Debug("url start")
	defer s.Done()

	file := d.File
	nredir := 0
	res := ""

	for {
		// request, redirect
		resp, fail := d.makeRequest(file)
		if fail {
			return
		}

		head := headers(resp)
		file = redirect(head)
		if file == "" {
			res = resp
			break
		}

		d.S.Debug("redirect to %s", file)
		nredir++
		if nredir >= MAXREDIRECT {
			d.S.Fail("redirect loop")
			return
		}
	}

	// check result
	sects := strings.SplitN(res, "\r\n\r\n", 2)
	head := sects[0]
	heads := headers(head)
	ctype := contentType(getHeader("Content-Type", heads))

	// QQQ - check for 200?
	// NB - argus3 checked the entire response, not just the content
	// but we want to do some jsontastic things...

	if len(sects) > 1 {
		body := sects[1]
		d.S.CheckValue(body, ctype)
	} else {
		d.S.CheckValue("", "")
	}

}

func (d *Url) makeRequest(file string) (string, bool) {

	d.ToSend = d.httpSend(file)
	return d.MakeRequest()
}

func (d *Url) httpSend(file string) string {

	send := "GET " + d.File + " HTTP/1.1\r\n" +
		"Host: " + d.Host + "\r\n" +
		"Connection: Close\r\n"

	if d.UCf.Browser != "" {
		send += "User-Agent: " + d.UCf.Browser + "\r\n"
	}
	if d.UCf.Referer != "" {
		send += "Referer: " + d.UCf.Referer + "\r\n"
	}
	if d.UCf.HTTP_Accept != "" {
		send += "Accept: " + d.UCf.HTTP_Accept + "\r\n"
	}
	// RSN - X-Argus-*
	send += "\r\n"

	return send
}

func headers(resp string) []string {
	delim := strings.Index(resp, "\r\n\r\n")
	if delim == -1 {
		delim = len(resp)
	}
	headers := resp[:delim]
	return strings.Split(headers, "\r\n")
}

func getHeader(h string, hs []string) string {

	for _, line := range hs {
		if strings.HasPrefix(line, h) {
			return strings.Trim(line[len(h)+1:], " \t")
		}
	}
	return ""
}

// is this a redirect? to where?
// NB - we only redirect to locations on the same host
func redirect(headers []string) string {

	loc := getHeader("Location", headers)
	if loc == "" {
		return ""
	}
	purl, err := url.Parse(loc)
	if err == nil {
		return purl.RequestURI()
	}

	return ""
}

func contentType(ct string) string {

	if strings.Index(ct, "json") != -1 || strings.Index(ct, "javascript") != -1 {
		return "json"
	}
	return ct
}

func (u *Url) WebJson(md map[string]interface{}) {
	md["URL"] = u.UCf.URL
}
