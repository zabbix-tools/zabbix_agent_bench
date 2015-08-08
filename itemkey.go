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
	"encoding/json"
	"sort"
	"strings"
	"time"
)

// An ItemKey is a single Zabbix agent item check key
type ItemKey struct {
	Key             string
	IsDiscoveryRule bool
	IsPrototype     bool
	Prototypes      ItemKeys
}

// ItemKeys is an array of pointers to ItemKey structs
type ItemKeys []*ItemKey

// DiscoveryData is a key/val array of values returned embedded in a discovery
// rule response
type DiscoveryData []map[string]string

// DiscoveryResponse is the JSON packet returned by the Zabbix agent for
// discovery rules
type DiscoveryResponse struct {
	Data DiscoveryData
}

// LongestKeyName returns the length in characters of the longest key name
// in an array of keys.
// Used for formatting output.
func (c ItemKeys) LongestKeyName() int {
	longestKeyName := 0
	for _, key := range c {
		if len(key.Key) > longestKeyName {
			longestKeyName = len(key.Key)
		}
	}

	return longestKeyName
}

// SortedKeyNames returns the name of all keys in the this key array sorted
// alphanumerically.
func (c ItemKeys) SortedKeyNames() []string {
	keyNames := []string{}
	for _, key := range c {
		keyNames = append(keyNames, key.Key)
	}
	sort.Strings(keyNames)

	return keyNames
}

// Discover sends a 'get' request to a Zabbix agent and expand the key's
// discovery prototypes into new standard keys using the response from the
// Zabbix agent
func (c *ItemKey) Discover(host string, timeout time.Duration) (ItemKeys, error) {
	if !c.IsDiscoveryRule {
		return nil, NewError(nil, "Item is not a discovery rule: %s", c.Key)
	}

	// get discovery items to expand prototypes
	dprintf("Executing discovery rule: %s\n", c.Key)
	val, err := Get(host, c.Key, timeout)
	if err != nil {
		return nil, NewError(err, "Failed to get discovery data for item: %s", c.Key)
	}

	// check if result is unsupported
	if strings.HasPrefix(val, ZBX_NOTSUPPORTED) {
		return nil, NewError(nil, "Discovery rule unsupported for item: %s", c.Key)
	}

	// bind JSON discovery data
	response := DiscoveryResponse{}
	err = json.Unmarshal([]byte(val), &response)
	if err != nil {
		return nil, NewError(err, "Failed to parse discovery json data for item: %s\n%s", c.Key, val)
	}

	// Parse each discovered instance
	keys := ItemKeys{}
	for _, instance := range response.Data {

		// Create prototypes
		for _, proto := range c.Prototypes {

			// Expand macros
			newKey := proto.Key
			for macro, val := range instance {
				newKey = strings.Replace(newKey, macro, val, -1)
			}

			dprintf("Added discovered key: %s\n", newKey)

			// Item discovered item
			keys = append(keys, &ItemKey{newKey, false, true, []*ItemKey{}})
		}
	}

	return keys, nil
}

// Expand executes a discovery on all discovery rules in a list of keys and
// appends the expanded prototypes to the returned array
func (c ItemKeys) Expand(host string, timeout time.Duration) (ItemKeys, error) {
	keys := c
	for _, key := range c {
		if key.IsDiscoveryRule {
			discoveredKeys, err := key.Discover(host, timeout)
			if err != nil {
				return nil, NewError(err, "Failed to expand prototypes for discovery rule: %s", key.Key)
			}

			keys = append(keys, discoveredKeys...)
		}
	}

	return keys, nil
}
