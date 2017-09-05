// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Sep-04 10:41 (EDT)
// Function: save/restore to disk

package monel

import (
	"encoding/json"
	"io/ioutil"
	"os"

	"github.com/mitchellh/mapstructure"

	"argus/config"
	"argus/diag"
)

func (m *M) Persist() {

	dat := make(map[string]interface{})
	m.persist(dat)
	m.Me.Persist(dat)

	cf := config.Cf()
	file := cf.Datadir + "/stats/" + m.Pathname("", "")
	temp := file + ".tmp"
	dl.Debug("persisting to '%s'", file)

	js, _ := json.Marshal(dat)

	fd, err := os.Create(temp)
	if err != nil {
		diag.Problem("cannot save stats to '%s': %v", temp, err)
		return
	}

	fd.Write(js)
	fd.Close()
	os.Rename(temp, file)
}

func (m *M) Restore() {

	cf := config.Cf()
	file := cf.Datadir + "/stats/" + m.Pathname("", "")
	dl.Debug("restoring from '%s'", file)

	js, err := ioutil.ReadFile(file)
	if err != nil {
		dl.Debug("cannot read file: %v", err)
		return
	}

	// if the save file is corrupt, the restore may panic
	defer func() {
		if err := recover(); err != nil {
			diag.Problem("error restoring '%s': %v", file, err)
		}
	}()

	dat := make(map[string]interface{})
	err = json.Unmarshal(js, &dat)
	if err != nil {
		dl.Debug("js error: %v", err)
		return
	}

	m.restore(dat)
	m.Me.Restore(dat)
}

func (m *M) persist(pm map[string]interface{}) {
	pm["monel"] = &m.p
}

func (m *M) restore(pm map[string]interface{}) {

	p := &m.p

	err := mapstructure.Decode(pm["monel"].(map[string]interface{}), p)
	if err != nil {
		return
	}
}
