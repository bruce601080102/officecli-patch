package main

import "testing"

func TestDefaultRewritePathPreservesOfficeExtension(t *testing.T) {
	tests := map[string]string{
		"report.docx": "report.rewritten.docx",
		"book.xlsx":   "book.rewritten.xlsx",
		"slides.pptx": "slides.rewritten.pptx",
	}
	for input, want := range tests {
		if got := defaultRewritePath(input); got != want {
			t.Errorf("defaultRewritePath(%q) = %q, want %q", input, got, want)
		}
	}
}
