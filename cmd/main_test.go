package main

import (
	"testing"
)

func TestIsFileMatch(t *testing.T) {
	cases := []struct {
		fileName string
		nameTerm string
		ext      string
		want     bool
	}{
		{
			"messages.csv",
			"messages",
			"csv",
			true,
		},
		{
			"Messages.csv",
			"messages",
			"csv",
			true,
		},
		{
			"Messages.cSv",
			"messages",
			"csv",
			true,
		},
		{
			"12xasdlkjf-_messages.csv",
			"messages",
			"csv",
			true,
		},
		{
			"messages12xasdlkjf-_.csv",
			"messages",
			"csv",
			true,
		},
		{
			"messages12xasdlkjf-_.csv",
			"messages",
			"csv",
			true,
		},
		{
			"12xasdlkjf-_messages12xasdlkjf-_.csv",
			"messages",
			"csv",
			true,
		},
		{
			"&messages.csv",
			"messages",
			"csv",
			false,
		},
		{
			"messagescsv",
			"messages",
			"csv",
			false,
		},
		{
			"messages..csv",
			"messages",
			"csv",
			false,
		},
		{
			"message.csv",
			"messages",
			"csv",
			false,
		},
		{
			"messages&.csv",
			"messages",
			"csv",
			false,
		},
		{
			"messages.toml",
			"messages",
			"csv",
			false,
		},
		{
			"messages.csv",
			"messages",
			"toml",
			false,
		},
		{
			"messages",
			"messages",
			"",
			true,
		},
		{
			"messages",
			"messages.",
			"",
			false,
		},
		{
			"messages",
			"messages.messages",
			"",
			false,
		},
	}

	for _, c := range cases {
		got := isFileMatch(c.fileName, c.nameTerm, c.ext)
		if got != c.want {
			t.Errorf("isFileMatch(%s, %s, $s) == %v, want %v", c.fileName, c.nameTerm, c.ext, c.want)
		}
	}
}
