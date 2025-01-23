package main

import (
	"log"
	"sync"
	"github.com/atselvan/ankiconnect"
	mapset "github.com/deckarep/golang-set/v2"
	"github.com/dlclark/regexp2"
)

var (
    knownWordsList mapset.Set[string]
    calcKnownWordsListOnce sync.Once
)

func getAnkiCardQuestions() *[]ankiconnect.ResultCardsInfo {
	client := ankiconnect.NewClient()

	log.Println("Fetching all non-new cards from ankiconnect")

	// Get all cards that are not new
	cards, err := client.Cards.Get("-is:new")
	if err != nil {
        log.Panicf("Failed to query anki cards: #%v", err)
	}

	log.Printf("Gathered %d cards from anki\n", len(*cards))

	return cards
}

type CardsProcessor struct {
	Ch         chan []string
	Wg         *sync.WaitGroup
	NoBracketR *regexp2.Regexp
    Ac         *ArticleContext
}

func newCardProcessor(chanLen int, ac *ArticleContext) CardsProcessor {

	log.Println("Initializing cards processor")

	ch := make(chan []string, chanLen)
	var wg sync.WaitGroup

	// Regex to only filter hiragana, katakana, and kanji
	pattern := `\p{Hiragana}|\p{Katakana}|\p{Han}`

	pattern = `\[[^\]]*\]|\{[^}]*\}`
	noBracketR := regexp2.MustCompile(pattern, regexp2.None)

    return CardsProcessor{Ch: ch, Wg: &wg, NoBracketR: noBracketR, Ac: ac}
}

func determineSuitableField(card ankiconnect.ResultCardsInfo, cp *CardsProcessor) *string {
	/*
	 * Find the appropriate field to query
	 * TODO instead of hardcoding field values, make it configurable. Or, find some other non-hacky way to do this reliably
	 */

	if val, ok := card.Fields["Sentence"]; ok {
		cleanedText, _ := cp.NoBracketR.Replace(val.Value, "", 0, -1)
		return &cleanedText
	}

	if val, ok := card.Fields["kanji"]; ok {
		return &val.Value
	}

	if val, ok := card.Fields["Kanji"]; ok {
		return &val.Value
	}

	if val, ok := card.Fields["Radical"]; ok {
		return &val.Value
	}

	if val, ok := card.Fields["Vocabulary-Kanji"]; ok {
		return &val.Value
	}

    // If none of these fields are available, default to the question card
	return &card.Question
}

func handleCard(card ankiconnect.ResultCardsInfo, cp *CardsProcessor) {
    defer cp.Wg.Done()
	field := determineSuitableField(card, cp)
	if field == nil {
		return
	}
    segs := cp.Ac.ParseJapaneseText(*field)

    if len(segs) > 0 {
        cp.Ch <- segs
    }

}

func GetKnownWordsList(ac *ArticleContext) *mapset.Set[string] {
    calcKnownWordsListOnce.Do(func() {
        knownWordsList = GenerateKnownWordList(ac)
    })
    return &knownWordsList
}

func GenerateKnownWordList(ac *ArticleContext) mapset.Set[string] {

    log.Println("Gathering cards from anki")

    cards := *getAnkiCardQuestions()

	cp := newCardProcessor(len(cards), ac)

	log.Printf("Processing %d cards", len(cards))

	for _, card := range cards {
		cp.Wg.Add(1)
		go handleCard(card, &cp)
	}

	go func() {
		cp.Wg.Wait()
		close(cp.Ch)
	}()

	result := mapset.NewSet[string]()
	for msgs := range cp.Ch {
		for _, msg := range msgs {
			result.Add(msg)
		}
	}

    log.Printf("Detected %d words", result.Cardinality())

	return result

}
