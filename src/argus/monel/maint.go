// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Sep-06 22:48 (EDT)
// Function: periodic maintenance

package monel

import (
	"argus.domain/argus/sched"
)

var monelCron = sched.NewFunc(&sched.Conf{
	Freq:  3600,
	Phase: 3600,
	Auto:  true,
	Text:  "MonEl Maint",
}, func() {
	lock.RLock()
	defer lock.RUnlock()

	for _, m := range byname {
		m.StatsPeriodic()
		m.Persist()
	}
})
