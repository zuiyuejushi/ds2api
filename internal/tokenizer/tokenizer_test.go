package tokenizer

import "testing"

func TestCountTokensEmpty(t *testing.T) {
	if n := CountTokens(""); n != 0 {
		t.Errorf("expected 0 for empty string, got %d", n)
	}
}

func TestCountTokensEnglish(t *testing.T) {
	// "Hello, world!" is about 3-4 tokens with cl100k_base
	n := CountTokens("Hello, world!")
	if n < 2 || n > 6 {
		t.Errorf("unexpected token count for 'Hello, world!': %d", n)
	}
}

func TestCountTokensChinese(t *testing.T) {
	// Chinese text: each character is typically 1-2 tokens
	n := CountTokens("你好世界")
	if n < 2 || n > 12 {
		t.Errorf("unexpected token count for '你好世界': %d", n)
	}
}

func TestCountTokensLongText(t *testing.T) {
	text := "The quick brown fox jumps over the lazy dog. " +
		"The quick brown fox jumps over the lazy dog. " +
		"The quick brown fox jumps over the lazy dog."
	n := CountTokens(text)
	if n < 20 || n > 60 {
		t.Errorf("unexpected token count for long text: %d", n)
	}
}
