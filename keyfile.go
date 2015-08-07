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
	"bufio"
	"os"
	"regexp"
)

type KeyFile struct {
	Path string
	Keys ItemKeys
}

var (
	commentPattern = regexp.MustCompile(`^\s*(#.*)?$`)
	indentPattern  = regexp.MustCompile(`^\s+`)
)

// NewKeyFile loads Zabbix agent keys from a plain text file
func NewKeyFile(path string) (*KeyFile, error) {

	// Open key file
	dprintf("Loading keys from file: %s\n", path)
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var keyfile = &KeyFile{
		Path: path,
		Keys: make(ItemKeys, 0),
	}

	var (
		lastKey   *ItemKey
		parentKey *ItemKey
	)

	// Read one key per line
	buf := bufio.NewScanner(file)
	for buf.Scan() {
		line := buf.Text()

		// Ignore blanks lines and comments
		if !commentPattern.MatchString(line) {
			newKey := ItemKey{line, false, false, []*ItemKey{}}

			// is this a child prototype item?
			if indentPattern.MatchString(line) {
				// Strip out indentation
				newKey.Key = indentPattern.ReplaceAllString(newKey.Key, "")
				dprintf("Added key prototype: %s\n", newKey.Key)

				// Make the parent a Discovery Rule if not already
				newKey.IsPrototype = true
				if parentKey == nil {
					parentKey = lastKey
					parentKey.IsDiscoveryRule = true
				}

				// Append to parent
				parentKey.Prototypes = append(parentKey.Prototypes, &newKey)
			} else {
				// This is a normal key
				dprintf("Added key: %s\n", newKey.Key)
				parentKey = nil
				keyfile.Keys = append(keyfile.Keys, &newKey)
			}

			lastKey = &newKey
		}
	}

	dprintf("Finished loading key file\n")
	return keyfile, nil
}
