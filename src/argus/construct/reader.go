// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Sep-04 13:48 (EDT)
// Function: read files

package construct

import (
	"bufio"
	"io"
	"os"
	"os/exec"
	"sort"
	"strings"

	"argus/argus"
)

type openfile struct {
	closer func()
	bfd    *bufio.Reader
	file   string
	line   int
}

type Files struct {
	curr    *openfile
	basedir string
	files   []string
	opens   []*openfile
	ungot   string
}

// file or directory
func NewReader(file string) *Files {

	f := &Files{}

	s, err := os.Lstat(file)
	if err == nil && s.IsDir() {
		f.basedir = file
		f.files = filesInDir(file)
	} else {
		sl := strings.LastIndexByte(file, '/')
		if sl != -1 {
			f.basedir = file[:sl]
		}

		f.files = append(f.files, file)
	}

	dl.Debug("f %v", f)
	f.nextFile()
	return f
}

func (f *Files) CurrFile() string {

	if f.curr != nil {
		return f.curr.file
	}

	if len(f.files) != 0 {
		return f.files[0]
	}
	return ""
}

func (f *Files) CurrLine() int {
	if f.curr == nil {
		return 0
	}
	return f.curr.line
}

func (f *Files) openFile(file string) bool {

	pathname := file
	if file[0] != '/' && f.basedir != "" {
		pathname = f.basedir + "/" + file
	}

	fd, closer, err := openOrPopen(pathname)
	if err != nil {
		argus.Loggit("cannot open config file '%s': %v", file, err)
		return false
	}
	bfd := bufio.NewReader(fd)

	o := &openfile{closer, bfd, file, 0}

	if f.curr != nil {
		f.opens = append(f.opens, f.curr)
	}
	f.curr = o
	return true
}

func (f *Files) nextFile() bool {

	if f.curr != nil {
		dl.Debug("close curr")
		f.curr.closer()
		f.curr = nil
	}

	if len(f.opens) != 0 {
		dl.Debug("pop open file")
		f.curr = f.opens[len(f.opens)-1]
		f.opens = f.opens[:len(f.opens)-1]
		return true
	}

	if len(f.files) != 0 {
		file := f.files[0]
		f.files = f.files[1:]

		dl.Debug("open file %s", file)
		ok := f.openFile(file)
		if ok {
			return true
		}
		return f.nextFile()
	}

	dl.Debug("no files")
	return false
}

func (f *Files) NextLine() (string, bool) {

	if f.ungot != "" {
		x := f.ungot
		f.ungot = ""
		return x, true
	}

	if f.curr == nil {
		return "", false
	}

	var a []byte

	for {
		b, _, err := f.curr.bfd.ReadLine()
		if err != nil {
			ok := f.nextFile()
			if !ok {
				return "", false
			}
			continue
		}
		f.curr.line++

		if len(b) == 0 {
			continue
		}

		a = append(a, b...)

		if a[len(a)-1] == '\\' {
			a = a[:len(a)-1]
			continue
		}

		// remove comments, whitespace
		// convert \# -> #, etal
		a = cleanLine(a)

		if len(a) == 0 {
			continue
		}

		l := string(a)

		// include
		if strings.HasPrefix(l, "include ") {
			f.include(f.includeFileName(l))
			a = nil
			continue
		}

		//dl.Debug("line> [%d] '%s'", len(l), l)

		return l, true
	}
}

func (f *Files) UnGetLine(l string) {

	f.ungot = l
}

func (f *Files) include(file string) {

	if file == "" {
		return
	}

	f.opens = append(f.opens, f.curr)
	f.openFile(file)
}

// remove whitespace, comments
func cleanLine(l []byte) []byte {

	buf := make([]byte, len(l))
	j := 0

	// remove comments, convert \# -> #
	// convert \r\n...
	for i := 0; i < len(l); i++ {
		c := l[i]

		if (c == ' ' || c == '\t') && j == 0 {
			// skip leading white
			continue
		}

		if c == '\\' && i != len(l)-1 {
			i++
			switch l[i] {
			case 'n':
				buf[j] = '\n'
			case 'r':
				buf[j] = '\r'
			case 't':
				buf[j] = '\t'
			case '#':
				buf[j] = '#'

				// RSN - \xFF
			default:
				buf[j] = c
				j++
				buf[j] = l[i]
			}
			j++
			continue
		}

		if c == '#' {
			break
		}
		buf[j] = c
		j++
	}

	// trim trailing white
Loop:
	for j > 0 {
		switch buf[j-1] {
		case ' ', '\t', '\n', '\r':
			j--
		default:
			break Loop
		}
	}

	return buf[:j]
}

func (f *Files) includeFileName(l string) string {

	a := strings.Index(l, `"`)
	b := strings.LastIndex(l, `"`)

	if a == -1 || b == -1 {
		// syntax error?
		argus.Loggit("invalid include. file %s, line %d", f.curr.file, f.curr.line)
		return ""
	}

	return l[a+1 : b]

}

func filesInDir(dir string) []string {

	f, err := os.Open(dir)
	if err != nil {
		argus.Loggit("cannot open config dir '%s': %v", dir, err)
		return nil
	}

	all, _ := f.Readdirnames(-1)
	f.Close()

	var use []string

	for _, d := range all {
		// skip version control and backup files, ...
		if d == "" {
			continue
		}
		if d[0] == '.' || d[0] == '#' {
			continue
		}
		if d == "CVS" {
			continue
		}
		if strings.HasSuffix(d, ".bkp") {
			continue
		}

		use = append(use, d)
	}

	sort.Strings(use)
	return use
}

func openOrPopen(pathname string) (io.Reader, func(), error) {

	st, err := os.Stat(pathname)

	if err == nil && st.Mode()&0111 != 0 {
		// popen
		fd, closer, err := popen(pathname)
		if err != nil {
			return nil, nil, err
		}
		return fd, closer, nil
	}

	// open normal file
	fd, err := os.Open(pathname)
	if err != nil {
		return nil, nil, err
	}

	return fd, func() { fd.Close() }, nil
}

func popen(pathname string) (io.Reader, func(), error) {

	cmd := exec.Command(pathname)
	stdout, err := cmd.StdoutPipe()

	if err != nil {
		return nil, nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, nil, err
	}

	return stdout, func() { cmd.Wait() }, nil
}
