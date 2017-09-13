// Copyright (c) 2017
// Author: Jeff Weisberg <jaw @ tcp4me.com>
// Created: 2017-Sep-12 21:41 (EDT)
// Function: database tests

package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"argus/configure"
	"argus/diag"
	"argus/service"
)

type Conf struct {
	Dbtype string
	Dsn    string
	Sql    string
}

type DB struct {
	S  *service.Service
	Cf Conf
}

var dl = diag.Logger("database")

func init() {
	// register with service factory
	service.Register("DB", New)
}

func New(conf *configure.CF, s *service.Service) service.Monitor {
	p := &DB{S: s}
	return p
}

func (d *DB) Config(conf *configure.CF, s *service.Service) error {

	conf.InitFromConfig(&d.Cf, "db", "")

	// validate
	if d.Cf.Dbtype == "" {
		return errors.New("db type not specified")
	}
	if d.Cf.Dsn == "" {
		return errors.New("dsn not specified")
	}
	if d.Cf.Sql == "" {
		return errors.New("sql not specified")
	}

	// set names + labels
	uname := "DB_" + d.Cf.Dsn

	s.SetNames(uname, "DB", d.Cf.Dsn)

	return nil
}

func (d *DB) Init() error {
	return nil
}

func (d *DB) Recycle() {
}
func (d *DB) Abort() {
}
func (d *DB) DoneConfig() {
}

func (d *DB) Start(s *service.Service) {

	s.Debug("prog start")
	defer s.Done()

	db, err := sql.Open(d.Cf.Dbtype, d.Cf.Dsn)
	if err != nil {
		dl.Debug("open failed: %v", err)
		s.Fail("open failed")
		return
	}
	defer db.Close()

	timeout := time.Duration(d.S.Cf.Timeout) * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	row := db.QueryRowContext(ctx, d.Cf.Sql)

	var vi interface{}
	err = row.Scan(&vi)
	if err != nil {
		dl.Debug("select failed: %v", err)
		s.Fail(fmt.Sprintf("select failed: %v", err))
		return
	}

	res := fmt.Sprintf("%v", vi)

	s.CheckValue(res, "data")
}
