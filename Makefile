APP = zabbix_agent_bench
APPVER = 0.1.0
ARCH = $(shell uname -i)

GO = go
GFLAGS = -x
RM = rm -f
FPM = fpm
TAR = tar

all: $(APP)

$(APP): main.go zabbix_get.go
	$(GO) build $(GFLAGS)

clean:
	$(GO) clean
	$(RM) -f $(APP).tar.gz

tar: zabbix_agent_bench README.md keys/
	$(TAR) -czf $(APP)-$(APPVER).$(ARCH).tar.gz $(APP) README.md keys/

rpm: zabbix_agent_bench
	$(FPM) -f -s dir -t rpm -n $(APP) -v $(APPVER) $(APP)=/usr/bin/$(APP)
