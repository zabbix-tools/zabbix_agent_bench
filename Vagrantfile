# -*- mode: ruby -*-
# vi: set ft=ruby :

script = <<script
#!/bin/bash
BULLET="==>"

ZBX_MAJ=2
ZBX_MIN=4
ZBX_PATCH=4
ZBX_REL=1
ZBX_VER="${ZBX_MAJ}.${ZBX_MIN}.${ZBX_PATCH}-${ZBX_REL}"

cd /vagrant

# Install Zabbix
echo -e "${BULLET} Installing Zabbix..."
rpm -qa | grep zabbix-release >/dev/null || yum localinstall -y --nogpgcheck http://repo.zabbix.com/zabbix/${ZBX_MAJ}.${ZBX_MIN}/rhel/7/x86_64/zabbix-release-${ZBX_MAJ}.${ZBX_MIN}-1.el7.noarch.rpm
yum install -y --nogpgcheck zabbix-agent zabbix-get
chkconfig zabbix-agent on || systemctl enable zabbix-agent
service zabbix-agent start || systemctl start zabbix-agent

# Install Go
echo -e "${BULLET} Installing Go..."
yum install -y --nogpgcheck git mercurial
test -f go1.4.2.linux-amd64.tar.gz || curl -sLO https://storage.googleapis.com/golang/go1.4.2.linux-amd64.tar.gz
test -d /usr/local/go || tar -xzC /usr/local -f go1.4.2.linux-amd64.tar.gz

cat > /etc/profile.d/go.sh <<EOL
export GOROOT=/usr/local/go
export PATH=\\$PATH:\\$GOROOT/bin
EOL
. /etc/profile.d/go.sh

mkdir -p /home/vagrant/gocode/{src,pkg,bin}
mkdir -p /home/vagrant/gocode/src/github.com/cavaliercoder
ln -s /vagrant /home/vagrant/gocode/src/github.com/cavaliercoder/zabbix_agent_bench

grep GOPATH /home/vagrant/.bashrc >/dev/null || cat >> /home/vagrant/.bashrc <<EOL
export GOPATH=\\$HOME/gocode
export PATH=\\$PATH:\\$HOME/gocode/bin
EOL

chown vagrant.vagrant /home/vagrant/gocode

go version

echo -e "${BULLET} All done."
script

Vagrant.configure(2) do |config|
  config.vm.box = "doe/centos-7.0"
  config.vm.provision "shell", inline: script
end
