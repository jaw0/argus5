// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Sep-07 22:50 (EDT)
// Function: notification methods

package notify

import (
	"fmt"
	"strings"

	"argus.domain/argus/argus"
	"argus.domain/argus/configure"
)

type Method struct {
	builtin bool
	Command string
	Send    string
	Qtime   int64                               `cfconv:"timespec"`
	Permit  [argus.CRITICAL + 1]*argus.Schedule `cfconv:"dotsev"`
}

var methods = map[string]*Method{
	"mail": &Method{
		builtin: true,
		Qtime:   300,
		Command: "sendmail -t -f {{.MAILFROM}}",
		Send:    "To: {{.MAILTO}}\nFrom: {{.MAILFROM}}\nSubject: {{.SUBJECT}}\n\n{{.CONTENT}}\n",
	},
	"qpage": &Method{
		builtin: true,
		Command: "qpage",
		Send:    "{{.CONTENT}}{{.ESCALATED}}",
	},
}

func NewMethod(conf *configure.CF) error {

	m := &Method{}
	conf.InitFromConfig(m, "method", "")

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

func methodForDst(dst string) (*Method, string) {

	f := strings.SplitN(dst, ":", 2)
	if len(f) == 1 {
		return methods["mail"], dst
	}
	return methods[f[0]], f[1]
}

// ################################################################

// called with package lock held
func (m *Method) transmit(dst string, addr string, notes []*N) {

	// build content

	subj := "Argus"
	esced := false
	joinWith := "\n"
	msgs := []string{}

	for _, n := range notes {
		n.lock.RLock()
		if n.p.OvStatus != argus.CLEAR {
			subj = "Argus - DOWN"
		}
		if strings.IndexByte(n.p.MessageFmted, '\n') != -1 {
			joinWith = "\n\n"
		}
		if n.p.Escalated {
			esced = true
		}
		msgs = append(msgs, n.p.MessageFmted)
		n.lock.RUnlock()
	}

	if esced {
		subj = subj + " (Escalated)"
	}

	content := strings.Join(msgs, joinWith)

	dat := map[string]interface{}{
		"MAILFROM": globalDefaults.Mail_From,
		"SENDER":   globalDefaults.Mail_From,
		"MAILTO":   addr,
		"ADDR":     addr,
		"RCPT":     addr,
		"SUBJECT":  subj,
	}

	// expand command + send
	notes[0].lock.RLock()
	command := notes[0].expand(m.Command, content, dat)
	send := notes[0].expand(m.Send, content, dat)
	notes[0].lock.RUnlock()

	dl.Debug("cmd: %s; send %s", command, send)

	runCommand(command, send)
}
