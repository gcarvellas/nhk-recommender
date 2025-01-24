package main

func main() {

    config := ParseConfig()
    ac := NewArticleContext(&config)

    kwl := NewKnownWordsList()

    // Compute this in the background so it's ready when the article parser needs it
    go kwl.Get(&ac)

    CompareArticles(&config, &ac, &kwl)
}
