// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Sep-14 11:02 (EDT)
// Function: save+restore

package argus

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
)

func Load(file string, thing interface{}) (er error) {

	js, err := ioutil.ReadFile(file)
	if err != nil {
		return err
	}

	// if the save file is corrupt, the restore may panic
	defer func() {
		if err := recover(); err != nil {
			er = fmt.Errorf("error: %v", err)
		}
	}()

	err = json.Unmarshal(js, thing)
	if err != nil {
		return err
	}

	return nil
}

func Save(file string, thing interface{}) error {

	temp := file + ".tmp"

	js, _ := json.Marshal(thing)

	fd, err := os.Create(temp)
	if err != nil {
		return err
	}

	fd.Write(js)
	fd.Close()
	os.Rename(temp, file)

	return nil
}
