// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Oct-02 19:47 (EDT)
// Function: read data buffered circularly

package graphd

import (
	"os"
)

const BUFSIZE = 4096

type CbufReader struct {
	f        *os.File
	buf      [BUFSIZE]byte
	sectoff  int64
	sectsize int64
	sectpos  int64
	bpos     int
	bend     int
}

func NewCbufReader(f *os.File, start int64, size int64) *CbufReader {

	rr := &CbufReader{
		f:        f,
		sectoff:  start,
		sectsize: size,
	}

	return rr
}

func (rr *CbufReader) Read(p []byte) (int, error) {
	n := len(p)
	nr := 0

	for {
		if n <= rr.bend-rr.bpos {
			// copy data from buffer
			copy(p, rr.buf[rr.bpos:])
			rr.bpos += n
			nr += n
			return nr, nil
		}
		if rr.bend != rr.bpos {
			// copy remaining buffered data
			copy(p, rr.buf[rr.bpos:rr.bend])
			len := rr.bend - rr.bpos
			n -= len
			nr += len
			p = p[len:]
			rr.bpos = 0
			rr.bend = 0
		}
		if rr.bpos == rr.bend {
			rr.bpos = 0
			rr.bend = 0
		}

		// read more
		rlen := rr.sectsize - rr.sectpos
		//dl.Debug("size %d - pos %d = rlen %d", rr.sectsize, rr.sectpos, rlen)

		if rlen > BUFSIZE {
			rlen = BUFSIZE
		}

		if rr.sectpos&^(BUFSIZE-1) != (rr.sectpos+rlen)&^(BUFSIZE-1) {
			// align
			rlen -= (rr.sectpos + rlen) & (BUFSIZE - 1)
			//dl.Debug("aligned %d", rlen)
		}

		r, err := rr.f.Read(rr.buf[:rlen])
		if err != nil {
			return nr, err
		}
		dl.Debug("read %d %d", rlen, r)
		rr.bend = r
		rr.sectpos += int64(r)
		if rr.sectpos >= rr.sectsize {
			// wrap back around
			rr.Seek(0)
		}
	}
}

func (rr *CbufReader) Seek(pos int64) {
	rr.sectpos = pos
	rr.f.Seek(rr.sectpos+rr.sectoff, 0)
}
