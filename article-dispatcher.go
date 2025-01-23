package main

import (
    "fmt"
    "log"
    "regexp"
    "strings"
    "sync"
    "sync/atomic"

    mapset "github.com/deckarep/golang-set/v2"
    "github.com/dlclark/regexp2"
    "github.com/ikawaha/kagome-dict/ipa"
    "github.com/ikawaha/kagome/v2/tokenizer"
    "go.uber.org/ratelimit"
)

// Keyword list API structs

type KeywordItemApi struct {
    Word    string `json:"word"`
    Link    string `json:"link"`
    Kijinum int    `json:"kijinum"`
}

type KeywordsApi struct {
    Item []KeywordItemApi `json:"item"`
}

// Article list API struct

type ArticleListApi struct {
    Channel struct {
        LastBuildDate string `json:"lastBuildDate"`
        HasNext       bool   `json:"hasNext"`
        Word          string `json:"word"`
        Item          []struct {
            ID            string   `json:"id"`
            Title         string   `json:"title"`
            PubDate       string   `json:"pubDate"`
            Cate          string   `json:"cate"`
            CateGroup     []string `json:"cate_group"`
            Link          string   `json:"link"`
            ImgPath       string   `json:"imgPath"`
            IconPath      string   `json:"iconPath"`
            VideoPath     string   `json:"videoPath"`
            VideoDuration string   `json:"videoDuration"`
            RelationNews  []struct {
                Link  string `json:"link"`
                Title string `json:"title"`
            } `json:"relationNews"`
        } `json:"item"`
    } `json:"channel"`
}

// User-Defined Structs

type ArticleMetadata struct {
    Title string
    Link  string
}

type ArticleContext struct {
    ArticleSearchCh  chan ArticleMetadata
    Wg               *sync.WaitGroup
    ArticleProcessCh chan Article
    ArticlesRead     *atomic.Uint64
    Rl               *ratelimit.Limiter
    ArticleIdRegex   *regexp.Regexp
    JpRegex          *regexp2.Regexp
    JpTokenizer      *tokenizer.Tokenizer
    endArticleCalculationOnce sync.Once
}

func (ac *ArticleContext) Close() {
    ac.Wg.Wait()
    close(ac.ArticleProcessCh)
    close(ac.ArticleSearchCh)
}

func (ac *ArticleContext) ParseJapaneseText(field string) []string {

    var filtered strings.Builder
    match, _ := ac.JpRegex.FindStringMatch(field)
    for match != nil {
        filtered.WriteString(match.String())
        match, _ = ac.JpRegex.FindNextMatch(match)
    }

    return ac.JpTokenizer.Wakati(filtered.String())
}

func newArticleContext(config *Config) ArticleContext {
    articleSearchCh := make(chan ArticleMetadata, config.NumArticles)
    articleProcessCh := make(chan Article, config.NumArticles)

    var wg sync.WaitGroup

    var articlesRead atomic.Uint64

    rl := ratelimit.New(config.RequestsPerSecond)

    articleIdRegex := regexp.MustCompile(`([0-9]+)`)

    jpTokenizer, err := tokenizer.New(ipa.Dict(), tokenizer.OmitBosEos())
    if err != nil {
        log.Panicf("Failed to initialize tokenizer: %s", err)
    }

    pattern := `\p{Hiragana}|\p{Katakana}|\p{Han}`
    jpRegex := regexp2.MustCompile(pattern, regexp2.None)

    return ArticleContext{JpTokenizer: jpTokenizer, JpRegex: jpRegex, ArticleSearchCh: articleSearchCh, ArticleProcessCh: articleProcessCh, Wg: &wg, ArticlesRead: &articlesRead, Rl: &rl, ArticleIdRegex: articleIdRegex}
}

func shouldEndArticleSearch(config *Config, ac *ArticleContext) bool {
    res := ac.ArticlesRead.Load() >= config.NumArticles

    // The waitgroup gets counted down when all articles are done. See CompareArticles
    if res == true {
        ac.endArticleCalculationOnce.Do(func() {
            ac.Wg.Done()
        })
    }

    return res
}

func discoverArticleLoop(item KeywordItemApi, config *Config, ac *ArticleContext) {

    defer ac.Wg.Done()

    // Parse the article list id out from the item
    articleListId := ac.ArticleIdRegex.FindString(item.Link)

    if articleListId == "" {
        log.Printf("Error: Unknown id in item. Skipping key. %s", item.Link)
        return
    }

    // Cannot paginage past 999 in the nhk api
    for page := 1; page <= 999; page++ {

        if shouldEndArticleSearch(config, ac) {
            break
        }

        url := fmt.Sprintf("/%s_%03d.json", articleListId, page)
        articleList, err := ApiDecode[ArticleListApi](config.Routes.ArticleListBaseUrl+url, ac)
        if err != nil {
            log.Printf("Error when fetching %#v. Skipping key. Message: %s", item, err)
            break
        }

        for _, item := range articleList.Channel.Item {
            article := ArticleMetadata{Title: item.Title, Link: item.Link}
            ac.ArticleSearchCh <- article
        }

        if !articleList.Channel.HasNext {
            break
        }
    }
}

func findArticles(config *Config, ac *ArticleContext) {

    defer ac.Wg.Done()

    keys, err := ApiDecode[KeywordsApi](config.Routes.KeywordBaseUrl, ac)
    if err != nil {
        log.Panicf("Failed to get keyword list: %s", err)
    }

    items := keys.Item

    // Only iterate the necessary amount of keys
    keysToIterate := min(int(config.NumArticles), len(items))

    for _, key := range items[:keysToIterate] {
        ac.Wg.Add(1)
        go discoverArticleLoop(key, config, ac)
    }
}

func CompareArticles(config *Config, ac *ArticleContext, knownWordsList *mapset.Set[string]) {

    defer ac.Close()

    // This gets counted down when all articles are done. See shouldEndArticleSearch 
    ac.Wg.Add(1)

    ac.Wg.Add(1)
    go findArticles(config, ac)

    go func() {
        for article := range ac.ArticleSearchCh {
            ac.Wg.Add(1)
            go ParseArticle(article, config, ac)
        }
    }()

    go func() {
        for article := range ac.ArticleProcessCh {
            fmt.Printf("%f,%s,%s\n", article.Difficulty, article.Name, article.Url)
        }
    }()

}
