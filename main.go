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
	"flag"
	"fmt"
	"github.com/mitchellh/colorstring"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"time"
)

const (
	APP         = "zabbix_agent_bench"
	APP_VERSION = "0.2.0"
	APP_AUTHOR  = "Ryan Armstrong <ryan@cavaliercoder.com>"

	ZBX_NOTSUPPORTED = "ZBX_NOTSUPPORTED"
)

// command args
var (
	host           string
	port           int
	timeoutMsArg   int
	staggerMsArg   int
	timeLimitArg   int
	iterationLimit int
	threadCount    int
	keyFilePath    string
	key            string
	exitErrorCount bool
	verbose        bool
	debug          bool
	version        bool
)

// agent get request timeout
var timeout time.Duration

// flag to signal all threads to stop gracefully
var stop = false

func main() {

	// Configure from command line
	flag.BoolVar(&version, "version", false, "print version")
	flag.StringVar(&host, "host", "localhost", "remote Zabbix agent host")
	flag.IntVar(&port, "port", 10050, "remote Zabbix agent TCP port")
	flag.IntVar(&timeoutMsArg, "timeout", 3000, "timeout in milliseconds for each zabbix_get request")
	flag.IntVar(&staggerMsArg, "offset", 0, "offset each thread start in milliseconds")
	flag.IntVar(&threadCount, "threads", runtime.NumCPU(), "number of test threads")
	flag.IntVar(&timeLimitArg, "timelimit", 0, "time limit in seconds")
	flag.IntVar(&iterationLimit, "iterations", 0, "maximum test iterations of each key")
	flag.StringVar(&keyFilePath, "keys", "", "read keys from file path")
	flag.StringVar(&key, "key", "", "benchmark a single agent item key")
	flag.BoolVar(&exitErrorCount, "strict", false, "exit code to include tally of unsupported items")
	flag.BoolVar(&verbose, "verbose", false, "print more output")
	flag.BoolVar(&debug, "debug", false, "print program debug messages")
	flag.Parse()

	timeout = time.Duration(timeoutMsArg) * time.Millisecond
	stagger := time.Duration(staggerMsArg) * time.Millisecond
	timeLimit := time.Duration(timeLimitArg) * time.Second

	// print version and exit
	if version {
		fmt.Printf("%s v%s\n", APP, APP_VERSION)
		os.Exit(0)
	}

	// Bind threads to each core
	runtime.GOMAXPROCS(runtime.NumCPU())

	// Create a list of keys for processing
	queuedKeys := ItemKeys{}

	// user specified a single key
	if key != "" {
		queuedKeys = append(queuedKeys, &ItemKey{key, false, false, []*ItemKey{}})
	}

	// load item keys from text file
	if keyFilePath != "" {
		keyFile, err := NewKeyFile(keyFilePath)
		PanicOn(err, "Failed to open key file")

		// expand discovery item prototypes by doing an actual agent discovery
		queuedKeys, err = keyFile.Keys.Expand(host, timeout)
		PanicOn(err, "Failed to expand discovery items")
	}

	// Make sure we have work to do
	if 0 == len(queuedKeys) {
		fmt.Fprintf(os.Stderr, "No agent item keys specified for testing\n")
		os.Exit(1)
	}

	// TODO: deduplicate the key list

	// start producer thread
	fmt.Printf("Testing %d keys with %d threads (press Ctrl-C to cancel)...\n", len(queuedKeys), threadCount)
	HandleSignals()
	statsChan := make(chan *ThreadStats)
	producer := StartProducer(queuedKeys, statsChan)

	// set time limit if set
	if 0 < timeLimit {
		timer := time.NewTimer(timeLimit)
		go func() {
			<-timer.C
			stop = true
		}()
	}
	start := time.Now()

	// fan out consumer threads to start work
	for i := 0; !stop && i < threadCount; i++ {
		// Stagger thread start
		time.Sleep(stagger)

		dprintf("Starting thread %d...\n", i+1)
		go StartConsumer(producer, statsChan)
	}

	// Fan in threads to gather stats
	totals := NewThreadStats()
	for i := 0; i < threadCount+1; i++ {
		threadStats := <-statsChan
		totals.Add(threadStats)
	}

	duration := time.Now().Sub(start)

	// Sort the key list
	keyNames := queuedKeys.SortedKeyNames()

	// Print results per key
	longestKeyName := queuedKeys.LongestKeyName()
	for _, key := range keyNames {
		keyStats := totals.KeyStats[key]

		// escape %'s in key name
		key = strings.Replace(key, "%", "%%", -1)

		// show stats
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
	} else {
		os.Exit(int(totals.ErrorCount))
	}
}

// HandleSignals starts a new goroutine to handle signals from the operating
// system and signal other goroutine to gracefully stop.
func HandleSignals() {
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
}

// StartProducer starts a goroutine which iterates through the list of queued
// agent item check keys and published them sequentially to the returned
// channel until the runtime limits are reached.
func StartProducer(keys ItemKeys, statsChan chan *ThreadStats) <-chan *ItemKey {
	c := make(chan *ItemKey)
	go func() {
		stats := ThreadStats{}
		for i := 0; !stop && (iterationLimit <= 0 || i < iterationLimit); i++ {
			for _, key := range keys {
				if stop {
					break
				}

				// send key to a consumer
				c <- key
			}

			stats.Iterations++
		}

		close(c)
		statsChan <- &stats
	}()

	return c
}

// StartConsumer consumes ItemKeys from a producer channel, queries the Zabbix
// agent for a response and submits the results to a ThreadStats channel.
func StartConsumer(producer <-chan *ItemKey, statsChan chan *ThreadStats) {
	threadStats := NewThreadStats()

	// process items as long the producer produces them
	for key := range producer {
		keyStats := threadStats.KeyStats[key.Key]

		// Get the value from Zabbix agent
		val, err := Get(host, key.Key, timeout)

		// tally stats
		if err != nil {
			threadStats.ErrorCount++
			keyStats.Error++
		} else {
			threadStats.TotalValues++
			if strings.HasPrefix(val, ErrorMessage) {
				threadStats.UnsupportedValues++
				keyStats.NotSupported++
			} else {
				keyStats.Success++
			}

			// Print response
			if verbose {
				typ := "item"
				if key.IsPrototype {
					typ = "proto"
				} else if key.IsDiscoveryRule {
					typ = "disco"
				}

				fmt.Printf("[%s] %s: %s\n", typ, key.Key, val)
			}
		}

		threadStats.KeyStats[key.Key] = keyStats
	}

	// Push stats to collector channel
	statsChan <- threadStats
}

// dprintf prints debug output if debug is enabled.
func dprintf(format string, a ...interface{}) {
	if debug {
		fmt.Fprintf(os.Stderr, format, a...)
	}
}

func hl(val int64, color string) string {
	if val > 0 {
		return fmt.Sprintf("[%s]%d[default]", color, val)
	} else {
		return fmt.Sprintf("%d", val)
	}
}
