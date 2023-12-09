// Copyright 2023-2024, Matthew Winter
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/rs/zerolog"
)

var logger zerolog.Logger
var applicationText = "%s 0.3.1%s"
var copyrightText = "Copyright 2023-2024, Matthew Winter\n"
var indent = "..."
var doubleIndent = "......."

var helpText = `
A command line application designed to crawl a given set of URLs and scrape
the JSON Linked Data (JSON-LD) contained within the webpage before writing the
data entries out to a CSV file.

Use --help for more details.


USAGE:
    get-linked-data -i URL_CSV -s ELEMENT_SELECTOR -o OUTPUT_CSV -e FAILED_URL_CSV

ARGS:
`

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, applicationText, filepath.Base(os.Args[0]), "\n")
		fmt.Fprint(os.Stderr, copyrightText)
		fmt.Fprint(os.Stderr, helpText)
		flag.PrintDefaults()
	}

	// Define the Long CLI flag names
	var inputCsvFile = flag.String("i", "", "CSV File containing URLs to Scrape  (Required)")
	var elementSelector = flag.String("s", "", "Element Selector  (Required)")
	var jqSelector = flag.String("j", "", "jq Selector")
	var outputCsvFile = flag.String("o", "", "Output Scraped Data CSV File  (Required)")
	var errorCsvFile = flag.String("e", "", "Failed Request URLs Output CSV File  (Required)")
	var fieldDelimiter = flag.String("d", ",", "Field Delimiter  (Required)")
	var parallelism = flag.Int("p", 100, "Parallelism or Maximum allowed Concurrent Requests")
	var waitTime = flag.Int("w", 2000, "Random Wait Time in Milliseconds between Requests")
	var scrapeXML = flag.Bool("x", false, "Scrape XML not HTML")
	var verbose = flag.Bool("v", false, "Output Verbose Detail")

	// Parse the flags
	flag.Parse()

	// Validate the Required Flags
	if *inputCsvFile == "" || *elementSelector == "" || *outputCsvFile == "" || *errorCsvFile == "" {
		flag.Usage()
		os.Exit(1)
	}

	// Validate that the Field Delimiter is 1 character
	if len(*fieldDelimiter) != 1 {
		flag.Usage()
		os.Exit(1)
	}

	// Setup Zero Log for Consolo Output
	output := zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339}
	logger = zerolog.New(output).With().Timestamp().Logger()
	zerolog.TimeFieldFormat = "2006-01-02 15:04:05.000"
	zerolog.DurationFieldUnit = time.Millisecond
	zerolog.DurationFieldInteger = true
	if *verbose {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	} else {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}

	// Output Header
	logger.Info().Msgf(applicationText, filepath.Base(os.Args[0]), "")
	logger.Info().Msg("Arguments")
	logger.Info().Str("CSV File containing URLs to Scrape", *inputCsvFile).Msg(indent)
	logger.Info().Str("Element Selector", *elementSelector).Msg(indent)
	logger.Info().Str("jq Selector", *jqSelector).Msg(indent)
	logger.Info().Str("Output Scraped Data CSV File", *outputCsvFile).Msg(indent)
	logger.Info().Str("Failed Request URLs Output CSV File", *errorCsvFile).Msg(indent)
	logger.Info().Str("Field Delimiter", *fieldDelimiter).Msg(indent)
	logger.Info().Int("Parallelism or Maximum allowed Concurrent Requests", *parallelism).Msg(indent)
	logger.Info().Int("Random Wait Time in Milliseconds between Requests", *waitTime).Msg(indent)
	logger.Info().Bool("Scrape XML not HTML", *scrapeXML).Msg(indent)
	logger.Info().Msg("Begin")

	// Load the URLs into memory ready for Colly to crawl & scrape the Linked Data
	var crawler = NewCrawler(*elementSelector, *jqSelector, *waitTime, *parallelism)
	if err := crawler.LoadUrlFile(*inputCsvFile, *fieldDelimiter); err != nil {
		logger.Error().Err(err).Msg("Failed Loading URL List")
		os.Exit(1)
	}

	// Set the Allowed Domain List for the Colly Collector
	if err := crawler.SetAllowedDomains(); err != nil {
		logger.Error().Err(err).Msg("Failed to Set Allowed Domain List")
		os.Exit(1)
	}

	// Shuffle the URL List, changing the order Colly scrapes them
	if err := crawler.ShuffleURLs(); err != nil {
		logger.Error().Err(err).Msg("Failed to Shuffle URL List")
		os.Exit(1)
	}

	// Execute the Colly Collector
	if err := crawler.ExecuteScrape(*scrapeXML); err != nil {
		logger.Error().Err(err).Msg("Scraping Linked Data Failed")
		os.Exit(1)
	}

	// Write the Scraped Data out to a File
	if err := crawler.WriteDataFile(*outputCsvFile, *fieldDelimiter); err != nil {
		logger.Error().Err(err).Msg("Writing Data File Failed")
		os.Exit(1)
	}

	// Write the Failed Request URLs out to a File
	if err := crawler.WriteErrorFile(*errorCsvFile, *fieldDelimiter); err != nil {
		logger.Error().Err(err).Msg("Writing Error File Failed")
		os.Exit(1)
	}

	logger.Info().Msg("Done!")
}
