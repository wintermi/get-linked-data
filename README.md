# Get Linked Data

[![Workflows](https://github.com/wintermi/get-linked-data/workflows/Go/badge.svg)](https://github.com/wintermi/get-linked-data/actions)
[![Go Report](https://goreportcard.com/badge/github.com/wintermi/get-linked-data)](https://goreportcard.com/report/github.com/wintermi/get-linked-data)
[![License](https://img.shields.io/github/license/wintermi/get-linked-data.svg)](https://github.com/wintermi/get-linked-data/blob/main/LICENSE)
[![Release](https://img.shields.io/github/v/release/wintermi/get-linked-data?include_prereleases)](https://github.com/wintermi/get-linked-data/releases)


## Description

A command line application designed to crawl a given set of URLs and scrape the JSON Linked Data (JSON-LD) contained within the webpage before writing the data entries out to a CSV file.

```
USAGE:
    get-linked-data -i URL_CSV -e ELEMENT_SELECTOR -o OUTPUT_CSV

ARGS:
  -c	Crawl URLs before Scraping
  -d string
    	Field Delimiter  (Required) (default ",")
  -e string
    	Element Selector  (Required)
  -i string
    	CSV File containing URLs to Scrape  (Required)
  -o string
    	Output CSV File  (Required)
  -v	Output Verbose Detail
```

## Example

```
get-linked-data -i "urls.csv" -e "script#product-schema" -o "results.csv"
```


## License

**get-linked-data** is released under the [Apache License 2.0](https://github.com/wintermi/get-linked-data/blob/main/LICENSE) unless explicitly mentioned in the file header.
