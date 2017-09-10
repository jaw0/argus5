// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Sep-09 12:48 (EDT)
// Function: run program

package notify

import (
	"context"
	"io"
	"os/exec"
	"time"
)

const TIMEOUT = 15 * time.Second

func runCommand(command string, send string) {

	ctx, cancel := context.WithTimeout(context.Background(), TIMEOUT)
	defer cancel()

	dl.Debug("running '%s'", command)
	cmd := exec.CommandContext(ctx, "sh", "-c", command)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		dl.Problem("cannot run '%s': %v", command, err)
		return
	}

	go func() {
		io.WriteString(stdin, send)
		stdin.Close()
	}()

	out, err := cmd.CombinedOutput()
	if err != nil {
		dl.Problem("command failed '%s': %v", command, err)
	}
	if len(out) > 0 {
		dl.Debug("command output: %s", out)
	}
}
