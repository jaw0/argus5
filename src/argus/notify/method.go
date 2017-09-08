// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Sep-07 22:50 (EDT)
// Function: notification methods

package notify

import (
	"fmt"

	"argus/argus"
	"argus/configure"
)

type Method struct {
	builtin bool
	PATH    string
	Command string
	Send    string
	Qtime   int64                    `cfconv:"timespec"`
	Permit  [argus.CRITICAL + 1]bool `cfconv:"dotsev"`
}

var methods = map[string]*Method{
	"mail": &Method{
		builtin: true,
		Qtime:   300,
		Command: "sendmail -t -f {{.MAILFROM}}",
		Send:    "To: {{.MAILTO}}\nFrom: {{.MAILFROM}}\nSubject: {{.SUBJECT}}{{.ESCALATED}}\n\n{{.CONTENT}}\n",
	},
	"qpage": &Method{
		builtin: true,
		Command: "qpage",
		Send:    "{{.CONTENT}}{{.ESCALATED}}",
	},
}

func NewMethod(conf *configure.CF) error {

	m := &Method{}
	conf.InitFromConfig(&m, "method", "")

	if m.Command == "" {
		return fmt.Errorf("Invalid Notification Method - command not specified")
	}

	if methods[conf.Name] != nil && !methods[conf.Name].builtin {
		return fmt.Errorf("Duplicate Method '%s'", conf.Name)
	}

	conf.CheckTypos()
	methods[conf.Name] = m
	return nil
}
