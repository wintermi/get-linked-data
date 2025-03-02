# Get Linked Data

[![Workflows](https://github.com/wintermi/get-linked-data/workflows/Go%20-%20Build/badge.svg)](https://github.com/wintermi/get-linked-data/actions)
[![Go Report](https://goreportcard.com/badge/github.com/wintermi/get-linked-data)](https://goreportcard.com/report/github.com/wintermi/get-linked-data)
[![License](https://img.shields.io/github/license/wintermi/get-linked-data.svg)](https://github.com/wintermi/get-linked-data/blob/main/LICENSE)
[![Release](https://img.shields.io/github/v/release/wintermi/get-linked-data?include_prereleases)](https://github.com/wintermi/get-linked-data/releases)


## Description

A command line application designed to crawl a given set of URLs and scrape the JSON Linked Data (JSON-LD) contained within the webpage before writing the data entries out to a CSV file.

```
USAGE:
    get-linked-data -i URL_CSV -s ELEMENT_SELECTOR -o OUTPUT_CSV -e FAILED_URL_CSV

ARGS:
  -d string
    	Field Delimiter  (Required) (default ",")
  -e string
    	Failed Request URLs Output CSV File  (Required)
  -g	Scrape Google's Cached Version Instead
  -i string
    	CSV File containing URLs to Scrape  (Required)
  -j string
    	jq Selector
  -o string
    	Output Scraped Data CSV File  (Required)
  -p int
    	Parallelism or Maximum allowed Concurrent Requests (default 100)
  -s string
    	Element Selector  (Required)
  -v	Output Verbose Detail
  -w int
    	Random Wait Time in Milliseconds between Requests (default 2000)
  -x	Scrape XML not HTML
```

## Example

```
get-linked-data -i "urls.csv" -e "script#product-schema" -o "results.csv"
```


## License

**get-linked-data** is released under the [Apache License 2.0](https://github.com/wintermi/get-linked-data/blob/main/LICENSE) unless explicitly mentioned in the file header.
