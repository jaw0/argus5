// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Sep-06 00:21 (EDT)
// Function: for Sydney Bristow

package alias

import (
	"errors"

	"argus/api"
	"argus/configure"
	"argus/diag"
	"argus/monel"
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
	a.Target = conf.Extra

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

	// short ckt straight to my parents
	t.AddParent(a.mon.Parent[0])

	a.Object = t
	return t
}

func (a *Alias) Children() []*monel.M {
	m := a.aliasLookup()
	if m == nil {
		return nil
	}
	return m.Children
}

func (a *Alias) WebJson(md map[string]interface{}) {
}

func (a *Alias) Dump(ctx *api.Context) {
	ctx.SendKVP("alias/target", a.Target)
}
