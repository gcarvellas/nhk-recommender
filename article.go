package main

import (
	"errors"
	"log"
	"strings"

	"github.com/antchfx/htmlquery"
	mapset "github.com/deckarep/golang-set/v2"
)

type Article struct {
	Name       string
	Url        string
	Difficulty float32
}

func getArticleData(article ArticleMetadata, config *Config, ac *ArticleContext) (string, error) {

	resp, err := Api(config.Routes.BaseUrl+article.Link, ac)
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	log.Printf("Parsing and querying article HTML #%v", article)

	doc, err := htmlquery.Parse(resp.Body)
	if err != nil {
		return "", err
	}

	nodes, err := htmlquery.QueryAll(doc, `//*[@class='body-text' or @class='body-title' or @class='content--body' or @class='content--summary' or @class='news_add' or @id='news_textbody']`)
	if err != nil {
		return "", err
	}

	if len(nodes) == 0 {
		return "", errors.New("empty article data")
	}

	var sb strings.Builder

	for _, node := range nodes {
		text := htmlquery.InnerText(node)
		sb.WriteString(text)
		sb.WriteString(" ")
	}

	return sb.String(), nil
}

func ParseArticle(article ArticleMetadata, config *Config, ac *ArticleContext, kwl *KnownWordsList) {

	defer ac.Wg.Done()

	data, err := getArticleData(article, config, ac)
	if err != nil {
		log.Printf("Error fetching %s, Skipping article. Message: %s", article, err)
		return
	}

	log.Printf("Parsing Japanese text of article %#v", article)

	segsAsList := ac.ParseJapaneseText(data)
	if len(segsAsList) == 0 {
		log.Printf("Failed to parse Japanese text of article %#v. Data: %s", article, data)
		return
	}

	log.Printf("Computing article difficulty %#v", article)

	// Convert to a set
	segs := mapset.NewSet[string]()
	for _, seg := range segsAsList {
		segs.Add(seg)
	}

	knownWordsList := kwl.Get(ac)

	difficulty := float32(segs.Intersect(*knownWordsList).Cardinality()) / float32(segs.Cardinality())

	res := Article{Name: article.Title, Url: article.Link, Difficulty: difficulty}

    if !shouldEndArticleSearch(config, ac) {
        ac.ArticlesRead.Add(1)
        log.Printf("Finished computing article %#v", article)
        ac.ArticleProcessCh <- res
    } else {
        log.Printf("Search ended. Discarding article %#v", article)
    }

}
