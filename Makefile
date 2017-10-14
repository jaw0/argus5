# Copyright (c) 2017
# Author: Jeff Weisberg <jaw @ tcp4me.com>
# Created: 2017-Oct-14 15:56 (EDT)
# Function: makefile

# where should argus install?
INSTALL_BIN   = /usr/local/bin
INSTALL_SBIN  = /usr/local/sbin
INSTALL_HTDIR = /etc/argus/htdir

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
	cp bin/argusd $(INSTALL_SBIN)
	cp bin/argusctl $(INSTALL_BIN)
	-mkdir -p $(INSTALL_HTDIR)
	cp -R htdir/* $(INSTALL_HTDIR)

clean:
	-rm -rf src/github.com src/golang.org
	-rm -rf pkg/*
	-rm -f bin/*
	-rm -f src/.deps

