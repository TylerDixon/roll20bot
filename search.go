package main

import (
	"io/ioutil"
	"log"
	"net/http"
	"net/url"

	"encoding/json"
)

const searchUrl = "https://roll20.net/compendium/compendium/globalsearch/dnd5e?terms="

type SearchResult struct {
	Value        string `json:"value"`
	PageName     string `json:"pagename"`
	GroupByValue string `json:"groupbyvalue"`
}

func SearchRoll(searchParam string) ([]SearchResult, error) {
	escapedSearch := url.QueryEscape(searchParam)
	searchResponse, err := http.Get(searchUrl + escapedSearch)
	if err != nil {
		log.Fatal("Failed to search for term %s", searchParam)
		log.Fatal(err)
		return []SearchResult{}, err
	}
	body, err := ioutil.ReadAll(searchResponse.Body)
	if err != nil {
		log.Fatal("Failed to parse the search for term %s", searchParam)
		return []SearchResult{}, err
	}
	log.Print(string(body))
	result := []SearchResult{}
	json.Unmarshal(body, &result)
	return result, nil
}

func GetPage(result SearchResult) *http.Response {
	resp, err := http.Get(GetPageUrl(result))
	if err != nil {
		log.Fatal(err)
	}

	return resp
}

func GetPageUrl(result SearchResult) string {
	return "https://roll20.net/compendium/dnd5e/" + result.PageName + "#h-" + result.GroupByValue
}
