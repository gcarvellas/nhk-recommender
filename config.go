package main

import (
    "flag"
    "io"
    "log"
)

type Config struct {
    NumArticles       uint64
    RequestsPerSecond int
    Routes            Routes
}

type Routes struct {
    BaseUrl            string
    KeywordBaseUrl     string
    NewsBaseUrl        string
    ArticleListBaseUrl string
}

const DefaultNumArticles = 100
const DefaultNhkBaseUrl = "https://www3.nhk.or.jp"
const DefaultRequestsPerSecond = 3

func ParseConfig() Config {
    numArticles := flag.Uint64("articles", DefaultNumArticles, "number of articles to search in nhk")
    requestsPerSecond := flag.Int("requestsPerSecond", DefaultRequestsPerSecond, "number of http requests to NHK per second")
    debug := flag.Bool("debug", false, "enable debug logging")

    flag.Parse()

    if !*debug {
        log.SetOutput(io.Discard)
    }

    baseUrl := DefaultNhkBaseUrl
    newsBaseUrl := baseUrl + "/news"
    keywordBaseUrl := newsBaseUrl + "/json16/keyword_list.json"
    articleListBaseUrl := newsBaseUrl + "/json16/word"

    routes := Routes{BaseUrl: baseUrl, KeywordBaseUrl: keywordBaseUrl, NewsBaseUrl: newsBaseUrl, ArticleListBaseUrl: articleListBaseUrl}

    return Config{NumArticles: *numArticles, RequestsPerSecond: *requestsPerSecond, Routes: routes}
}
