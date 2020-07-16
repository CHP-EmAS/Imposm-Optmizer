package tagfinder

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
)

type result struct {
	IsKey     bool   `json:"isKey"`
	IsTag     bool   `json:"isTag"`
	PrefLabel string `json:"prefLabel"`
}

//FindTagKey Search via API for all Key accourences that have a specific tag
func FindTagKey(tag string) []string {
	url := "http://tagfinder.herokuapp.com/api/search?query=" + tag

	response, err := http.Get(url)

	if err != nil {
		fmt.Println("The HTTP request failed with error " + err.Error())
		return []string{}
	}

	data, _ := ioutil.ReadAll(response.Body)

	foundKeyList := make([]string, 0)
	apiResponses := []result{}

	json.Unmarshal(data, &apiResponses)

	for _, singleResult := range apiResponses {
		if singleResult.IsTag && !singleResult.IsKey {
			splitLable := strings.Split(singleResult.PrefLabel, "=")

			if len(splitLable) >= 2 {
				if splitLable[1] == tag {
					if !strings.Contains(splitLable[0], ":") {
						if CheckIfKeyExists(splitLable[0]) {
							foundKeyList = append(foundKeyList, splitLable[0])
						}
					}
				}
			}
		}
	}

	return foundKeyList
}

//CheckIfKeyExists Checks via API if a specific Key exists or not
func CheckIfKeyExists(key string) bool {

	url := "http://tagfinder.herokuapp.com/api/search?query=" + key

	response, err := http.Get(url)

	if err != nil {
		fmt.Println("The HTTP request failed with error " + err.Error())
		return false
	}

	data, _ := ioutil.ReadAll(response.Body)

	apiResponses := []result{}

	json.Unmarshal(data, &apiResponses)

	for _, singleResult := range apiResponses {
		if !singleResult.IsTag && singleResult.IsKey {
			if singleResult.PrefLabel == key {
				return true
			}
		}
	}

	return false
}
