// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Sep-09 20:49 (EDT)
// Function:

package monitor

import (
	_ "argus/monitor/asterisk"
	_ "argus/monitor/compute"
	_ "argus/monitor/database"
	_ "argus/monitor/dns"
	_ "argus/monitor/freeswitch"
	_ "argus/monitor/isforced"
	_ "argus/monitor/ping"
	_ "argus/monitor/prog"
	_ "argus/monitor/snmp"
	_ "argus/monitor/tcp"
	_ "argus/monitor/udp"
	_ "argus/monitor/url"
)
