// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Oct-01 11:05 (EDT)
// Function: graphs

package graph

import (
	"argus/argus"
	"argus/graph/graphd"
)

// RSN - pluggable graphing subsystems: graphite, grafana, influxdb, ...
type Grapher interface{}

func Add(file string, when int64, status argus.Status, val float64, yn float64, dn float64) {

	graphd.Add(file, when, status, val, yn, dn)
}
