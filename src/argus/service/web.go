// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Sep-17 14:31 (EDT)
// Function:

package service

import (
	"argus/argus"
)

type darpStatusResult struct {
	Status argus.Status
	Result string
}

func (s *Service) WebJson(md map[string]interface{}) {

	s.mon.Lock.RLock()
	defer s.mon.Lock.RUnlock()

	md["lasttest"] = s.Lasttest
	darp := make(map[string]darpStatusResult)
	md["darp"] = darp
	md["hostname"] = s.check.Hostname()

	for k, st := range s.p.Statuses {
		r := s.p.Results[k]
		darp[k] = darpStatusResult{st, r}
	}

	testinfo := make(map[string]interface{})
	s.check.WebJson(testinfo)
	md["testinfo"] = testinfo
}

func (s *Service) WebMeta(md map[string]interface{}) {

	s.mon.Lock.RLock()
	defer s.mon.Lock.RUnlock()

	md["lasttest"] = s.Lasttest
	md["result"] = limitString(s.mon.P.Result, 32)
}
