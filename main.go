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
var applicationText = "%s 0.1.0%s"
var copyrightText = "Copyright 2023-2024, Matthew Winter\n"
var indent = "..."

var helpText = `
A command line application designed to crawl a given set of URLs and scrape
the JSON Linked Data (JSON-LD) contained within the webpage before writing the
data entries out to a CSV file.

Use --help for more details.


USAGE:
    get-linked-data -i URL_CSV -e ELEMENT_SELECTOR -o OUTPUT_CSV

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
	var elementSelector = flag.String("e", "", "Element Selector  (Required)")
	var outputCsvFile = flag.String("o", "", "Output CSV File  (Required)")
	var fieldDelimiter = flag.String("d", ",", "Field Delimiter  (Required)")
	var crawlURLs = flag.Bool("c", false, "Crawl URLs")
	var verbose = flag.Bool("v", false, "Output Verbose Detail")

	// Parse the flags
	flag.Parse()

	// Validate the Required Flags
	if *inputCsvFile == "" || *elementSelector == "" || *outputCsvFile == "" {
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
	logger.Info().Str("Output CSV File", *outputCsvFile).Msg(indent)
	logger.Info().Str("Field Delimiter", *fieldDelimiter).Msg(indent)
	logger.Info().Bool("Crawl URLs", *crawlURLs).Msg(indent)
	logger.Info().Msg("Begin")

	// Load the URLs into memory ready to crawl & scrape the Linked Data
	var crawler = NewCrawler()
	err := crawler.LoadUrlFile(*inputCsvFile, *fieldDelimiter)
	if err != nil {
		logger.Error().Err(err).Msg("Failed Loading Queries")
		os.Exit(1)
	}

	logger.Info().Msg("Done!")
}
