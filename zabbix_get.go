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
	"errors"
	"fmt"
	"io"
	"net"
	"time"
)

const (
	AgentDefaultPort  = 10050
	ServerDefaultPort = 10051
	HeaderString      = "ZBXD"
	HeaderLength      = len(HeaderString)
	HeaderVersion     = uint8(1)
	DataLengthOffset  = int64(HeaderLength + 1)
	DataLengthSize    = int64(8)
	DataOffset        = int64(DataLengthOffset + DataLengthSize)
	ErrorMessage      = "ZBX_NOTSUPPORTED"
)

var (
	ErrorMessageBytes = []byte(ErrorMessage)
	Terminator        = []byte("\n")
	HeaderBytes       = []byte(HeaderString)
)

func FillDefaultPort(addr string, port int) string {
	_, _, err := net.SplitHostPort(addr)
	if err != nil {
		return fmt.Sprintf("%s:%d", addr, port)
	}
	return addr
}

func Get(addr string, key string, timeout time.Duration) (value string, err error) {
	addr = FillDefaultPort(addr, AgentDefaultPort)
	conn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return
	}
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(timeout))

	msg := Data2Packet([]byte(key))
	_, err = conn.Write(msg)
	if err != nil {
		return
	}
	_value, err := Stream2Data(conn)
	return string(_value), err
}

func Data2Packet(data []byte) []byte {
	buf := new(bytes.Buffer)
	Data2Stream(data, buf)
	return buf.Bytes()
}

func Data2Stream(data []byte, conn io.Writer) (int, error) {
	conn.Write(HeaderBytes)
	binary.Write(conn, binary.LittleEndian, HeaderVersion)
	binary.Write(conn, binary.LittleEndian, int64(len(data)))
	return conn.Write(data)
}

func Packet2Data(packet []byte) (data []byte, err error) {
	var dataLength int64
	if len(packet) < int(DataOffset) {
		err = errors.New("zabbix protocol packet too short")
		return
	}

	// read header
	headBuf := bytes.NewReader(packet[0:DataLengthOffset])
	head := make([]byte, DataLengthOffset)
	_, err = headBuf.Read(head)
	if !bytes.Equal(head[0:HeaderLength], HeaderBytes) || head[HeaderLength] != byte(HeaderVersion) {
		err = errors.New("invalid packet header")
		return
	}

	// read data
	buf := bytes.NewReader(packet[DataLengthOffset:DataOffset])
	err = binary.Read(buf, binary.LittleEndian, &dataLength)
	if err != nil {
		return
	}
	data = packet[DataOffset : DataOffset+dataLength]
	return
}

func Stream2Data(conn io.Reader) (rdata []byte, err error) {
	// read header "ZBXD\x01"
	head := make([]byte, DataLengthOffset)
	_, err = conn.Read(head)
	if err != nil {
		return
	}
	if bytes.Equal(head[0:HeaderLength], HeaderBytes) && head[HeaderLength] == byte(HeaderVersion) {
		rdata, err = parseBinary(conn)
	} else {
		rdata, err = parseText(conn, head)
	}
	return
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

func parseText(conn io.Reader, head []byte) (rdata []byte, err error) {
	data := new(bytes.Buffer)
	data.Write(head)
	buf := make([]byte, 1024)
	size := 0
	for {
		// read data while "\n" found
		size, err = conn.Read(buf)
		if err != nil {
			return
		}
		if size == 0 {
			break
		}
		i := bytes.Index(buf[0:size], Terminator)
		if i == -1 {
			// terminator not found
			data.Write(buf[0:size])
			continue
		}
		// terminator found
		data.Write(buf[0 : i+1])
		break
	}
	rdata = data.Bytes()
	return
}
