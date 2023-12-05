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
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/gocolly/colly"
	"github.com/weppos/publicsuffix-go/publicsuffix"
)

type Crawler struct {
	Collector   *colly.Collector
	Selector    string
	URL         []string
	ScrapedData []string
}

//---------------------------------------------------------------------------------------

// Return New Instance of a Crawler with an Embedded Colly Collector
func NewCrawler(selector string) *Crawler {

	// Initialise New Crawler
	crawler := new(Crawler)
	crawler.Collector = colly.NewCollector()
	crawler.Selector = selector

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

	// Define a hash map and domain array list
	bucket := make(map[string]bool)
	var allowedDomains []string

	logger.Info().Msgf("%s Allowed Domain List", indent)

	// Iterate through the URL list and create a deduped domain list
	for _, rawURL := range crawler.URL {
		// Parse URL and trieve the hostname
		u, err := url.Parse(rawURL)
		if err != nil {
			return fmt.Errorf("[SetAllowedDomains] URL Parse Failed: %w", err)
		}
		hostname := u.Hostname()

		// Parse the domain name from the hostname
		domain, err := publicsuffix.Domain(hostname)
		if err != nil {
			return fmt.Errorf("[SetAllowedDomains] Domain Parse Failed: %w", err)
		}

		// Add domain name, e.g. google.com
		if _, ok := bucket[domain]; !ok {
			bucket[domain] = true
			allowedDomains = append(allowedDomains, domain)
			logger.Info().Str("allowed", domain).Msg(doubleIndent)
		}

		// Add hostname, e.g. www.google.com
		if _, ok := bucket[hostname]; !ok {
			bucket[hostname] = true
			allowedDomains = append(allowedDomains, hostname)
			logger.Info().Str("allowed", hostname).Msg(doubleIndent)
		}
	}

	// Set the Collector Allowed Domain List
	crawler.Collector.AllowedDomains = allowedDomains

	return nil
}

//---------------------------------------------------------------------------------------

// Execute Scraping of URLs
func (crawler *Crawler) ExecuteScrape() error {

	// Initialise Scraped Data Output
	crawler.ScrapedData = make([]string, 0)

	logger.Info().Msgf("%s Colly Collection Started", indent)

	// Define the Selector Callback Function
	crawler.Collector.OnHTML(crawler.Selector, func(element *colly.HTMLElement) {
		crawler.ScrapedData = append(crawler.ScrapedData, element.Text)
		logger.Info().Msgf("%s    - %v", doubleIndent, element)
	})

	// Iterate through the URL and send the Collector for a Visit
	for _, url := range crawler.URL {
		logger.Info().Str("visiting", url).Msg(doubleIndent)
		if err := crawler.Collector.Visit(url); err != nil {
			return fmt.Errorf("[ExecuteScrape] Colly Collector Visit Failed: %w", err)
		}
		time.Sleep(time.Millisecond * time.Duration(100))
	}

	logger.Info().Msgf("%s Colly Collection Finished", indent)

	return nil
}
