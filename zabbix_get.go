/*
 * Zabbix Agent Bench (C) 2014  Ryan Armstrong <ryan@cavaliercoder.com>
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

/*
 * This file courtesy: https://github.com/fujiwara/go-zabbix-get
 */

package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"time"
)

const (
	AgentDefaultPort = 10050
	HeaderString     = "ZBXD"
	HeaderLength     = len(HeaderString)
	HeaderVersion    = uint8(1)
	DataLengthOffset = int64(HeaderLength + 1)
	DataLengthSize   = int64(8)
	DataOffset       = int64(DataLengthOffset + DataLengthSize)
	ErrorMessage     = "ZBX_NOTSUPPORTED"
)

var (
	ErrorMessageBytes = []byte(ErrorMessage)
	Terminator        = []byte("\n")
	HeaderBytes       = []byte(HeaderString)
)

func Get(addr string, key string, timeout time.Duration) (value string, err error) {
	// Append port specifier to socket address
	_, _, err = net.SplitHostPort(addr)
	if err != nil {
		addr = fmt.Sprintf("%s:%d", addr, AgentDefaultPort)
	}

	// Connect via TCP
	conn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return
	}
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(timeout))

	// Build the request
	buf := new(bytes.Buffer)
	buf.Write(HeaderBytes)
	binary.Write(buf, binary.LittleEndian, HeaderVersion)
	binary.Write(buf, binary.LittleEndian, int64(len([]byte(key))))
	buf.Write([]byte(key))

	// Send the request
	_, err = conn.Write(buf.Bytes())
	if err != nil {
		return
	}

	// read header "ZBXD\x01"
	head := make([]byte, DataLengthOffset)
	_, err = conn.Read(head)
	if err != nil {
		return
	}

	val, err := parseBinary(conn)

	return string(val), err
}

func parseBinary(conn io.Reader) (rdata []byte, err error) {
	// read data length
	var dataLength int64
	err = binary.Read(conn, binary.LittleEndian, &dataLength)
	if err != nil {
		return
	}

	// read data body
	buf := make([]byte, 1024)
	data := new(bytes.Buffer)
	total := 0
	size := 0
	for total < int(dataLength) {
		size, err = conn.Read(buf)
		if err != nil {
			return
		}
		if size == 0 {
			break
		}
		total = total + size
		data.Write(buf[0:size])
	}
	rdata = data.Bytes()
	return
}
