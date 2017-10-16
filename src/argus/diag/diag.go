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
	"context"
	"flag"
	"fmt"
	"log/syslog"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"
)

// defaults
const (
	stack_max = 1048576
)

var hostname = "?"
var progname = "?"
var debugall = false
var usestderr = true

var lock sync.RWMutex
var config = &Config{}
var defaultDiag = &Diag{"default"}

var slog *syslog.Writer

type Diag struct {
	section string
}

type Config struct {
	Mailto   string
	Mailfrom string
	Facility string
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
	diag(logconf{
		logprio:   syslog.LOG_INFO,
		to_stderr: true,
	}, defaultDiag, format, args)
}
func Problem(format string, args ...interface{}) {
	diag(logconf{
		logprio:   syslog.LOG_WARNING,
		to_stderr: true,
		to_email:  true,
		with_info: true,
	}, defaultDiag, format, args)
}
func Bug(format string, args ...interface{}) {
	diag(logconf{
		logprio:    syslog.LOG_ERR,
		to_stderr:  true,
		to_email:   true,
		with_info:  true,
		with_trace: true,
	}, defaultDiag, format, args)
}
func Fatal(format string, args ...interface{}) {
	diag(logconf{
		logprio:    syslog.LOG_ERR,
		to_stderr:  true,
		to_email:   true,
		with_info:  true,
		with_trace: true,
	}, defaultDiag, format, args)

	os.Exit(-1)
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

	if slog == nil {
		openSyslog(cf.Facility)
	}
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
			fileshort := cleanFilename(file)

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
	if slog != nil {
		sendToSyslog(cf.logprio, out)
	}

	// email
	if cf.to_email {
		sendEmail(out, cf.with_trace)
	}

}

func sendEmail(txt string, with_trace bool) {

	cf := getConfig()
	if cf == nil || cf.Mailto == "" || cf.Mailfrom == "" {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "sendmail", "-t", "-f", cf.Mailfrom)

	p, _ := cmd.StdinPipe()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	cmd.Start()

	go func() {
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
	}()

	cmd.Wait()
}

func sendToSyslog(prio syslog.Priority, msg string) {

	switch prio {
	case syslog.LOG_DEBUG:
		slog.Debug(msg)
	case syslog.LOG_INFO:
		slog.Info(msg)
	case syslog.LOG_NOTICE:
		slog.Notice(msg)
	case syslog.LOG_WARNING:
		slog.Warning(msg)
	case syslog.LOG_ERR:
		slog.Err(msg)
	case syslog.LOG_ALERT:
		slog.Alert(msg)
	case syslog.LOG_EMERG:
		slog.Emerg(msg)
	case syslog.LOG_CRIT:
		slog.Crit(msg)
	}
}

func openSyslog(fac string) {

	var p syslog.Priority

	switch strings.ToLower(fac) {
	case "kern":
		p = syslog.LOG_KERN
	case "user":
		p = syslog.LOG_USER
	case "mail":
		p = syslog.LOG_MAIL
	case "daemon":
		p = syslog.LOG_DAEMON
	case "auth":
		p = syslog.LOG_AUTH
	case "syslog":
		p = syslog.LOG_SYSLOG
	case "lpr":
		p = syslog.LOG_LPR
	case "news":
		p = syslog.LOG_NEWS
	case "uucp":
		p = syslog.LOG_UUCP
	case "cron":
		p = syslog.LOG_CRON
	case "authpriv":
		p = syslog.LOG_AUTHPRIV
	case "ftp":
		p = syslog.LOG_FTP
	case "local0":
		p = syslog.LOG_LOCAL0
	case "local1":
		p = syslog.LOG_LOCAL1
	case "local2":
		p = syslog.LOG_LOCAL2
	case "local3":
		p = syslog.LOG_LOCAL3
	case "local4":
		p = syslog.LOG_LOCAL4
	case "local5":
		p = syslog.LOG_LOCAL5
	case "local6":
		p = syslog.LOG_LOCAL6
	case "local7":
		p = syslog.LOG_LOCAL7
	default:
		return
	}

	slog, _ = syslog.New(p, progname)
}

// trim full pathname to dir/file.go
func cleanFilename(file string) string {

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
