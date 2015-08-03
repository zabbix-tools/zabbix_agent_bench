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
	"github.com/mitchellh/colorstring"
	"os"
	"os/signal"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"time"
)

const (
	APP         = "zabbix_agent_bench"
	APP_VERSION = "0.1.0"
	APP_AUTHOR  = "Ryan Armstrong <ryan@cavaliercoder.com>"

	ZBX_NOTSUPPORTED = "ZBX_NOTSUPPORTED"
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

type KeyTally map[string]struct {
	Success      int64
	NotSupported int64
	Error        int64
}

type ThreadStats struct {
	Duration          time.Duration
	Iterations        int64
	TotalValues       int64
	UnsupportedValues int64
	ErrorCount        int64
	Keys              KeyTally
}

func main() {
	var host string
	var port int
	var timeoutMsArg int
	var staggerMsArg int
	var timeLimitArg int
	var iterationLimit int
	var threadCount int
	var keyFile string
	var key string
	var exitErrorCount bool
	var verbose, version bool

	// Configure from command line
	flag.BoolVar(&version, "version", false, "print application version")
	flag.StringVar(&host, "host", "localhost", "remote Zabbix agent host")
	flag.IntVar(&port, "port", 10050, "remote Zabbix agent TCP port")
	flag.IntVar(&timeoutMsArg, "timeout", 3000, "timeout in milliseconds for each Zabbix Get request")
	flag.IntVar(&staggerMsArg, "stagger", 0, "stagger the start of each thread by milliseconds")
	flag.IntVar(&threadCount, "threads", 3, "number of test threads")
	flag.IntVar(&timeLimitArg, "timelimit", 0, "time limit in seconds")
	flag.IntVar(&iterationLimit, "limit", 0, "maximum test iterations of each key per thread")
	flag.StringVar(&keyFile, "keys", "", "read keys from file path")
	flag.StringVar(&key, "key", "", "benchmark a single agent item key")
	flag.BoolVar(&exitErrorCount, "errorcount", false, "set exit code to the sum of unsupported and failed items")
	flag.BoolVar(&verbose, "verbose", false, "print more output")
	flag.Parse()

	timeout := time.Duration(timeoutMsArg) * time.Millisecond
	stagger := time.Duration(staggerMsArg) * time.Millisecond
	timeLimit := time.Duration(timeLimitArg) * time.Second

	if version {
		fmt.Printf("%s v%s\n", APP, APP_VERSION)
		os.Exit(0)
	}

	// Bind threads to each core
	runtime.GOMAXPROCS(runtime.NumCPU())

	// Create a list of keys for processing
	keys := []*AgentCheck{}
	if key != "" {
		keys = append(keys, &AgentCheck{key, false, false, []*AgentCheck{}})
	}

	// Load item keys from text file
	if keyFile != "" {
		var commentPattern = regexp.MustCompile(`^\s*(#.*)?$`)
		var indentPattern = regexp.MustCompile(`^\s+`)

		// Open key file
		file, err := os.Open(keyFile)
		if err != nil {
			return
		}
		defer file.Close()

		var lastKey *AgentCheck
		var parentKey *AgentCheck

		// Read one key per line
		buf := bufio.NewScanner(file)
		for buf.Scan() {
			key = buf.Text()

			// Ignore blanks lines and comments
			if !commentPattern.MatchString(key) {
				newKey := AgentCheck{key, false, false, []*AgentCheck{}}

				// is this a child prototype item?
				if indentPattern.MatchString(key) {
					newKey.IsPrototype = true

					// Strip out indentation
					newKey.Key = indentPattern.ReplaceAllString(newKey.Key, "")

					// Make the parent a Discovery Rule if not already
					if parentKey == nil {
						parentKey = lastKey
						parentKey.IsDiscoveryRule = true
					}

					// Append to parent
					parentKey.Prototypes = append(parentKey.Prototypes, &newKey)
				} else {
					// This is a normal key
					parentKey = nil
					keys = append(keys, &newKey)
				}

				lastKey = &newKey
			}
		}

		// expand discovery item prototypes by doing an actual agent discovery
		for _, parentKey := range keys {
			if parentKey.IsDiscoveryRule {
				// get discovery items to expand prototypes
				val, err := Get(host, parentKey.Key, timeout)
				DoOrDie(err)

				if strings.HasPrefix(val, ZBX_NOTSUPPORTED) {
					Errorf("Discovery item unsupported: %s", parentKey.Key)
					continue
				}

				// bind JSON discovery data
				data := DiscoveryData{}
				err = json.Unmarshal([]byte(val), &data)
				DoOrDie(err, val)

				// Parse each discovered instance
				for _, instance := range data.Data {

					// Create prototypes
					for _, proto := range parentKey.Prototypes {

						// Expand macros
						newKey := proto.Key
						for macro, val := range instance {
							newKey = strings.Replace(newKey, macro, val, -1)
						}

						// Item discovered item
						keys = append(keys, &AgentCheck{newKey, false, true, []*AgentCheck{}})
					}
				}
			}
		}
	}

	// Make sure we have work to do
	if 0 == len(keys) {
		fmt.Fprintf(os.Stderr, "No agent item keys specified for testing\n")
		os.Exit(1)
	}

	// Find longest key name
	longestKeyName := 0
	for _, key := range keys {
		if len(key.Key) > longestKeyName {
			longestKeyName = len(key.Key)
		}
	}

	// Capture Ctrl+C SIGINTs
	stop := false
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for {
			<-c // Wait for signal

			if stop {
				// Force exit if user sent SIGINT during cleanup
				fmt.Printf("Aborting...\n")
				os.Exit(1)
			} else {
				fmt.Printf("Caught SIGINT. Cleaning up...\n")
				stop = true
			}
		}
	}()

	// Start operation timer
	if 0 < timeLimit {
		timer := time.NewTimer(timeLimit)
		go func() {
			<-timer.C
			stop = true
		}()
	}

	// go to work
	fmt.Printf("Testing %d keys across %d threads...\n", len(keys), threadCount)
	start := time.Now()
	statsChan := make(chan ThreadStats)
	for i := 0; !stop && i < threadCount; i++ {

		// Stagger thread start
		time.Sleep(stagger)

		// fmt.Printf("Starting thread %d...\n", i+1)
		go func(i int, stats chan ThreadStats) {
			threadStats := ThreadStats{}
			threadStats.Keys = make(KeyTally)

			for !stop {
				// Iterate over each key in the list
				for _, key := range keys {
					if stop {
						break
					}

					typ := "item"
					if key.IsPrototype {
						typ = "proto"
					} else if key.IsDiscoveryRule {
						typ = "disco"
					}

					keyStats := threadStats.Keys[key.Key]

					// Get the value from Zabbix agent
					val, err := Get(host, key.Key, timeout)
					if err != nil {
						// Transport error getting valuw
						// colorstring.Fprintf(os.StdErr, "[red][%s][default] %s: %s\n"), typ, key.Key, err.Error())
						threadStats.ErrorCount++
						keyStats.Error++
					} else {
						// Print response
						if verbose {
							fmt.Printf("[%s] %s: %s\n", typ, key.Key, val)
						}

						// Tally results
						threadStats.TotalValues++
						if strings.HasPrefix(val, ErrorMessage) {
							threadStats.UnsupportedValues++
							keyStats.NotSupported++
						} else {
							keyStats.Success++
						}
					}

					threadStats.Keys[key.Key] = keyStats
				}

				// Increment key list iteration count
				threadStats.Iterations++
				if 0 < iterationLimit && threadStats.Iterations >= int64(iterationLimit) {
					break
				}

				if stop {
					break
				}
			}

			// Push stats to collector
			statsChan <- threadStats
		}(i+1, statsChan)
	}

	// Gather stats
	totals := ThreadStats{}
	totals.Keys = make(KeyTally)
	for i := 0; i < threadCount; i++ {
		threadStats := <-statsChan
		totals.Iterations += threadStats.Iterations
		totals.TotalValues += threadStats.TotalValues
		totals.UnsupportedValues += threadStats.UnsupportedValues
		totals.ErrorCount += threadStats.ErrorCount

		for key, keyStats := range threadStats.Keys {
			tKeyStats := totals.Keys[key]
			tKeyStats.Success += keyStats.Success
			tKeyStats.NotSupported += keyStats.NotSupported
			tKeyStats.Error += keyStats.Error

			totals.Keys[key] = tKeyStats
		}
	}
	duration := time.Now().Sub(start)

	// Sort the key list
	keyNames := []string{}
	for _, key := range keys {
		keyNames = append(keyNames, key.Key)
	}
	sort.Strings(keyNames)

	// Print results per key
	for _, key := range keyNames {
		keyStats := totals.Keys[key]
		row := fmt.Sprintf("%-*s :\t%s\t%s\t%s\n", longestKeyName, key, hl(keyStats.Success, "green"), hl(keyStats.NotSupported, "yellow"), hl(keyStats.Error, "red"))
		colorstring.Printf(row)
	}

	// Print totals
	fmt.Printf("\n=== Totals ===\n\n")
	fmt.Printf("Total values processed:\t\t%d\n", totals.TotalValues)
	fmt.Printf("Total unsupported values:\t%d\n", totals.UnsupportedValues)
	fmt.Printf("Total transport errors:\t\t%d\n", totals.ErrorCount)
	fmt.Printf("Total key list iterations:\t%d\n", totals.Iterations)

	colorstring.Printf("\n[green]Finished![default] Processed %d values across %d threads in %s (%f NVPS)\n", totals.TotalValues, threadCount, duration.String(), (float64(totals.TotalValues) / duration.Seconds()))

	// exit code
	if exitErrorCount {
		os.Exit(int(totals.UnsupportedValues + totals.ErrorCount))
	}
}

func Errorf(format string, a ...interface{}) {
	colorstring.Fprintf(os.Stderr, "[red]Error:[default] %s\n", fmt.Sprintf(format, a...))
}

func DoOrDie(err error, v ...interface{}) {
	if err != nil {
		Errorf(err.Error())
		for _, x := range v {
			colorstring.Fprintf(os.Stderr, "%#v\n", x)
		}
		os.Exit(1)
	}
}

func hl(val int64, color string) string {
	if val > 0 {
		return fmt.Sprintf("[%s]%d[default]", color, val)
	} else {
		return fmt.Sprintf("%d", val)
	}
}
