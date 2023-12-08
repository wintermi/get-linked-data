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
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gocolly/colly"
	"github.com/itchyny/gojq"
	"github.com/weppos/publicsuffix-go/publicsuffix"
)

const ORIGINAL_URL = "ORIGINAL_URL"

type Crawler struct {
	Collector         *colly.Collector
	elementSelector   string
	jqSelector        string
	URLs              []string
	FailedRequestURLs []string
	ScrapedData       []string
}

//---------------------------------------------------------------------------------------

// Return New Instance of a Crawler with an Embedded Colly Collector
func NewCrawler(elementSelector string, jqSelector string, waitTime int, parallelism int) *Crawler {

	// Initialise New Crawler
	c := new(Crawler)
	c.Collector = colly.NewCollector(
		colly.UserAgent("Mozilla/5.0 (X11; Linux x86_64; rv:109.0) Gecko/20100101 Firefox/120.0"),
		colly.MaxDepth(1),
		colly.Async(true),
	)
	_ = c.Collector.Limit(&colly.LimitRule{
		DomainGlob:  "*",
		Parallelism: parallelism,
		RandomDelay: time.Millisecond * time.Duration(waitTime),
	})
	c.Collector.SetRequestTimeout(120 * time.Second)
	c.Collector.WithTransport(&http.Transport{
		DisableKeepAlives: true,
	})
	c.elementSelector = elementSelector
	c.jqSelector = jqSelector

	return c
}

//---------------------------------------------------------------------------------------

// Load all URLs from the first column of the provided CSV File
func (c *Crawler) LoadUrlFile(name string, delimiter string) error {

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
				c.URLs = append(c.URLs, url)
			}
		}
	}

	return nil
}

//---------------------------------------------------------------------------------------

// Deduplicate the list of URLs
func (c *Crawler) DeduplicateURLs() error {

	// Define a hash map and deduped array list
	bucket := make(map[string]bool)
	var deduped []string

	// Iterate through the URL list and remove duplicates
	for _, url := range c.URLs {
		if _, ok := bucket[url]; !ok {
			bucket[url] = true
			deduped = append(deduped, url)
		}
	}

	// Replace the Crawler URL list with the deduped list
	c.URLs = deduped

	return nil
}

//---------------------------------------------------------------------------------------

// Populate the Collector Allowed Domains
func (c *Crawler) SetAllowedDomains() error {

	// Define a hash map and domain array list
	bucket := make(map[string]bool)
	var allowedDomains []string

	logger.Info().Msgf("%s Allowed Domain List", indent)

	// Iterate through the URL list and create a deduped domain list
	for _, rawURL := range c.URLs {
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
	c.Collector.AllowedDomains = allowedDomains

	return nil
}

//---------------------------------------------------------------------------------------

// Execute Scraping of URLs
func (c *Crawler) ExecuteScrape(scrapeXML bool) error {
	defer timer("Colly Collection")()

	// Initialise Scraped Data Output
	c.ScrapedData = make([]string, 0)

	logger.Info().Msgf("%s Colly Collection Started", indent)

	// Executed on every request made by the Colly Collector
	c.Collector.OnRequest(func(r *colly.Request) {
		r.Headers.Set("Accept-Encoding", "gzip")
		r.Ctx.Put(ORIGINAL_URL, r.URL.String())
	})

	// Executed on every response received
	c.Collector.OnResponse(func(r *colly.Response) {
		originalURL := r.Request.Ctx.Get(ORIGINAL_URL)
		logger.Info().Int("Status Code", r.StatusCode).Str("Visited", originalURL).Msg(doubleIndent)
	})

	// Scrape XML or HTML
	if scrapeXML {
		// Executed on every XML element matched by the xpath Query parameter
		c.Collector.OnXML(c.elementSelector, func(element *colly.XMLElement) {
			c.ScrapedData = append(c.ScrapedData, element.Text)
		})
	} else {
		// Executed on every HTML element matched by the GoQuery Selector
		c.Collector.OnHTML(c.elementSelector, func(element *colly.HTMLElement) {

			// Execute the jq Selector
			textSelected, err := jqSelect(element.Text, c.jqSelector)
			if err != nil {
				logger.Error().Err(fmt.Errorf("jq Selector Failed: %w", err)).Msg(doubleIndent)
				return
			}

			c.ScrapedData = append(c.ScrapedData, textSelected)
		})
	}

	// Executed if an error occurs during the HTTP request
	c.Collector.OnError(func(r *colly.Response, err error) {
		originalURL := r.Request.Ctx.Get(ORIGINAL_URL)
		c.FailedRequestURLs = append(c.FailedRequestURLs, originalURL)
		logger.Error().Int("Status Code", r.StatusCode).Err(err).Str("Visited", originalURL).Msg(doubleIndent)
		logger.Debug().Any("Response", r).Msg(doubleIndent)
	})

	// Iterate through the URL List and add to the Collector queue for a Visit
	for _, url := range c.URLs {
		_ = c.Collector.Visit(url)
	}
	c.Collector.Wait()

	logger.Info().Msgf("%s Colly Collection Finished", indent)

	return nil
}

//---------------------------------------------------------------------------------------

// Write the Scraped Data out to a File
func (c *Crawler) WriteDataFile(name string, delimiter string) error {

	logger.Info().Msgf("%s Writing Scraped Data Output File", indent)

	// Open file ready for writing
	file, err := os.Create(name)
	if err != nil {
		return fmt.Errorf("[WriteDataFile] Create File Failed: %w", err)
	}
	defer file.Close()

	// Ready the CSV Writer and use a buffered io writer
	w := csv.NewWriter(bufio.NewWriter(file))
	w.Comma = rune(delimiter[0])
	defer w.Flush()

	// Iterate through the Scraped Data and Write to file
	for _, data := range c.ScrapedData {

		var row []string = make([]string, 1)
		row[0] = strings.Replace(data, "\n", "", -1)

		if err := w.Write(row); err != nil {
			return fmt.Errorf("[WriteDataFile] Failed Writing to the File: %w", err)
		}
	}

	return nil
}

//---------------------------------------------------------------------------------------

// Write the Failed Request URLs out to a File
func (c *Crawler) WriteErrorFile(name string, delimiter string) error {

	logger.Info().Msgf("%s Writing Failed Request URLs Output File", indent)

	// Open file ready for writing
	file, err := os.Create(name)
	if err != nil {
		return fmt.Errorf("[WriteErrorFile] Create File Failed: %w", err)
	}
	defer file.Close()

	// Ready the CSV Writer and use a buffered io writer
	w := csv.NewWriter(bufio.NewWriter(file))
	w.Comma = rune(delimiter[0])
	defer w.Flush()

	// Iterate through the Scraped Data and Write to file
	for _, data := range c.FailedRequestURLs {

		var row []string = make([]string, 1)
		row[0] = strings.Replace(data, "\n", "", -1)

		if err := w.Write(row); err != nil {
			return fmt.Errorf("[WriteErrorFile] Failed Writing to the File: %w", err)
		}
	}

	return nil
}

//---------------------------------------------------------------------------------------

// Execute the 'jq' Selector against the JSON Object text returned
func jqSelect(selectedText string, query string) (string, error) {

	// If the JSON Selector Query was NOT provided then return the element text
	if query == "" {
		return selectedText, nil
	}

	// Convert the element text to a JSON Object before querying
	var jsonData map[string]any
	if err := json.Unmarshal([]byte(selectedText), &jsonData); err != nil {
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

//---------------------------------------------------------------------------------------

// Execute the 'jq' Selector against the JSON Object text returned
func timer(name string) func() {
	start := time.Now()
	return func() {
		logger.Info().Msgf("%s %s took %v", indent, name, time.Since(start))
	}
}
