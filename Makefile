APP = zabbix_agent_bench
APPVER = 0.1.0
ARCH = $(shell uname -i)
TARBALL = $(APP)-$(APPVER).$(ARCH)

GO = go
GFLAGS = -x -a -race -v -x
RM = rm -f
FPM = fpm
TAR = tar

all: $(APP)

$(APP): main.go zabbix_get.go keyfile.go itemkey.go error.go stats.go
	$(GO) build $(GFLAGS) -o $(APP)

get-deps:
	$(GO) get -u github.com/mitchellh/colorstring

clean:
	$(GO) clean
	$(RM) -f $(APP) $(TARBALL).tar.gz $(APP)-$(APPVER)-1.$(ARCH).rpm

tar: zabbix_agent_bench README.md keys/
	mkdir $(TARBALL)
	cp -r $(APP) README.md keys/ $(TARBALL)/
	$(TAR) -czf $(TARBALL).tar.gz $(TARBALL)/
	rm -rf $(TARBALL)/

rpm: zabbix_agent_bench
	$(FPM) -f -s dir -t rpm -n $(APP) -v $(APPVER) $(APP)=/usr/bin/$(APP)
