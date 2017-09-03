// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Sep-02 16:51 (EDT)
// Function: monitor elements

package monel

import (
	"argus/argus"
	"argus/configure"
)

// Service, Group, Alias
type Moneler interface {
	Persist(map[string]interface{})
	Restore(map[string]interface{})
	Config(*configure.CF) error
	Init() error
}

type Conf struct {
}
type Persist struct {
}

type M struct {
	Me     Moneler
	Parent *M
	cf     Conf
	p      Persist
	config *configure.CF
	// stats, logs, notif
	// ov, anno
}

func New(me Moneler, parent *M) *M {

	m := &M{
		Me:     me,
		Parent: parent,
	}
	return m
}

func (m *M) Config(conf *configure.CF) error {

	// RSN - configure
	// conf.InitFromConfig(&m.cf, "monel", prefix)
	err := m.Me.Config(conf)
	if err != nil {
		return err
	}
	m.Restore()
	m.Init()

	return nil
}

func (m *M) Init() {

	// RSN - init
	// byname{} = m

	m.Me.Init()
}

func (m *M) Update(status argus.Status) {

}

func (m *M) Debug(x ...interface{}) {

}
func (m *M) Loggit(x ...interface{}) {

}

func (m *M) Persist() {

}

func (m *M) Restore() {

	// defer recover
}
