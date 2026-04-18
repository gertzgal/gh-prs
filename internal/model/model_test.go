package model_test

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/gertzgal/gh-prs/internal/model"
)

func TestPR_MarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		pr      model.PR
		asserts []func(t *testing.T, raw []byte)
	}{
		{
			name: "zero-value review decision and ci state marshal to null",
			pr:   model.PR{Number: 1, Title: "t", URL: "u"},
			asserts: []func(t *testing.T, raw []byte){
				mustContain(`"reviewDecision":null`),
				mustContain(`"ciState":null`),
			},
		},
		{
			name: "populated review decision and ci state marshal to strings",
			pr: model.PR{
				Number:         42,
				Title:          "hello",
				URL:            "https://example.com/pull/42",
				ReviewDecision: model.ReviewApproved,
				CiState:        model.CiSuccess,
			},
			asserts: []func(t *testing.T, raw []byte){
				mustContain(`"reviewDecision":"APPROVED"`),
				mustContain(`"ciState":"SUCCESS"`),
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			raw, err := json.Marshal(tc.pr)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			for _, a := range tc.asserts {
				a(t, raw)
			}
		})
	}
}

func TestPR_AllFieldsRoundTrip(t *testing.T) {
	pr := model.PR{
		Number:           7,
		Title:            "Round trip test",
		URL:              "https://github.com/acme/repo/pull/7",
		IsDraft:          true,
		HeadRefName:      "feature/x",
		BaseRefName:      "main",
		Additions:        100,
		Deletions:        50,
		ChangedFiles:     4,
		ReviewDecision:   model.ReviewChangesRequested,
		CiState:          model.CiFailure,
		MergeStateStatus: "BLOCKED",
	}

	raw, err := json.Marshal(pr)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded model.PR
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if decoded != pr {
		t.Errorf("round-trip mismatch:\n got  %#v\n want %#v", decoded, pr)
	}
}

func TestPR_ZeroValueRoundTrip(t *testing.T) {
	pr := model.PR{Number: 1, Title: "t", URL: "u"}

	raw, err := json.Marshal(pr)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded model.PR
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if decoded != pr {
		t.Errorf("round-trip mismatch:\n got  %#v\n want %#v", decoded, pr)
	}
}

func mustContain(substr string) func(t *testing.T, raw []byte) {
	return func(t *testing.T, raw []byte) {
		t.Helper()
		if !bytes.Contains(raw, []byte(substr)) {
			t.Errorf("want output to contain %q, got: %s", substr, raw)
		}
	}
}
