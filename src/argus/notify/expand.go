// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Sep-08 19:11 (EDT)
// Function: expand templates

package notify

import (
	"bytes"
	"fmt"
	"text/template"
	"time"

	"argus/argus"
)

func (n *N) expand(templ string, content string, param map[string]interface{}) string {

	status := "DOWN"
	if n.p.OvStatus == argus.CLEAR {
		status = "UP"
	}

	tc := time.Unix(n.p.Created, 0)
	created := tc.Format("2/Jan 15:04")

	dat := map[string]interface{}{
		"IDNO":         fmt.Sprintf("%d", n.p.IdNo),
		"CREATED":      created,
		"TIME":         tc,
		"CONTENT":      content,
		"BODY":         content,
		"MESSAGE":      n.p.Message,
		"MESSAGEFMTED": n.p.MessageFmted,
		"OBJECT":       n.p.Unique,
		"UNAME":        n.p.ShortName,
		"FRIENDLY":     n.p.FriendlyName,
		"REASON":       n.p.Reason,
		"RESULT":       n.p.Result,
		"STATUS":       status,
		"SEVERITY":     n.p.OvStatus.String(),
		// RSN - objecturl, notifyurl, object-info/details ?
	}

	// copy in params
	for key, val := range param {
		dat[key] = val
	}

	t := template.New("x")
	_, err := t.Parse(templ)
	if err != nil {
		dl.Problem("cannot parse template '%s': %v", templ, err)
		return templ
	}
	var buf []byte
	bio := bytes.NewBuffer(buf)
	t.Execute(bio, dat)

	dl.Debug("templ %s => %s", templ, bio.String())
	return bio.String()
}
