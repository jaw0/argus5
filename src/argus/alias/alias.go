// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Sep-06 00:21 (EDT)
// Function: for Sydney Bristow

package alias

import (
	"errors"

	"argus.domain/argus/argus"
	"argus.domain/argus/configure"
	"argus.domain/argus/monel"
	"github.com/jaw0/acgo/diag"
)

type Alias struct {
	mon    *monel.M
	AName  string
	Target string
	Object *monel.M
}

var dl = diag.Logger("alias")

// construction starts here:
func New(conf *configure.CF, parent *monel.M) (*monel.M, error) {

	a := &Alias{}

	a.mon = monel.New(a, parent)

	err := a.mon.Config(conf)
	if err != nil {
		return nil, err
	}

	return a.mon, nil
}

func (a *Alias) Config(conf *configure.CF) error {

	//conf.InitFromConfig(&a.cf, "alias", "")
	a.AName = conf.Name
	a.Target = conf.Extra[0]

	if a.Target == "" {
		return errors.New("invalid alias - target not specified")
	}

	a.mon.SetNames(a.AName, a.AName, a.AName)

	return nil
}

func (a *Alias) Init() error {

	return nil
}

func (a *Alias) DoneConfig() {
	a.aliasLookup()
}

// destruction
func (a *Alias) Recycle() {

}

func (a *Alias) Persist(pm map[string]interface{}) {
}
func (a *Alias) Restore(pm map[string]interface{}) {

}

// ################################################################

func (a *Alias) aliasLookup() *monel.M {

	if a.Object != nil {
		return a.Object
	}

	t := monel.Find(a.Target)

	if t == nil {
		a.mon.ConfCF.Error("cannot resolve alias '%s' -> '%s'", a.AName, a.Target)
		return nil
	}

	t.AddParent(a.mon)

	a.Object = t
	return t
}

func (a *Alias) Children() []*monel.M {
	m := a.aliasLookup()
	if m == nil {
		return nil
	}

	if len(m.Children) != 0 {
		return m.Children
	}
	return []*monel.M{m}
}
func (a *Alias) Self() *monel.M {
	if a.Object != nil {
		return a.Object
	}
	return a.mon
}

func (a *Alias) WebJson(md map[string]interface{}) {

	if a.Object == nil {
		md["testinfo"] = struct {
			Cannot_Find string
		}{a.Target}
		return
	}
}
func (a *Alias) WebMeta(md map[string]interface{}) {
}
func (a *Alias) Dump(dx argus.Dumper) {
	argus.Dump(dx, "alias/target", a.Target)
}

func (a *Alias) CheckNow() {
}

func (a *Alias) GraphList(pfx string, gl []interface{}) []interface{} {
	return gl
}
