package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMarkdownDocLinkHandlerMapsFileLinksToAnchors(t *testing.T) {
	assert.Equal(t, "#yews-check", markdownDocLinkHandler("yews_check.md"))
	assert.Equal(t, "https://example.com", markdownDocLinkHandler("https://example.com"))
}
