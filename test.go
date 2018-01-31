package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"golang.org/x/net/html"

	"github.com/gorilla/mux"
)

var tagsToIgnore = [...]string{"script", "select", "option"}

func main() {

	r := mux.NewRouter()

	r.HandleFunc("/slack/search", func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseForm()
		if err != nil {
			log.Fatal("Failed slack command received")
			fmt.Fprintf(w, "Failed to parse slack command")
			return
		}
		if r.Form["text"] != nil {
			searchResults, _ := SearchRoll(r.Form["text"][0])
			if len(searchResults) > 1 {
				var suggestions []Option
				for _, result := range searchResults {
					marshaledResult, _ := json.Marshal(result)
					suggestions = append(suggestions, Option{result.Value, string(marshaledResult)})
				}
				reply := &SelectReply{
					MainText:      "I could tell you of many things, but which do you need to know of my child?",
					IsInChannel:   true,
					SecondaryText: "Tell me",
					CallbackId:    "option_select",
					ActionName:    "Info Select",
					DropDownText:  "Choose one...",
					Options:       suggestions,
				}
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprintf(w, string(reply.FormatReply()))
				return
			} else {
				searchResult := searchResults[0]
				resp := GetPage(searchResult)
				log.Printf("Retrieved page for %s", searchResult.PageName)
				w.Header().Set("Content-Type", "text/html")
				retBody := findSection(resp.Body, searchResult.GroupByValue)
				fmt.Fprintf(w, retBody)
			}
		} else if r.Form["payload"] != nil {
			var messageReply interface{}
			err := json.Unmarshal([]byte(r.Form["payload"][0]), &messageReply)
			if err != nil {
				fmt.Fprintf(w, "Failed to parse form payload")
				return
			}
			m := messageReply.(map[string]interface{})
			actions := m["actions"].([]interface{})
			if len(actions) == 0 {
				fmt.Fprintf(w, "Incoming payload had no action")
				return
			}
			selectedOptions := actions[0].(map[string]interface{})["selected_options"].([]interface{})
			if len(selectedOptions) == 0 {
				fmt.Fprintf(w, "Incoming payload had no selected options")
			}
			selectedValue := selectedOptions[0].(map[string]interface{})["value"].(string)
			var searchResult SearchResult
			searchResultParseErr := json.Unmarshal([]byte(selectedValue), &searchResult)
			if searchResultParseErr != nil {
				fmt.Fprintf(w, "Failed to parse search result from slack reply")
			}
			resp := GetPage(searchResult)
			log.Printf("Retrieved page for %s", searchResult.PageName)
			w.Header().Set("Content-Type", "text/html")
			retBody := findSection(resp.Body, searchResult.GroupByValue)
			reply, _ := json.Marshal(map[string]interface{}{
				"response_type": "in_channel",
				"text":          retBody,
				"attachments": []interface{}{
					map[string]interface{}{
						"text": retBody,
					},
				},
			})
			fmt.Fprintf(w, string(reply))
		}
	})

	r.HandleFunc("/compendium/dnd5e/{pagename}", func(w http.ResponseWriter, r *http.Request) {
		rawQuery := r.URL.RawQuery
		if len(rawQuery) < 2 {
			fmt.Fprint(w, "Failed to retrieve specific page info with URL ", r.URL)
			return
		}
		vars := mux.Vars(r)
		searchResult := SearchResult{
			GroupByValue: rawQuery[2:],
			PageName:     vars["pagename"],
		}
		resp := GetPage(searchResult)
		w.Header().Set("Content-Type", "text/html")
		retBody := findSection(resp.Body, searchResult.GroupByValue)
		fmt.Fprintf(w, retBody)
	})

	r.HandleFunc("/search/{search}", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		searchParam := vars["search"]
		searchForValue(searchParam, w, r)
	})
	log.Fatal(http.ListenAndServe(":80", r))
}

func searchForValue(searchParam string, w http.ResponseWriter, r *http.Request) {
	searchResults, _ := SearchRoll(searchParam)
	if len(searchResults) > 1 {
		var suggestions []string
		for _, result := range searchResults {
			suggestions = append(suggestions, "<a href=\"/search/"+result.Value+"\">"+result.Value+"</a>")
		}
		fmt.Fprintf(w, strings.Join(suggestions, "<br/>"))
		return
	} else if len(searchResults) == 1 {
		resp := GetPage(searchResults[0])
		w.Header().Set("Content-Type", "text/html")
		retBody := findSection(resp.Body, searchParam)
		fmt.Fprintf(w, retBody)
	} else {
		fmt.Fprintf(w, "Failed to retrieve search for %q", searchParam)
	}

}

func findSection(body io.Reader, title string) string {
	tokenizer := html.NewTokenizer(body)
	recording := false
	depth := 1
	recorded := "Could not parse information"
	recordingPaused := false
	currentTagType := "z"
	var recordedTagType string
	var tagsToAdd string
	for {
		tt := tokenizer.Next()
		if tt == html.ErrorToken {
			return recorded
		}
		if !recording {
			if tt == html.StartTagToken {
				token := tokenizer.Token()
				currentTagType = token.Data
			}
			if tt == html.EndTagToken {
				currentTagType = "z"
			}
			if tt == html.TextToken && currentTagType[0] == 'h' {
				nodeContent := string(tokenizer.Text())
				if nodeContent == title {
					log.Printf("Found content for title %s", title)
					recording = true
					recordedTagType = currentTagType
					recorded = "*" + nodeContent
				}
			}
		} else {
			if tt == html.StartTagToken {
				token := tokenizer.Token()
				for _, tag := range tagsToIgnore {
					if strings.Compare(tag, token.Data) == 0 {
						recordingPaused = true
					}
				}
				if token.Data == recordedTagType {
					return recorded
				}
				if token.Data[0] == 'h' {
					tagsToAdd += "\n*"
				} else if token.Data[0] == 'a' {
					var href string
					for _, a := range token.Attr {
						if a.Key == "href" {
							href = a.Val
							break
						}
					}
					tagsToAdd = "<https://roll20.net" + href + "|"
				} else if token.Data[0] == 'b' {
					tagsToAdd += "\n"
				} else if !recordingPaused {
					tokenString := token.String()
					tokenString = strings.Replace(tokenString, "#", "?", -1)
					tagsToAdd += tokenString
				}
				depth++
			} else if tt == html.EndTagToken {
				token := tokenizer.Token()
				for _, tag := range tagsToIgnore {
					if strings.Compare(tag, token.Data) == 0 {
						recordingPaused = false
					}
				}
				if token.Data[0] == 'h' {
					recorded += "*\n"
				} else if token.Data[0] == 'a' {
					recorded += ">"
				} else if !recordingPaused {
					recorded += token.String()
				}
				depth--
				if depth == -1 {
					return recorded
				}
			} else if !recordingPaused {
				recorded += tagsToAdd
				tagsToAdd = ""
				text := tokenizer.Token()
				recorded += text.Data
			}

		}
	}
	return recorded
}
