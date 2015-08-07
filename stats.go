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
	"time"
)

type KeyStats struct {
	Success      int64
	NotSupported int64
	Error        int64
}

// ThreadStats represents the sum statistics for all item keys gathered from a
// Zabbix agent by a single goroutine.
type ThreadStats struct {
	Duration          time.Duration
	Iterations        int64
	TotalValues       int64
	UnsupportedValues int64
	ErrorCount        int64
	KeyStats          map[string]KeyStats
}

func NewThreadStats() *ThreadStats {
	return &ThreadStats{
		KeyStats: make(map[string]KeyStats, 0),
	}
}
