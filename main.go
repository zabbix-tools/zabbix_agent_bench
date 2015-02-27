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
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"
)

type AgentCheck struct {
	Key             string
	IsDiscoveryRule bool
	IsPrototype     bool
	Prototypes      []*AgentCheck
}

type DiscoveryData struct {
	Data []map[string]string
}

func main() {
	var host string
	//var port int
	var timeoutMsArg int
	var staggerMsArg int
	var timeLimitArg int
	var threadCount int
	var keyFile string
	var key string

	// Configure from command line
	flag.IntVar(&timeoutMsArg, "timeout", 3000, "timeout in milliseconds for each Zabbix Get request")
	flag.IntVar(&staggerMsArg, "stagger", 300, "stagger the start of each thread by milliseconds")
	flag.IntVar(&threadCount, "threads", 3, "number of test threads")
	flag.IntVar(&timeLimitArg, "timelimit", 0, "time limit in seconds")
	flag.StringVar(&keyFile, "keys", "", "read keys from file path")
	flag.StringVar(&key, "key", "", "benchmark a single agent item key")
	flag.Parse()

	timeout := time.Duration(timeoutMsArg) * time.Millisecond
	stagger := time.Duration(staggerMsArg) * time.Microsecond
	timeLimit := time.Duration(timeLimitArg) * time.Second

	keys := []*AgentCheck{}

	if key != "" {
		keys = append(keys, &AgentCheck{key, false, false, []*AgentCheck{}})
	}

	// parse key file
	var commentPattern = regexp.MustCompile(`^\s*(#.*)?$`)
	var indentPattern = regexp.MustCompile(`^\s+`)

	if keyFile != "" {
		file, err := os.Open(keyFile)
		if err != nil {
			return
		}

		defer file.Close()

		buf := bufio.NewScanner(file)

		var lastKey *AgentCheck
		var parentKey *AgentCheck

		for buf.Scan() {
			key = buf.Text()

			if !commentPattern.MatchString(key) {
				newKey := AgentCheck{key, false, false, []*AgentCheck{}}

				// is this a child prototype item?
				if indentPattern.MatchString(key) {

					newKey.IsPrototype = true

					// Strip out indentation
					newKey.Key = indentPattern.ReplaceAllString(newKey.Key, "")

					// Make the parent a Discovery Rule
					if parentKey == nil {
						parentKey = lastKey
						parentKey.IsDiscoveryRule = true
					}

					// Append to parent
					parentKey.Prototypes = append(parentKey.Prototypes, &newKey)
				} else {
					// Have we finished processing a discovery rule and prototypes?
					if parentKey != nil {
						val, err := Get(host, parentKey.Key, timeout)
						if err != nil {
							panic(err)
						}

						data := DiscoveryData{}
						err = json.Unmarshal([]byte(val), &data)
						if err != nil {
							panic(err)
						}

						// Parse each discovered instance
						for _, instance := range data.Data {

							// Create prototypes
							for _, proto := range parentKey.Prototypes {

								// Expand macros
								for macro, val := range instance {
									newKey := strings.Replace(proto.Key, macro, val, -1)
									keys = append(keys, &AgentCheck{newKey, false, true, []*AgentCheck{}})
								}
							}
						}
					}

					// This is a normal key
					parentKey = nil
					keys = append(keys, &newKey)
				}

				lastKey = &newKey
			}
		}
	}

	// Make sure we have work to do
	if 0 == len(keys) {
		fmt.Fprintf(os.Stderr, "No agent item keys or keys file specified\n")
		os.Exit(1)
	}

	// Bootstrap threads
	var wg sync.WaitGroup
	wg.Add(threadCount)

	// go to work
	start := time.Now()
	for i := 0; i < threadCount; i++ {
		time.Sleep(stagger)

		fmt.Printf("Starting thread %d...\n", i+1)

		go func(i int) {
			defer wg.Done()

			stop := false
			for !stop {
				for _, key := range keys {
					val, err := Get(host, key.Key, timeout)
					if err != nil {
						panic(err)
					}

					typ := "item"
					if key.IsPrototype {
						typ = "proto"
					} else if key.IsDiscoveryRule {
						typ = "disco"
					}
					fmt.Printf("[%s] %s: %s\n", typ, key.Key, val)

					// See if we are out of time
					if 0 < timeLimit && time.Now().Sub(start) > timeLimit {
						stop = true
						break
					}
				}
			}

			fmt.Printf("Finished thread %d\n", i)
		}(i + 1)
	}

	wg.Wait()

	fmt.Printf("Fin.\n")
}
