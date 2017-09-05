// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Sep-04 13:48 (EDT)
// Function: read files

package construct

import (
	"bufio"
	"os"
	"sort"
	"strings"

	"argus/argus"
)

type openfile struct {
	fd   *os.File
	bfd  *bufio.Reader
	file string
	line int
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
	if err != nil && s.IsDir() {
		f.files = filesInDir(file)
	} else {
		f.files = append(f.files, file)
	}

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

	fd, err := os.Open(file)
	if err != nil {
		argus.Loggit("cannot open config file '%s': %v", file, err)
		return false
	}
	bfd := bufio.NewReader(fd)

	o := &openfile{fd, bfd, file, 0}

	if f.curr != nil {
		f.opens = append(f.opens, f.curr)
	}
	f.curr = o
	return true
}

func (f *Files) nextFile() bool {

	if f.curr != nil {
		f.curr.fd.Close()
		f.curr = nil
	}

	if len(f.opens) != 0 {
		f.curr = f.opens[len(f.opens)-1]
		f.opens = f.opens[:len(f.opens)-1]
		return true
	}

	if len(f.files) != 0 {
		file := f.files[0]
		f.files = f.files[1:]

		ok := f.openFile(file)
		if ok {
			return true
		}
		return f.nextFile()
	}

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
		l := string(b)

		// remove comments, whitespace
		// convert \# -> #
		l = cleanLine(l)

		if l == "" {
			continue
		}

		// include
		if strings.HasPrefix(l, "include ") {
			f.include(f.includeFileName(l))
			continue
		}

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

	if file[0] != '/' && f.basedir != "" {
		file = f.basedir + "/" + file
	}

	f.opens = append(f.opens, f.curr)
	f.openFile(file)
}

// remove whitespace, comments
func cleanLine(l string) string {

	buf := make([]byte, len(l))
	j := 0
	var p byte

	// remove comments, convert \# -> #
	for i := 0; i < len(l); i++ {
		if l[i] != '#' {
			buf[j] = byte(l[i])
			p = l[i]
			j++
			continue
		}

		if p == '\\' {
			// convert \# => #
			buf[j-1] = byte(l[i])
		} else {
			// skip comment to end
			break
		}
	}

	return strings.Trim(string(buf), " \t\r\n")
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
