package functions

import (
	"container/list"
	"fmt"
	"os"
	"strings"
)

//StringInList Checks if a List contains a specific string
func StringInList(a string, list *list.List) bool {
	for element := list.Front(); element != nil; element = element.Next() {
		if strings.Compare(a, fmt.Sprintf("%v", element.Value)) == 0 {
			return true
		}
	}
	return false
}

//StringInSlice Checks if a Slice contains a specific string
func StringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

//FileExists Checks if a file exists
func FileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

//DirExists Checks if a directory exists
func DirExists(dirName string) bool {
	info, err := os.Stat(dirName)
	if os.IsNotExist(err) {
		return false
	}
	return info.IsDir()
}
