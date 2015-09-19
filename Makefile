APP = zabbix_agent_bench
APPVER = 0.4.0
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
	$(GO) test -x -v

install: $(APP)
	$(INSTALL) $(APP) /usr/local/bin/$(APP)

packages: tar deb rpm

tar: $(APP) README.md keys/
	mkdir $(TARBALL)
	cp -vr $(APP) README.md keys/ $(TARBALL)/
	$(TAR) -czf $(TARBALL).tar.gz $(TARBALL)/
	rm -rf $(TARBALL)/

deb: $(APP)
	$(FPM) -f -s dir -t deb -n $(APP) -v $(APPVER) $(APP)=/usr/bin/$(APP)

rpm: $(APP)
	$(FPM) -f -s dir -t rpm -n $(APP) -v $(APPVER) $(APP)=/usr/bin/$(APP)

clean:
	$(GO) clean -i -x
	$(RM) -f $(APP) $(TARBALL).tar.gz $(APP)-$(APPVER)-1.$(ARCH).rpm zabbix-agent-bench_$(APPVER)_amd64.deb

docker-run:
	docker run -it --rm -v $(PWD):/go/src/github.com/cavaliercoder/zabbix_agent_bench -w /go/src/github.com/cavaliercoder/zabbix_agent_bench golang

.PHONY: all get-deps test install packages tar deb rpm clean docker-run

