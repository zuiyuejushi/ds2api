package tokenizer

import (
	"sync"

	"github.com/pkoukk/tiktoken-go"
)

var (
	once    sync.Once
	enc     *tiktoken.Tiktoken
	initErr error
)

func getEncoding() (*tiktoken.Tiktoken, error) {
	once.Do(func() {
		enc, initErr = tiktoken.GetEncoding("cl100k_base")
	})
	return enc, initErr
}

// CountTokens returns the accurate token count for the given text
// using the cl100k_base encoding (GPT-4 / GPT-3.5-turbo family).
// Falls back to a character-ratio heuristic if the tokenizer fails to load.
func CountTokens(text string) int {
	if text == "" {
		return 0
	}
	e, err := getEncoding()
	if err != nil {
		return fallbackEstimate(text)
	}
	tokens := e.Encode(text, nil, nil)
	return len(tokens)
}

// fallbackEstimate is the old character-ratio heuristic, copied inline
// to avoid importing the util package (circular dependency).
func fallbackEstimate(text string) int {
	asciiChars := 0
	nonASCIIChars := 0
	for _, r := range text {
		if r < 128 {
			asciiChars++
		} else {
			nonASCIIChars++
		}
	}
	n := asciiChars/4 + (nonASCIIChars*10+7)/13
	if n < 1 {
		return 1
	}
	return n
}
