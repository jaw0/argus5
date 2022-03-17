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
	"sync"

	"argus.domain/argus/clock"
	"argus.domain/argus/configure"
	"argus.domain/argus/monitor/tcp"
	"argus.domain/argus/service"

	"github.com/jaw0/acgo/diag"
)

type cacheEntry struct {
	content string
	ctype   string
	creator *Url
	created int64
}

type Conf struct {
	URL         string
	Browser     string
	Referer     string
	HTTP_Accept string
	HTTP_Cache  int `cfconv:"timespec"`
}

type Url struct {
	tcp.TCP
	UCf  Conf
	File string
	Host string
}

const MAXREDIRECT = 16

var dl = diag.Logger("tcp")

var cacheLock sync.Mutex
var webCache = make(map[string]*cacheEntry)
var urlCount = make(map[string]int)

func init() {
	// register with service factory
	service.Register("TCP/URL", New)
}

func New(conf *configure.CF, s *service.Service) service.Monitor {

	u := &Url{}
	// set defaults
	u.UCf.Browser = "Argus"
	u.UCf.HTTP_Cache = 60
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

	conf.Param["hostname_!"] = &configure.CFV{Value: purl.Hostname(), Used: true}
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

	if d.Cf.SSL_ServerName == "" {
		d.Cf.SSL_ServerName = d.Host
	}

	// set tcp config
	err = d.TCP.Config(conf, s)
	if err != nil {
		return err
	}

	urlCount[d.UCf.URL]++

	// determine names
	uname := fmt.Sprintf("URL_%s:%d%s", d.Host, d.Cf.Port, d.File)
	s.SetNames(uname, "URL", "URL "+d.UCf.URL)

	return nil
}

func (d *Url) Start(s *service.Service) {

	s.Debug("url start")
	defer s.Done()

	if content, ctype, isCached := d.checkCached(s); isCached {
		d.S.Debug("using cached content")
		d.S.CheckValue(content, ctype)
		return
	}

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
		d.addCached(body, ctype)
		d.S.CheckValue(body, ctype)
	} else {
		d.S.CheckValue("", "")
	}

}

func (d *Url) checkCached(s *service.Service) (string, string, bool) {

	if d.UCf.HTTP_Cache <= 0 {
		return "", "", false
	}

	now := clock.Unix()

	cacheLock.Lock()
	defer cacheLock.Unlock()

	ce, ok := webCache[d.UCf.URL]
	if !ok || ce.creator == d || ce.created+int64(d.UCf.HTTP_Cache) <= now || ce.created+int64(s.Cf.Frequency) <= now {
		return "", "", false
	}

	return ce.content, ce.ctype, true
}

func (d *Url) addCached(content string, ctype string) {

	if d.UCf.HTTP_Cache <= 0 {
		return
	}

	ce := &cacheEntry{
		content: content,
		ctype:   ctype,
		created: clock.Unix(),
		creator: d,
	}

	cacheLock.Lock()
	defer cacheLock.Unlock()

	if urlCount[d.UCf.URL] < 2 {
		// content not needed elsewhere, don't cache
		return
	}

	webCache[d.UCf.URL] = ce
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
