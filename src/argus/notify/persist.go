// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Sep-08 19:46 (EDT)
// Function: persist to disk

package notify

import (
	"fmt"

	"argus.domain/argus/argus"
	"argus.domain/argus/config"
)

func loadIdNo() {

	cf := config.Cf()
	if cf.Datadir == "" {
		dl.Debug("datadir not configured. not loading")
		return
	}

	file := cf.Datadir + "/notno"

	err := argus.Load(file, &idno)

	if err != nil {
		dl.Debug("cannot open file: %v", err)
		return
	}
}

func saveIdNo() {

	cf := config.Cf()
	if cf.Datadir == "" {
		dl.Debug("datadir not configured. not saving")
		return
	}
	file := cf.Datadir + "/notno"

	err := argus.Save(file, &idno)

	if err != nil {
		dl.Problem("cannot save notno to '%s': %v", file, err)
		return
	}
}

// ################################################################

func Load(conf *Conf, idno int, mon Remover) *N {

	if conf == nil {
		dl.Debug("no conf - skipping")
		return nil
	}

	n := &N{
		cf:  conf,
		mon: mon,
	}

	cf := config.Cf()
	if cf.Datadir == "" {
		dl.Debug("datadir not configured. cannot load")
		return nil
	}
	file := cf.Datadir + "/notify/" + fmt.Sprintf("%d", idno)

	err := argus.Load(file, &n.p)

	if err != nil {
		dl.Debug("cannot load notify: %v", err)
		return nil
	}

	// RSN - discard of old/outdated?

	notechan <- n
	return n
}

// lock should already be held
func (n *N) Save() {

	cf := config.Cf()
	if cf.Datadir == "" {
		dl.Debug("datadir not configured. not saving")
		return
	}
	file := cf.Datadir + "/notify/" + fmt.Sprintf("%d", n.p.IdNo)

	dl.Debug("persisting to '%s'", file)

	err := argus.Save(file, n.p)

	if err != nil {
		dl.Problem("cannot save notification to '%s': %v", file, err)
		return
	}

}
