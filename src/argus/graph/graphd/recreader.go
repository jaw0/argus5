// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Oct-02 19:47 (EDT)
// Function: read data buffered circularly

package graphd

import (
	"os"
)

const BUFSIZE = 4096

type RecReader struct {
	f        *os.File
	buf      [BUFSIZE]byte
	recsize  int
	sectoff  int64
	sectsize int64
	idx      int64
	idxno    int32
	sectpos  int64
	bpos     int
	bend     int
}

func NewRecReader(f *os.File, start int64, size int, idx int32, nmax int32) *RecReader {

	rr := &RecReader{
		f:        f,
		recsize:  size,
		sectoff:  start,
		sectsize: int64(size) * int64(nmax),
		idx:      int64(idx) * int64(size),
		idxno:    idx,
	}

	return rr
}

func (rr *RecReader) Read(p []byte) (int, error) {
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

		// read more
		rlen := rr.sectsize - rr.sectpos
		if rlen > BUFSIZE {
			rlen = BUFSIZE
		}
		r, err := rr.f.Read(rr.buf[:rlen])
		if err != nil {
			return nr, err
		}
		rr.bend = r
		if rr.sectpos+int64(r) >= rr.sectsize {
			// wrap back around
			rr.Seek(-rr.idxno)
		}
	}
}

func (rr *RecReader) Seek(recno int32) {
	rr.sectpos = (int64(recno)*int64(rr.recsize) + rr.idx + rr.sectsize) % rr.sectsize
	rr.bpos = 0
	rr.bend = 0
	rr.f.Seek(rr.sectpos+rr.sectoff, 0)
}
