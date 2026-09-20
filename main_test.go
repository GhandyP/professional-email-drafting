package main

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/GhandyP/professional-email-drafting/internal/email"
)

func TestExampleInputFixtureMatchesTheContract(t *testing.T) {
	contents, err := os.ReadFile("testdata/email.json")
	if err != nil {
		t.Fatalf("read testdata/email.json: %v", err)
	}

	var input email.Input
	if err := json.Unmarshal(contents, &input); err != nil {
		t.Fatalf("decode testdata/email.json: %v", err)
	}
	if err := input.Validate(); err != nil {
		t.Fatalf("testdata/email.json is not a valid input: %v", err)
	}
	if len(input.Facts) < 2 {
		t.Fatalf("testdata/email.json has %d facts, want at least 2", len(input.Facts))
	}
}
