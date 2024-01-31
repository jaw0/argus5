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

	_ "github.com/SAP/go-hdb/driver"
	_ "github.com/denisenkom/go-mssqldb"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	// add more drivers here (and update config below)

	"argus.domain/argus/configure"
	"github.com/jaw0/acdiag"
	"argus.domain/argus/service"
)

type Conf struct {
	Dbtype   string
	Dsn      string // actual dsn to use, if not set, build dsn from below
	User     string
	Pass     string
	Hostname string
	Dbname   string
	Dbparam  string // typically, of the form: var1=val;var2=val
	Sql      string // select statement to execute
}

type DB struct {
	S      *service.Service
	Cf     Conf
	dsn    string
	dsndpy string // display version of dsn (password obscured)
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

func (d *DB) PreConfig(conf *configure.CF, s *service.Service) error {
	return nil
}
func (d *DB) Config(conf *configure.CF, s *service.Service) error {

	conf.InitFromConfig(&d.Cf, "db", "")

	uname := ""
	host := d.Cf.Hostname

	// validate
	if d.Cf.Dbtype == "" {
		return errors.New("db type not specified")
	}
	if d.Cf.Dsn != "" {
		// use dsn exactly as provided
		d.dsn = d.Cf.Dsn
		d.dsndpy = d.Cf.Dsn
		uname = "DB_" + d.Cf.Dsn
	} else {
		// construct dsn
		switch d.Cf.Dbtype {
		case "postgres":
			// https://godoc.org/github.com/lib/pq
			// postgres://pqgotest:password@localhost/pqgotest?sslmode=verify-full
			d.dsn = "postgres://" + d.Cf.User + ":" + d.Cf.Pass + "@" + host +
				"/" + d.Cf.Dbname + "?" + d.Cf.Dbparam
			d.dsndpy = "postgres://" + d.Cf.User + ":" + "****" + "@" + host +
				"/" + d.Cf.Dbname + "?" + d.Cf.Dbparam

		case "mysql":
			// https://godoc.org/github.com/go-sql-driver/mysql
			// username:password@protocol(address)/dbname?param=value
			// user:password@tcp(localhost:5555)/dbname?tls=skip-verify
			d.dsn = d.Cf.User + ":" + d.Cf.Pass + "@tcp(" + host +
				")/" + d.Cf.Dbname + "?" + d.Cf.Dbparam
			d.dsndpy = d.Cf.User + ":" + "****" + "@tcp(" + host +
				")/" + d.Cf.Dbname + "?" + d.Cf.Dbparam

		case "sqlserver":
			// https://github.com/denisenkom/go-mssqldb
			// sqlserver://username:password@host/instance?param1=value&param2=value
			d.dsn = "sqlserver://" + d.Cf.User + ":" + d.Cf.Pass + "@" + host +
				"/" + d.Cf.Dbname + "?" + d.Cf.Dbparam
			d.dsndpy = "sqlserver://" + d.Cf.User + ":" + "****" + "@" + host +
				"/" + d.Cf.Dbname + "?" + d.Cf.Dbparam

		case "hdb":
			// https://github.com/SAP/go-hdb
			// hdb://user:password@host:port
			d.dsn = "hdb://" + d.Cf.User + ":" + d.Cf.Pass + "@" + host +
				"/" + d.Cf.Dbname + "?" + d.Cf.Dbparam
			d.dsndpy = "hdb://" + d.Cf.User + ":" + "****" + "@" + host +
				"/" + d.Cf.Dbname + "?" + d.Cf.Dbparam
		}

		uname = "DB_" + d.Cf.Dbtype + "_" + d.Cf.Dbname + "@" + host
	}

	if d.dsn == "" {
		return errors.New("cannot determine dsn")
	}
	if d.Cf.Sql == "" {
		return errors.New("sql not specified")
	}

	// set names + labels

	s.SetNames(uname, "DB", d.Cf.Dsn)

	return nil
}

func (d *DB) Init() error {
	return nil
}

func (d *DB) Hostname() string {
	return ""
}
func (d *DB) Priority() bool {
	return false
}
func (d *DB) Recycle() {
}
func (d *DB) Abort() {
}
func (d *DB) DoneConfig() {
}

func (d *DB) Start(s *service.Service) {

	s.Debug("db start")
	defer s.Done()

	db, err := sql.Open(d.Cf.Dbtype, d.dsn)
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

func (d *DB) DumpInfo() map[string]interface{} {
	return map[string]interface{}{
		"service/database/CF":  d.Cf,
		"service/database/dsn": d.dsndpy,
	}
}
func (d *DB) WebJson(md map[string]interface{}) {
	md["DSN"] = d.dsndpy
	md["SQL"] = d.Cf.Sql
}
