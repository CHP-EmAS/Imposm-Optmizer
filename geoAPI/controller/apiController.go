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

//AddLayer Search via API for all Key accourences that have a specific tag
func AddLayer(stylename string) []string {
	url := "http://127.0.0.1:8080/geoserver/api/v2/workspace/000001-4A25D/layer/0/style/name=" + stylename

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

func test(key string) bool {

	url := "http://127.0.0.1:8080/geoserver/api/v2/workspace/" + key

	response, err := http.Post(url)

	if err != nil {
		fmt.Println("The HTTP request failed with error " + err.Error())
		return false
	}
	return false
}
