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
	"argus/notify"
)

func (m *M) Persist() {

	dat := make(map[string]interface{})

	cf := config.Cf()
	if cf.Datadir == "" {
		return
	}
	file := cf.Datadir + "/stats/" + m.Pathname("", "")
	temp := file + ".tmp"

	m.StatsPeriodic()

	m.Lock.RLock()
	m.persist(dat)
	m.Me.Persist(dat)
	js, _ := json.Marshal(dat)
	m.Lock.RUnlock()

	dl.Debug("persisting to '%s'", file)

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
	if cf.Datadir == "" {
		return
	}
	file := cf.Datadir + "/stats/" + m.Pathname("", "")
	dl.Debug("restoring from '%s'", file)

	js, err := ioutil.ReadFile(file)
	if err != nil {
		dl.Debug("cannot read file: %v", err)
		return
	}

	// if the save file is corrupt, the restore may panic
	// in devmode, let it crash
	defer func() {
		if !cf.DevMode {
			if err := recover(); err != nil {
				diag.Problem("error restoring '%s': %v", file, err)
			}
		}
	}()

	dat := make(map[string]interface{})
	err = json.Unmarshal(js, &dat)
	if err != nil {
		dl.Debug("js error: %v", err)
		return
	}

	m.Lock.Lock()
	m.restore(dat)
	m.Me.Restore(dat)
	m.Lock.Unlock()
}

func (m *M) persist(pm map[string]interface{}) {

	notes := []int{}
	for _, n := range m.Notifies {
		n.Save()
		notes = append(notes, n.IdNo())
	}

	pm["monel"] = &m.P
	pm["notify"] = notes
}

func (m *M) restore(pm map[string]interface{}) {

	p := &m.P

	err := mapstructure.Decode(pm["monel"].(map[string]interface{}), p)
	if err != nil {
		dl.Debug("err: %v", err)
		return
	}

	notes := pm["notify"].([]interface{})

	for _, idi := range notes {
		id := idi.(int)
		n := notify.Load(m.NotifyCf, id)
		if n != nil {
			m.Notifies = append(m.Notifies, n)
		}
	}
}
