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
	"bufio"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gocolly/colly"
	"github.com/itchyny/gojq"
	"github.com/weppos/publicsuffix-go/publicsuffix"
)

type Crawler struct {
	Collector       *colly.Collector
	elementSelector string
	jqSelector      string
	URL             []string
	ScrapedData     []string
}

//---------------------------------------------------------------------------------------

// Return New Instance of a Crawler with an Embedded Colly Collector
func NewCrawler(elementSelector string, jqSelector string) *Crawler {

	// Initialise New Crawler
	crawler := new(Crawler)
	crawler.Collector = colly.NewCollector(
		colly.UserAgent("Mozilla/5.0 (X11; Linux x86_64; rv:109.0) Gecko/20100101 Firefox/120.0"),
		colly.MaxDepth(5),
	)
	crawler.elementSelector = elementSelector
	crawler.jqSelector = jqSelector

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
func (crawler *Crawler) ExecuteScrape(scrapeXML bool, waitTime int64) error {

	// Initialise Scraped Data Output
	crawler.ScrapedData = make([]string, 0)

	logger.Info().Msgf("%s Colly Collection Started", indent)

	// Allow for the modification of the request headers
	crawler.Collector.OnRequest(func(request *colly.Request) {
		request.Headers.Set("Accept-Encoding", "gzip")
	})

	// Scrape XML or HTML
	if scrapeXML {
		// Define the OnXML Selector Callback Function
		crawler.Collector.OnXML(crawler.elementSelector, func(element *colly.XMLElement) {
			crawler.ScrapedData = append(crawler.ScrapedData, element.Text)
		})
	} else {
		// Define the OnHTML Selector Callback Function
		crawler.Collector.OnHTML(crawler.elementSelector, func(element *colly.HTMLElement) {

			// Execute the jq Selector
			textSelected, err := jqSelect(element.Text, crawler.jqSelector)
			if err != nil {
				logger.Error().Err(fmt.Errorf("jq Selector Failed: %w", err)).Msg(doubleIndent)
				return
			}

			crawler.ScrapedData = append(crawler.ScrapedData, textSelected)
		})
	}

	// If errror occurred during the request, handle it!
	crawler.Collector.OnError(func(r *colly.Response, err error) {
		logger.Error().Err(fmt.Errorf("Colly Collector Visit Failed: %w", err)).Msg(doubleIndent)
		logger.Debug().Any("response", r).Msg(doubleIndent)
	})

	// Iterate through the URL and send the Collector for a Visit
	for _, url := range crawler.URL {
		logger.Info().Str("visiting", url).Msg(doubleIndent)
		_ = crawler.Collector.Visit(url)
		time.Sleep(time.Millisecond * time.Duration(waitTime))
	}

	logger.Info().Msgf("%s Colly Collection Finished", indent)

	return nil
}

//---------------------------------------------------------------------------------------

// Execute Scraping of URLs
func (crawler *Crawler) WriteFile(name string, delimiter string) error {

	logger.Info().Msgf("%s Writing Data to the Output File", indent)

	// Open file ready for writing
	file, err := os.Create(name)
	if err != nil {
		return fmt.Errorf("[WriteFile] Open File Failed: %w", err)
	}
	defer file.Close()

	// Ready the CSV Writer and use a buffered io writer
	w := csv.NewWriter(bufio.NewWriter(file))
	w.Comma = rune(delimiter[0])
	defer w.Flush()

	// Iterate through the Scraped Data and Write to file
	for _, data := range crawler.ScrapedData {

		var row []string = make([]string, 1)
		row[0] = strings.Replace(data, "\n", "", -1)

		if err := w.Write(row); err != nil {
			return fmt.Errorf("[WriteFile] Failed Writing to the File: %w", err)
		}
	}

	return nil
}

//---------------------------------------------------------------------------------------

// Execute Scraping of URL
func jqSelect(elementText string, query string) (string, error) {

	// If the JSON Selector Query was NOT provided then return the element text
	if query == "" {
		return elementText, nil
	}

	// Convert the element text to a JSON Object before querying
	var jsonData map[string]any
	if err := json.Unmarshal([]byte(elementText), &jsonData); err != nil {
		return "", fmt.Errorf("Selected Element Text is not a valid JSON Object: %w", err)
	}

	// Parse the provided jq selector text
	jq, err := gojq.Parse(query)
	if err != nil {
		return "", fmt.Errorf("jq Selector Parse Failed: %w", err)
	}

	// Execute the jq Selector against the element text only returning the first value
	jqSelector := jq.Run(jsonData)
	val, ok := jqSelector.Next()
	if !ok {
		return "", errors.New("jq Selector Failed to Find First Value")
	}

	// Check if the first value returned is actually an error
	if err, ok := val.(error); ok {
		return "", fmt.Errorf("jq Selector Run Failed: %w", err)
	}

	// Convert the first value returned to a raw JSON string and return
	rawJSON, _ := json.Marshal(val)
	return string(rawJSON), nil
}
