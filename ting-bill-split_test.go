package main

import (
	"reflect"
	"strings"
	"testing"
)

func TestParseMinutes(t *testing.T) {
	cases := []struct {
		in   string
		want map[string]int
	}{
		{
			"Date,Time,Incoming/Outgoing,Phone,Nickname,Location,Country,Partner's Phone,Partner Nickname,Partner's Location,Partner's Country,Duration (min),Surcharges ($),Features\n" +
				"\"February 03, 2011\",01:11,outgoing,1112223333,Phone 1,\"SPRINGFIELD, MO\",USA,7778889999,,USA,United States of America,1,0.0,\"\"\n" +
				"\"February 14, 2011\",01:22,outgoing,1112223333,Phone 1,\"SPRINGFIELD, MO\",USA,7778889999,,USA,United States of America,2,0.0,\"\"\n" +
				"\"February 14, 2011\",01:22,outgoing,1112224444,Phone 2,\"DELANO, KS\",USA,7778889999,,USA,United States of America,1,0.0,\"\"\n",
			map[string]int{
				"1112223333": 3,
				"1112224444": 1,
			},
		},
	}

	for _, c := range cases {
		got := parseMinutes(strings.NewReader(c.in))
		eq := reflect.DeepEqual(got, c.want)
		if !eq {
			t.Errorf("parseMinutes(%q) == %q, want %q", c.in, got, c.want)
		}
	}
}
