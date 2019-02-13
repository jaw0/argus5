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
DATE!=date +'%Y%m%d'

BIN=src/cmd/argusd src/cmd/argusctl
GO=env GOPATH=$(ROOT) go

VERSION=dev-$(DATE)
NAME=argus
CTLSOCK=/var/tmp/$(NAME).ctl

all: src/.deps
	(echo package argus; echo const Version = \"$(VERSION)\") > src/argus/argus/version.go
	(echo package argus; echo const ControlSocket = \"$(CTLSOCK)\") > src/argus/argus/ctlsock.go
	@for x in $(BIN); do \
		echo building $$x; \
		(cd $$x; $(GO) install); \
	done
	@echo
	@echo build of argus version $(VERSION) complete
	@echo now run \'make install\'

src/.deps: deps
	@echo fetching dependencies...
	@for d in `cat deps`; do \
		echo + $$d; \
		$(GO) get -insecure $$d; \
	done
	@touch src/.deps

install: all
	-mkdir -p $(INSTALL_BIN) $(INSTALL_SBIN) $(INSTALL_HTDIR)
	cp bin/argusd $(INSTALL_SBIN)/$(NAME)d
	cp bin/argusctl $(INSTALL_BIN)/$(NAME)ctl
	cp -R htdir/* $(INSTALL_HTDIR)
	@echo
	@echo install of argus version $(VERSION) complete

clean:
	-rm -rf src/github.com src/golang.org src/cloud.google.com
	-rm -rf pkg/*
	-rm -f bin/*
	-rm -f src/.deps


################################################################

TESTDIR=/tmp/argus5test

testbuild:
	rm -rf $(TESTDIR)
	git clone $(ROOT) $(TESTDIR)
	cd $(TESTDIR) ; make


dist:
	git archive --format=tar.gz --prefix=argus-$(VERSION)/ HEAD > argus-$(VERSION).tgz

www-code:
	scp argus-$(VERSION).tgz laertes:~www/htdocs/code/argus-archive/
