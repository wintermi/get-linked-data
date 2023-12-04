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
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"

	"github.com/gocolly/colly"
)

type Crawler struct {
	Collector *colly.Collector
	URL       []string
}

//---------------------------------------------------------------------------------------

// Return New Instance of a Crawler with an Embedded Colly Collector
func NewCrawler() *Crawler {
	crawler := new(Crawler)
	crawler.Collector = colly.NewCollector()
	return crawler
}

//---------------------------------------------------------------------------------------

// Load all URLs from the first column of the provided CSV File
func (crawler *Crawler) LoadUrlFile(name string, delimiter string) error {

	// Check file exists
	if _, err := os.Stat(name); err != nil {
		return fmt.Errorf("[LoadUrlFile] File Does Not Exist: %w", err)
	}
	filename, _ := filepath.Abs(name)

	// Open file ready for reading
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("[LoadUrlFile] Open File Failed: %w", err)
	}
	defer file.Close()

	// Configure CSV reader
	reader := csv.NewReader(file)
	reader.Comma = rune(delimiter[0])

	// Read all the records
	allRecords, err := reader.ReadAll()
	if err != nil {
		return fmt.Errorf("[LoadUrlFile] CSV Reader Failed: %w", err)
	}

	// Iterate through each record and retrieve the URL, or value from the
	// first column, whilst ensuring to deduplicate the final URL list
	bucket := make(map[string]bool)
	for _, value := range allRecords {
		// Only process if the record contains at least one column
		if len(value) > 0 {
			url := value[0]
			if _, ok := bucket[url]; !ok {
				bucket[url] = true
				crawler.URL = append(crawler.URL, url)
			}
		}
	}

	return nil
}

//---------------------------------------------------------------------------------------

// Deduplicate the list of URLs
func (crawler *Crawler) DeduplicateURLs() error {

	// Define a hash map and deduped array list
	bucket := make(map[string]bool)
	var deduped []string

	// Iterate through the URL list and remove duplicates
	for _, url := range crawler.URL {
		if _, ok := bucket[url]; !ok {
			bucket[url] = true
			deduped = append(deduped, url)
		}
	}

	// Replace the Crawler URL list with the deduped list
	crawler.URL = deduped

	return nil
}

//---------------------------------------------------------------------------------------

// Populate the Collector Allowed Domains
func (crawler *Crawler) SetAllowedDomains() error {
	return nil
}
