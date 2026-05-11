package app

import (
	"strings"
	"testing"
)

func TestParseCharacterCardV2JSON(t *testing.T) {
	raw := []byte(`{
		"spec": "chara_card_v2",
		"data": {
			"name": "Mira",
			"description": "A moonlit archivist.",
			"personality": "Gentle but sharp.",
			"scenario": "A library after midnight.",
			"first_mes": "Welcome back.",
			"character_book": {
				"entries": [
					{"comment": "Library", "keys": ["archive", "moon"], "content": "The archive remembers every promise.", "enabled": true},
					{"comment": "Disabled", "content": "Hidden", "enabled": false}
				]
			}
		}
	}`)

	result, err := parseCharacterCard(raw, ".json")
	if err != nil {
		t.Fatal(err)
	}
	if result.Name != "Mira" {
		t.Fatalf("expected name Mira, got %q", result.Name)
	}
	if !strings.Contains(result.RoleCard, "A moonlit archivist.") || !strings.Contains(result.RoleCard, "Welcome back.") {
		t.Fatalf("role card missing expected content: %s", result.RoleCard)
	}
	if !strings.Contains(result.WorldBook, "The archive remembers every promise.") || strings.Contains(result.WorldBook, "Hidden") {
		t.Fatalf("world book did not import enabled entries correctly: %s", result.WorldBook)
	}
}
