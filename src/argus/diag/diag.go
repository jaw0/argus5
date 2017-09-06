// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Jun-16 14:01 (EDT)
// Function: AC style diagnostics+logging

/*
in config file:
    debug section

at top of file:
    var dl = diag.Logger("section")

in code:
    dl.Debug(...)
    dl.Verbose(...)
    ...
*/

package diag

import (
	"flag"
	"fmt"
	"log/syslog"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
)

// defaults
const (
	stack_max = 32768
)

var hostname = "?"
var progname = "?"
var debugall = false
var usestderr = true

var lock sync.RWMutex
var config = &Config{}
var defaultDiag = &Diag{"default"}

type Diag struct {
	section string
}

type Config struct {
	Mailto   string
	Mailfrom string
	Debug    map[string]bool
}

type logconf struct {
	logprio    syslog.Priority
	to_stderr  bool
	to_email   bool
	with_info  bool
	with_trace bool
}

func init() {
	flag.BoolVar(&debugall, "d", false, "enable all debugging")
}

func (d *Diag) Verbose(format string, args ...interface{}) {
	diag(logconf{
		logprio:   syslog.LOG_INFO,
		to_stderr: true,
	}, d, format, args)
}

func (d *Diag) Debug(format string, args ...interface{}) {

	var cf = getConfig()

	if !debugall && !cf.Debug[d.section] && !cf.Debug["all"] {
		return
	}

	diag(logconf{
		logprio:   syslog.LOG_DEBUG,
		to_stderr: true,
		with_info: true,
	}, d, format, args)
}

func (d *Diag) Problem(format string, args ...interface{}) {
	diag(logconf{
		logprio:   syslog.LOG_WARNING,
		to_stderr: true,
		to_email:  true,
		with_info: true,
	}, d, format, args)
}

func (d *Diag) Bug(format string, args ...interface{}) {
	diag(logconf{
		logprio:    syslog.LOG_ERR,
		to_stderr:  true,
		to_email:   true,
		with_info:  true,
		with_trace: true,
	}, d, format, args)
}

func (d *Diag) Fatal(format string, args ...interface{}) {
	diag(logconf{
		logprio:    syslog.LOG_ERR,
		to_stderr:  true,
		to_email:   true,
		with_info:  true,
		with_trace: true,
	}, d, format, args)

	os.Exit(-1)
}

// ################################################################

func Verbose(format string, args ...interface{}) {
	defaultDiag.Verbose(format, args...)
}
func Problem(format string, args ...interface{}) {
	defaultDiag.Problem(format, args...)
}
func Bug(format string, args ...interface{}) {
	defaultDiag.Bug(format, args...)
}
func Fatal(format string, args ...interface{}) {
	defaultDiag.Fatal(format, args...)
}

// ################################################################

func Init(prog string) {
	progname = prog
	hostname, _ = os.Hostname()
}

func Logger(sect string) *Diag {
	return &Diag{section: sect}
}

func (d *Diag) Logger(sect string) *Diag {
	return &Diag{section: sect}
}

func SetConfig(cf *Config) {
	lock.Lock()
	defer lock.Unlock()
	config = cf
}

func getConfig() *Config {
	lock.RLock()
	defer lock.RUnlock()
	return config
}

// ################################################################

func diag(cf logconf, d *Diag, format string, args []interface{}) {

	var out string

	if cf.with_info {
		pc, file, line, ok := runtime.Caller(2)
		if ok {
			// file is full pathname - trim
			fileshort := clean_filename(file)

			// get function name
			fun := runtime.FuncForPC(pc)
			if fun != nil {
				out = fmt.Sprintf("%s:%d %s(): ", fileshort, line, fun.Name())
			} else {
				out = fmt.Sprintf("%s:%d ?(): ", fileshort, line)
			}
		} else {
			out = "?:?: "
		}
	}

	out = out + fmt.Sprintf(format, args...)

	if cf.to_stderr && usestderr {
		fmt.Fprintln(os.Stderr, out)
	}

	// syslog

	// email
	if cf.to_email {
		send_email(out, cf.with_trace)
	}

}

func send_email(txt string, with_trace bool) {

	// hangs on mac -
	//  cmd := exec.Command("sendmail", "-f", email_from)

	cf := getConfig()
	if cf == nil || cf.Mailto == "" || cf.Mailfrom == "" {
		return
	}

	cmd := exec.Command("cat", "-u")
	p, _ := cmd.StdinPipe()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	cmd.Start()

	fmt.Fprintf(p, "To: %s\nFrom: %s\nSubject: %s daemon error\n\n",
		cf.Mailto, cf.Mailfrom, progname)

	fmt.Fprintf(p, "an error was detected in %s\n\nhost:   %s\npid:    %d\n\n",
		progname, hostname, os.Getpid())

	fmt.Fprintf(p, "error:\n%s\n", txt)

	if with_trace {
		var stack = make([]byte, stack_max)
		stack = stack[:runtime.Stack(stack, true)]
		fmt.Fprintf(p, "\n\n%s\n", stack)
	}

	p.Close()
	cmd.Wait()
}

// trim full pathname to dir/file.go
func clean_filename(file string) string {

	si := strings.LastIndex(file, "/")

	if si == -1 {
		return file
	}

	ssi := strings.LastIndex(file[0:si-1], "/")
	if ssi != -1 {
		si = ssi
	}

	return file[si+1:]
}
