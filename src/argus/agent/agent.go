// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Oct-16 20:19 (EDT)
// Function: agent to monitor remote hosts (server side)

package agent

import (
	"context"
	"expvar"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"argus.domain/argus/api"
	"argus.domain/argus/service"
	"github.com/jaw0/acgo/diag"
)

var dl = diag.Logger("agent")

func init() {

	api.Add(true, "agent", apiAgent)
	api.Add(true, "getos", apiOS)
}

func apiOS(ctx *api.Context) {

	ctx.SendOK()
	ctx.SendKVP("os", runtime.GOOS)
	ctx.SendFinal()
}

// args: command, timeout, pluck, jpath
func apiAgent(ctx *api.Context) {

	self := ctx.Args["self"]
	command := ctx.Args["command"]
	os := ctx.Args["os"]

	if os != "" && os != runtime.GOOS {
		apiOS(ctx)
		return
	}

	if self != "" {
		agentSelf(self, ctx)
		return
	}

	if command != "" {
		agentCommand(command, ctx)
		return
	}

	ctx.Send404()
}

func agentSelf(self string, ctx *api.Context) {

	cv := expvar.Get(self)

	if cv == nil {
		ctx.Send404()
		return
	}

	ctx.SendOK()
	ctx.SendKVP("result", cv.String())
	ctx.SendKVP("os", runtime.GOOS)
	ctx.SendFinal()
}

func agentCommand(command string, ctx *api.Context) {

	timeout, _ := strconv.Atoi(ctx.Args["timeout"])
	pluck := ctx.Args["pluck"]
	jpath := ctx.Args["jpath"]
	column := ctx.Args["column"]

	res, fail := runProg(command, timeout)

	if column != "" {
		c, _ := strconv.Atoi(column)
		f := strings.Fields(res)

		if len(f) > c {
			res = f[c]
		} else {
			res = ""
		}
	}

	if pluck != "" {
		res = service.Pluck(pluck, res)
	}
	if jpath != "" {
		res, _ = service.JsonPath(jpath, res)
	}

	dl.Debug("result: %s", res)

	ctx.SendOK()
	ctx.SendKVP("result", res)
	ctx.SendKVP("os", runtime.GOOS)

	if fail {
		ctx.SendKVP("fail", "1")
	}

	ctx.SendFinal()
}

func runProg(command string, timeout int) (string, bool) {

	if timeout < 1 {
		timeout = 1
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel()

	dl.Debug("running '%s' [to=%d]", command, timeout)
	cmd := exec.CommandContext(ctx, "sh", "-c", command)

	// if we are running as root, attempt to switch to a nonpriveleged uid
	if os.Geteuid() == 0 {
		cmd.SysProcAttr = &syscall.SysProcAttr{
			Credential: &syscall.Credential{
				Uid: 65535,
				Gid: 65535,
			},
		}
	}

	out, err := cmd.CombinedOutput()
	if err != nil {
		dl.Debug("command failed: %v", err)
		return "", true
	}
	if len(out) > 0 {
		dl.Debug("command output: %s", out)
	}

	return string(out), false
}
