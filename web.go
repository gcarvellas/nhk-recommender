package main

import (
	"encoding/json"
	"log"
	"net/http"
)

func Api(url string, ac *ArticleContext) (*http.Response, error) {
    rl := *ac.Rl
    rl.Take() 
    log.Printf("Web request to: %s", url)
    return http.Get(url)
}

func ApiDecode[T any](url string, ac *ArticleContext) (T, error) {

    var decoded T

    resp, err := Api(url, ac)
    if err != nil {
        return decoded, err
    }
    defer resp.Body.Close()

    err = json.NewDecoder(resp.Body).Decode(&decoded)

    return decoded, err
}

