package main

func main() {

    config := ParseConfig()
    ac := newArticleContext(&config)

    go GetKnownWordsList(&ac)
    CompareArticles(&config, &ac, &knownWordsList)
}
