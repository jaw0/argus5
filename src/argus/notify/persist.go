// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Sep-08 19:46 (EDT)
// Function: persist to disk

package notify

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"argus/config"
)

func loadIdNo() {

	cf := config.Cf()
	if cf.Datadir == "" {
		dl.Debug("datadir not configured. not loading")
		return
	}

	file := cf.Datadir + "/notno"
	fd, err := os.Open(file)

	if err != nil {
		dl.Debug("cannot open file: %v", err)
		return
	}

	defer fd.Close()
	fmt.Fscan(fd, &idno)

}
func saveIdNo() {

	cf := config.Cf()
	if cf.Datadir == "" {
		dl.Debug("datadir not configured. not saving")
		return
	}
	file := cf.Datadir + "/notno"
	temp := file + ".tmp"

	fd, err := os.Create(temp)
	if err != nil {
		dl.Problem("cannot save notno to '%s': %v", temp, err)
		return
	}

	io.WriteString(fd, fmt.Sprintf("%d\n", idno))
	fd.Close()
	os.Rename(temp, file)

}

// ################################################################

func Load(conf *Conf, idno int) *N {

	if conf == nil {
		dl.Debug("no conf - skipping")
		return nil
	}

	n := &N{
		cf: conf,
	}

	cf := config.Cf()
	if cf.Datadir == "" {
		dl.Debug("datadir not configured. cannot load")
		return nil
	}
	file := cf.Datadir + "/notify/" + fmt.Sprintf("%d", idno)

	js, err := ioutil.ReadFile(file)
	if err != nil {
		dl.Problem("cannot read file: %v", err)
		return nil
	}

	// if the save file is corrupt, the restore may panic
	defer func() {
		if err := recover(); err != nil {
			dl.Problem("error restoring '%s': %v", file, err)
		}
	}()

	err = json.Unmarshal(js, n.p)
	if err != nil {
		dl.Debug("js error: %v", err)
		return nil
	}

	// RSN - discard of old?

	notechan <- n
	return n
}

func (n *N) Save() {

	cf := config.Cf()
	if cf.Datadir == "" {
		dl.Debug("datadir not configured. not saving")
		return
	}
	file := cf.Datadir + "/notify/" + fmt.Sprintf("%d", n.p.IdNo)
	temp := file + ".tmp"

	n.lock.RLock()
	js, _ := json.Marshal(n.p)
	n.lock.RUnlock()

	dl.Debug("persisting to '%s'", file)

	fd, err := os.Create(temp)
	if err != nil {
		dl.Problem("cannot save notification to '%s': %v", temp, err)
		return
	}

	fd.Write(js)
	fd.Close()
	os.Rename(temp, file)
}
