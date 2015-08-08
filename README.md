# Zabbix Agent Bench [![Build Status](https://travis-ci.org/cavaliercoder/zabbix_agent_bench.svg?branch=master)](https://travis-ci.org/cavaliercoder/zabbix_agent_bench) [![Download zabbix_agent_bench](https://img.shields.io/sourceforge/dm/zabbixagentbench.svg)](https://sourceforge.net/projects/zabbixagentbench/files/)

A multithreaded Zabbix agent benchmarking tool with support for custom keys and
discovery item prototypes.

This tool is useful for developing custom Zabbix agent items and quickly
identifying memory or file handle leaks, concurrency problems such as race
conditions and other performance issues.

## Usage

    $ zabbix_agent_bench --help
    Usage of ./zabbix_agent_bench:
      -debug=false: print program debug messages
      -host="localhost": remote Zabbix agent host
      -iterations=0: maximum test iterations of each key
      -key="": benchmark a single agent item key
      -keys="": read keys from file path
      -offset=0: offset each thread start in milliseconds
      -port=10050: remote Zabbix agent TCP port
      -strict=false: exit code to include tally of unsupported items
      -threads=1: number of test threads
      -timelimit=0: time limit in seconds
      -timeout=3000: timeout in milliseconds for each zabbix_get request
      -verbose=false: print more output
      -version=false: print version

Test a single key until cancelled with `Ctrl-C`:

    $ zabbix_agent_bench -key agent.version

Test a list of keys (including discovery rules and prototypes):

    $ zabbix_agent_bench -keys linux_keys.conf

Simple unit-test style check of a list of keys:

    $ zabbix_agent_bench -keys linux_keys.conf -iterations 1 -strict


## Key files

You can test multiple keys by creating a text file with one key per line. You
may then pass this file to the `-keys` argument.

To create a discovery rule, you may specify item prototypes immediately
following an item definition, simply by prepending the prototype key with a tab
or space.

E.g.

    vfs.fs.discovery
        vfs.fs.size[{#FSNAME},total]
        vfs.fs.size[{#FSNAME},free]
        vfs.fs.size[{#FSNAME},used]
        vfs.fs.size[{#FSNAME},pfree]
        vfs.fs.size[{#FSNAME},pused]

Whitespace and lines prefixed with `#` are ignored as comments.


## Installation

Pre-compiled binaries and packages are available for
[download on SourceForge](https://sourceforge.net/projects/zabbixagentbench/files/).

Alternatively, you can build the project yourself in Go. Once you have a
working [installation of Go](https://golang.org/doc/install), simply run:

    go get github.com/cavaliercoder/zabbix_agent_bench


## License

Zabbix Agent Bench Copyright (C) 2014 Ryan Armstrong (ryan@cavaliercoder.com)

This program is free software: you can redistribute it and/or modify it under
the terms of the GNU General Public License as published by the Free Software
Foundation, either version 3 of the License, or (at your option) any later
version.

This program is distributed in the hope that it will be useful, but WITHOUT ANY
WARRANTY; without even the implied warranty of MERCHANTABILITY or FITNESS FOR A
PARTICULAR PURPOSE. See the GNU General Public License for more details.

You should have received a copy of the GNU General Public License along with
this program. If not, see http://www.gnu.org/licenses/.
