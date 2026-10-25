package main

import "testing"

func TestMatterDecision(t *testing.T) {
	cases := []struct {
		name    string
		input   MatterNotice
		want    string
		message bool
	}{
		{"intake allowed", MatterNotice{MatterID: "M-42", Stage: "intake", Phone: "+15550101010", Detail: "We received your intake"}, "ready", true},
		{"signed document opted out", MatterNotice{MatterID: "M-42", Stage: "signed_document", Phone: "+15550101010", Detail: "Review the signed copy", OptedOut: true}, "skipped_opt_out", false},
		{"deadline suppressed", MatterNotice{MatterID: "M-42", Stage: "deadline", Phone: "+15550101010", Detail: "Response due tomorrow", SuppressedPhones: []string{"+15550101010"}}, "skipped_suppression", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, msg, err := decide(tc.input)
			if err != nil || got.Status != tc.want || (msg != "") != tc.message {
				t.Fatalf("decision=%+v message=%q err=%v", got, msg, err)
			}
		})
	}
}
