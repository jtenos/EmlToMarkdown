package main

import "testing"

func TestConvert(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "plain file name", input: "email.eml", want: "File email.eml has been converted"},
		{name: "strips directory", input: "/tmp/mail/email.eml", want: "File email.eml has been converted"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := Convert(tc.input); got != tc.want {
				t.Errorf("Convert(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}
