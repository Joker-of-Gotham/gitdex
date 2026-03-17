package contract

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

type validationGoldenCase struct {
	Name         string         `json:"name"`
	Kind         string         `json:"kind"`
	Suggestion   SuggestionItem `json:"suggestion"`
	Action       ActionSpec     `json:"action"`
	WantError    bool           `json:"want_error"`
	ErrorContain string         `json:"error_contains"`
}

func TestValidationGolden(t *testing.T) {
	raw, err := os.ReadFile("testdata/validation_golden.json")
	if err != nil {
		t.Fatalf("read golden file: %v", err)
	}
	var cases []validationGoldenCase
	if err := json.Unmarshal(raw, &cases); err != nil {
		t.Fatalf("unmarshal golden cases: %v", err)
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			var gotErr error
			switch strings.ToLower(strings.TrimSpace(tc.Kind)) {
			case "suggestion":
				gotErr = ValidateSuggestion(tc.Suggestion)
			case "action":
				gotErr = ValidateAction(tc.Action)
			default:
				t.Fatalf("unsupported golden kind: %q", tc.Kind)
			}
			if tc.WantError && gotErr == nil {
				t.Fatalf("expected error, got nil")
			}
			if !tc.WantError && gotErr != nil {
				t.Fatalf("expected nil error, got %v", gotErr)
			}
			if tc.ErrorContain != "" && gotErr != nil && !strings.Contains(gotErr.Error(), tc.ErrorContain) {
				t.Fatalf("expected error to contain %q, got %q", tc.ErrorContain, gotErr.Error())
			}
		})
	}
}
