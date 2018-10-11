# Copyright (c) 2017
# Author: Jeff Weisberg <jaw @ tcp4me.com>
# Created: 2017-Oct-14 15:56 (EDT)
# Function: makefile

# where should argus install?
INSTALL_BIN   = /usr/local/bin
INSTALL_SBIN  = /usr/local/sbin
# see also src/argus/conf.go
INSTALL_HTDIR = /usr/local/share/argus/htdir

################################################################

ROOT!=pwd
BIN=src/cmd/argusd src/cmd/argusctl
GO=env GOPATH=$(ROOT) go

all: src/.deps
	for x in $(BIN); do \
		(cd $$x; $(GO) install ); \
	done

src/.deps: deps
	for d in `cat deps`; do \
		$(GO) get -insecure $$d; \
	done
	touch src/.deps

install: all
	-mkdir -p $(INSTALL_BIN) $(INSTALL_SBIN) $(INSTALL_HTDIR)
	cp bin/argusd $(INSTALL_SBIN)
	cp bin/argusctl $(INSTALL_BIN)
	cp -R htdir/* $(INSTALL_HTDIR)

clean:
	-rm -rf src/github.com src/golang.org
	-rm -rf pkg/*
	-rm -f bin/*
	-rm -f src/.deps
