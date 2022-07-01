package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"
)

var (
	ErrAuth            = errors.New("authentication error. please verify that the api key and secret are correct")
	ErrNoExtract       = errors.New("the account associated with this api key has no files, please contact your technical account manager")
	ErrServerError     = errors.New("server error, please contact your technical account manager")
	ErrUnknownStatus   = errors.New("unexpected error retrieving extract info, try again and contact support if the problem persists")
	ErrMissingAuth     = errors.New("config file requires api_key and api_secret")
	ErrDefaultConfig   = errors.New("default values found, update the config file with your api key and secret")
	ErrNoMatchingFiles = errors.New("no matching extracts found, please check your filters and try again")
)

func contains(str string, list []string) bool {
	for _, entry := range list {
		if entry == str {
			return true
		}
	}
	return false
}

func GetInput(prompt string, allowedResponses []string, acceptFirstChar bool) string {
	var resp string

	var responses []string
	for _, s := range allowedResponses {
		s := strings.ToLower(s)
		responses = append(responses, s)
		if acceptFirstChar {
			responses = append(responses, string(s[0]))
		}
	}

	for !contains(resp, responses) {
		fmt.Println(prompt)
		reader := bufio.NewReader(os.Stdin)
		resp, _ = reader.ReadString('\n')
		resp = strings.ToLower(strings.TrimSuffix(resp, "\n"))
		if acceptFirstChar {
			resp = string(resp[0])
		}
	}
	return resp
}
