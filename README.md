# Zabbix Agent Bench

A multithreaded Zabbix agent benchmarking tool with support for custom keys and
discovery item prototypes.

This tool is useful for developing custom Zabbix agent items and quickly
identifying memory or file handle leaks, concurrency problems such as race
conditions and other performance issues.

    $ zabbix_agent_bench --help
    Usage of ./zabbix_agent_bench:
      -host="localhost": remote Zabbix agent host
      -key="": benchmark a single agent item key
      -keys="": read keys from file path
      -limit=0: maximum test iterations of each key
      -port=10050: remote Zabbix agent TCP port
      -stagger=0: stagger the start of each thread by milliseconds
      -threads=3: number of test threads
      -timelimit=0: time limit in seconds
      -timeout=3000: timeout in milliseconds for each Zabbix Get request
      -verbose=false: print more output
      -version=false: print application version


## Key files

Create a list of agent item keys to test by providing a text file with one key
per line to the `-keys` argument. Whitespace and lines prefixed with `#` are
ignored as comments.

For discovery items, you can specify item prototypes immediately following a
discovery item, simply by prepending the key with a tab or space.

E.g.

    vfs.fs.discovery
        vfs.fs.size[{#FSNAME},total]
        vfs.fs.size[{#FSNAME},free]
        vfs.fs.size[{#FSNAME},used]
        vfs.fs.size[{#FSNAME},pfree]
        vfs.fs.size[{#FSNAME},pused]


## Installation

Pre-compiled binaries are available on [download on SourceForge](https://sourceforge.net/projects/zabbixagentbench/files/).

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
