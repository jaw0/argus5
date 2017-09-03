// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Sep-02 19:12 (EDT)
// Function:

package service

import (
	"github.com/mitchellh/mapstructure"
)

// this should be fast-ish
func (s *Service) Persist(pm map[string]interface{}) {

	pm["service"] = &s.p
}

// this need not be as fast.
func (s *Service) Restore(pm map[string]interface{}) {

	p := &s.p

	err := mapstructure.Decode(pm["service"].(map[string]interface{}), p)
	if err != nil {
		return
	}
}
