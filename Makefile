APP = zabbix_agent_bench
APPVER = 0.1.0

GO = go
GFLAGS = -x
RM = rm -f
FPM = fpm

all: $(APP)

$(APP):
	$(GO) $(GLFAGS) build

clean:
	$(RM) $(APP)

rpm: zabbix_agent_bench
	$(FPM) -f -s dir -t rpm -n $(APP) -v $(APPVER) $(APP)=/usr/bin/$(APP)