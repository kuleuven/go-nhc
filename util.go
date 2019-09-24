package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"strings"
)

func AssureExists(file string) (Status, string) {
	if _, err := os.Stat(file); err != nil {
		return Critical, fmt.Sprintf("File %s missing: %s", file, err.Error())
	}
	return OK, ""
}

func AssureContent(file string, re *regexp.Regexp) (Status, string) {
	if _, err := os.Stat(file); err != nil {
		return Critical, fmt.Sprintf("File %s missing: %s", file, err.Error())
	}

	handle, err := os.Open(file)
	defer handle.Close()
	if err != nil {
		return Unknown, fmt.Sprintf("Could not open file %s: %s", file, err.Error())
	}

	b, err := ioutil.ReadAll(handle)
	if err != nil {
		return Unknown, fmt.Sprintf("Could not read file %s: %s", file, err.Error())
	}

	if !re.Match(b) {
		s := strings.TrimSuffix(string(b), "\n")
		return Critical, fmt.Sprintf("File %s does not contain %s: %s", file, re.String(), s)
	}

	return OK, ""
}
