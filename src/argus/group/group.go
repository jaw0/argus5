// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Sep-06 00:21 (EDT)
// Function:

package group

import (
	"argus/monel"
)

type Group struct {
	mon *monel.M
}

// construction starts here:
func New(conf *configure.CF, parent *monel.M) (*monel.M, error) {

	g := &Group{}

	g.mon = monel.New(g, parent)

	err := g.mon.Config(conf)
	if err != nil {
		return nil, err
	}

	return g.mon, nil
}

func (g *Group) Config(conf *configure.CF) error {

	//conf.InitFromConfig(&g.cf, "group", "")

	return nil
}

func (g *Group) Init() error {

	return nil
}

// destruction
func (g *Group) Recycle() {

}
