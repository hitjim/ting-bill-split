package main

import (
	"reflect"
	"strings"
	"testing"

	"github.com/shopspring/decimal"
)

func TestParseMinutes(t *testing.T) {
	cases := []struct {
		in   string
		want map[string]int
	}{
		{
			`Date,Time,Incoming/Outgoing,Phone,Nickname,Location,Country,Partner's Phone,Partner Nickname,Partner's Location,Partner's Country,Duration (min),Surcharges ($),Features
"February 03, 2011",01:11,outgoing,1112223333,Phone 1,"SPRINGFIELD, MO",USA,7778889999,,USA,United States of America,1,0.0,""
"February 14, 2011",01:22,outgoing,1112223333,Phone 1,"SPRINGFIELD, MO",USA,7778889999,,USA,United States of America,2,0.0,""
"February 14, 2011",01:22,outgoing,1112224444,Phone 2,"DELANO, KS",USA,7778889999,,USA,United States of America,1,0.0,""`,
			map[string]int{
				"1112223333": 3,
				"1112224444": 1,
			},
		},
	}

	for _, c := range cases {
		got, err := parseMinutes(strings.NewReader(c.in))
		if err != nil {
			t.Errorf("parseMinutes(%v) err, %v", c.in, err)
		}
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("parseMinutes(%v) == %v, want %v", c.in, got, c.want)
		}
	}
}

func TestParseMessages(t *testing.T) {
	cases := []struct {
		in   string
		want map[string]int
	}{
		{
			`Date,Time,Phone,Nickname,Partner's Phone,Partner's Nickname,Sent/Received,Roaming,Roaming Country,Surcharges ($)
"February 03, 2011",01:11,1112223333,Phone 1,7778889999,Phone 7,sent,no,"",0.0
"February 03, 2011",01:12,1112223333,Phone 1,7778889999,Phone 7,received,no,"",0.0
"February 03, 2011",01:12,1112224444,Phone 1,7778889999,Phone 7,received,no,"",0.0`,
			map[string]int{
				"1112223333": 2,
				"1112224444": 1,
			},
		},
	}

	for _, c := range cases {
		got, err := parseMessages(strings.NewReader(c.in))
		if err != nil {
			t.Errorf("parseMessages(%v) err, %v", c.in, err)
		}
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("parseMessages(%v) == %v, want %v", c.in, got, c.want)
		}
	}
}

func TestParseMegabytes(t *testing.T) {
	cases := []struct {
		in   string
		want map[string]int
	}{
		{
			`Date,Device,Nickname,Location,Kilobytes,Surcharges ($),Type
"February 03, 2011",1112223333,Phone 1,United States of America,1336,0.0,4G LTE
"February 03, 2011",1112223333,Phone 1,United States of America,2024,0.0,3G
"February 04, 2011",1112223333,Phone 1,United States of America,1336,0.0,4G LTE
"February 04, 2011",1112224444,Phone 2,United States of America,1532,0.0,4G LTE`,
			map[string]int{
				"1112223333": 4696,
				"1112224444": 1532,
			},
		},
	}

	for _, c := range cases {
		got, err := parseMegabytes(strings.NewReader(c.in))
		if err != nil {
			t.Errorf("parseMegabytes(%v) err, %v", c.in, err)
		}
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("parseMegabytes(%v) == %v, want %v", c.in, got, c.want)
		}
	}
}

func TestParseBill(t *testing.T) {
	cases := []struct {
		in   string
		want bill
	}{
		{
			`minutes = 35.00
messages = 8.00
megabytes = 20.00
devices = 42.00
extras = 1.00
fees = 12.84
deviceIds = [ "1112223333", "1112224444", "1112220000" ]
shortStrawId = "1112220000"`,
			bill{
				Minutes:      35.00,
				Messages:     8.00,
				Megabytes:    20.00,
				Devices:      42.00,
				Extras:       1.00,
				Fees:         12.84,
				DeviceIds:    []string{"1112223333", "1112224444", "1112220000"},
				ShortStrawId: "1112220000",
			},
		},
		{
			`minutes = 35.00
messages = 8.00
megabytes = 20.00
devices = 42.00
extras = 1.00
fees = 12.84
deviceIds = [ "1112223333", "1112224444", "1112220000" ]
shortStrawId = "wrongnumber"`,
			bill{
				Minutes:      35.00,
				Messages:     8.00,
				Megabytes:    20.00,
				Devices:      42.00,
				Extras:       1.00,
				Fees:         12.84,
				DeviceIds:    []string{"1112223333", "1112224444", "1112220000"},
				ShortStrawId: "1112223333",
			},
		},
		{
			`minutes = 35.00
messages = 8.00
megabytes = 20.00
devices = 42.00
extras = 1.00
fees = 12.84
deviceIds = [ "1112223333", "1112224444", "1112220000" ]`,
			bill{
				Minutes:      35.00,
				Messages:     8.00,
				Megabytes:    20.00,
				Devices:      42.00,
				Extras:       1.00,
				Fees:         12.84,
				DeviceIds:    []string{"1112223333", "1112224444", "1112220000"},
				ShortStrawId: "1112223333",
			},
		},
	}

	for _, c := range cases {
		got, err := parseBill(strings.NewReader(c.in))
		if err != nil {
			t.Errorf("parseBill(%v) err, %v", c.in, err)
		}
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("parseBill(%v) == %v, want %v", c.in, got, c.want)
		}
	}
}

func TestParseMaps(t *testing.T) {
	cases := []struct {
		min  map[string]int
		msg  map[string]int
		meg  map[string]int
		bil  bill
		want billSplit
	}{
		{
			map[string]int{
				"1112223333": 3,
				"1112224444": 1,
			},
			map[string]int{
				"1112223333": 4696,
				"1112224444": 1532,
			},
			map[string]int{
				"1112223333": 8001,
				"1112224444": 2999,
			},
			bill{
				Minutes:      35.00,
				Messages:     8.00,
				Megabytes:    20.00,
				Devices:      42.00,
				Extras:       1.00,
				Fees:         12.84,
				DeviceIds:    []string{"1112223333", "1112224444", "1112220000"},
				ShortStrawId: "1112220000",
				Total:        118.84,
			},
			billSplit{
				MinSubs: map[string]decimal.Decimal{
					"1112220000": decimal.NewFromFloat(0),
					"1112223333": decimal.NewFromFloat(26.25),
					"1112224444": decimal.NewFromFloat(8.75),
				},
				MsgSubs: map[string]decimal.Decimal{
					"1112220000": decimal.NewFromFloat(0),
					"1112223333": decimal.NewFromFloat(6.03),
					"1112224444": decimal.NewFromFloat(1.97),
				},
				MegSubs: map[string]decimal.Decimal{
					"1112220000": decimal.NewFromFloat(0),
					"1112223333": decimal.NewFromFloat(14.55),
					"1112224444": decimal.NewFromFloat(5.45),
				},
				DeltaSubs: map[string]decimal.Decimal{
					"1112223333": decimal.NewFromFloat(18.61),
					"1112224444": decimal.NewFromFloat(18.61),
					"1112220000": decimal.NewFromFloat(18.62),
				},
			},
		},
	}

	for _, c := range cases {
		got, err := parseMaps(c.min, c.msg, c.meg, c.bil)
		if err != nil {
			t.Errorf("parseMaps(%v, %v, %v, %v) err, %v", c.min, c.msg, c.meg, c.bil, err)
		}
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("parseMaps(%v, %v, %v, %v) == %v, want %v", c.min, c.msg, c.meg, c.bil, got, c.want)
		}
	}
}
