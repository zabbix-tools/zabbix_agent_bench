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
package main

import (
	"bytes"
	"fmt"
	"github.com/mitchellh/colorstring"
	"os"
)

type Error struct {
	Message    string
	InnerError error
}

func NewError(err error, format string, a ...interface{}) error {
	return &Error{
		Message:    fmt.Sprintf(format, a...),
		InnerError: err,
	}
}

func (c *Error) Error() string {
	buffer := bytes.NewBufferString(c.Message)

	indent := 0
	next := c.InnerError
	for next != nil {
		buffer.WriteString("\n")

		indent++
		for i := 0; i < indent; i++ {
			buffer.WriteString(" ")
		}

		buffer.WriteString("-> ")

		if nerr, ok := next.(*Error); ok {
			buffer.WriteString(nerr.Message)
			next = nerr.InnerError
		} else {
			buffer.WriteString(next.Error())
			next = nil
		}
	}

	return buffer.String()
}

func PrintError(err error) {
	colorstring.Fprintf(os.Stderr, "[red]Error:[default] %s\n", err.Error())
}

func PanicOn(err error, format string, a ...interface{}) {
	if err != nil {
		PrintError(NewError(err, format, a...))
		os.Exit(1)
	}
}
