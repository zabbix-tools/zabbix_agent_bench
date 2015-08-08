APP = zabbix_agent_bench
APPVER = 0.2.0
ARCH = $(shell uname -m)
TARBALL = $(APP)-$(APPVER).$(ARCH)

GO = go
GFLAGS = -x -a -v -x
RM = rm -f
FPM = fpm
TAR = tar
INSTALL = install -c

all: $(APP)

$(APP): main.go zabbix_get.go keyfile.go itemkey.go error.go stats.go
	$(GO) build $(GFLAGS) -o $(APP)

get-deps:
	$(GO) get -u github.com/mitchellh/colorstring

test:
	$(GO) test -v

install: $(APP)
	$(INSTALL) $(APP) /usr/local/bin/$(APP)

tar: $(APP) README.md keys/
	mkdir $(TARBALL)
	cp -vr $(APP) README.md keys/ $(TARBALL)/
	$(TAR) -czf $(TARBALL).tar.gz $(TARBALL)/
	rm -rf $(TARBALL)/

rpm: $(APP)
	$(FPM) -f -s dir -t rpm -n $(APP) -v $(APPVER) $(APP)=/usr/bin/$(APP)

clean:
	$(GO) clean
	$(RM) -f $(APP) $(TARBALL).tar.gz $(APP)-$(APPVER)-1.$(ARCH).rpm

.PHONY: all get-deps test install tar rpm clean
