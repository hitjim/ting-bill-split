package main

import (
	"strings"
	"testing"

	"github.com/hitjim/ting-bill-split/internal/tingbill"

	"github.com/google/go-cmp/cmp"
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
		if !cmp.Equal(got, c.want) {
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
		if !cmp.Equal(got, c.want) {
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
		if !cmp.Equal(got, c.want) {
			t.Errorf("parseMegabytes(%v) == %v, want %v", c.in, got, c.want)
		}
	}
}

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

func TestParseBill(t *testing.T) {
	cases := []struct {
		in   string
		want tingbill.Bill
	}{
		{
			`description = "First test desc"
minutes = 35.00
messages = 8.00
megabytes = 20.00
devicesCost = 42.00
extraMinutes = 1.00
extraMessages = 2.00
extraMegabytes = 3.00
total = 118.84
fees = 12.85
shortStrawId = "1112223333"

[[devices]]
deviceId = "1112223333"
owner = "owner1"

[[devices]]
deviceId = "1112224444"
owner = "owner2"

[[devices]]
deviceId = "1112220000"
owner = "owner2"`,
			tingbill.Bill{
				Description:    "First test desc",
				DevicesCost:    42.00,
				Minutes:        35.00,
				Messages:       8.00,
				Megabytes:      20.00,
				ExtraMinutes:   1.00,
				ExtraMessages:  2.00,
				ExtraMegabytes: 3.00,
				Fees:           12.85,
				Devices: []tingbill.Device{
					tingbill.Device{
						DeviceID: "1112223333",
						Owner:    "owner1",
					},
					tingbill.Device{
						DeviceID: "1112224444",
						Owner:    "owner2",
					},
					tingbill.Device{
						DeviceID: "1112220000",
						Owner:    "owner2",
					},
				},
				ShortStrawID: "1112223333",
				Total:        118.84,
			},
		},
	}

	for _, c := range cases {
		got, err := parseBill(strings.NewReader(c.in))
		if err != nil {
			t.Errorf("parseBill(%v) err, %v", c.in, err)
		}
		if !cmp.Equal(got, c.want) {
			t.Errorf("parseBill(%v) == %v, want %v", c.in, got, c.want)
		}
	}
}

func TestParseMaps(t *testing.T) {
	DecimalPrecision := int32(6)
	cases := []struct {
		min  map[string]int
		msg  map[string]int
		meg  map[string]int
		bil  tingbill.Bill
		want tingbill.BillSplit
	}{
		{
			map[string]int{
				"1112223333": 4,
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
			tingbill.Bill{
				Description:    "TestParseMaps",
				DevicesCost:    42.00,
				Minutes:        35.00,
				Messages:       8.00,
				Megabytes:      20.00,
				ExtraMinutes:   1.00,
				ExtraMessages:  2.00,
				ExtraMegabytes: 3.00,
				Fees:           12.85,
				Devices: []tingbill.Device{
					tingbill.Device{
						DeviceID: "1112223333",
						Owner:    "owner1",
					},
					tingbill.Device{
						DeviceID: "1112224444",
						Owner:    "owner2",
					},
					tingbill.Device{
						DeviceID: "1112220000",
						Owner:    "owner1",
					},
				},
				ShortStrawID: "1112220000",
				Total:        118.84,
			},
			tingbill.BillSplit{
				MinuteCosts: map[string]decimal.Decimal{
					"1112220000": decimal.NewFromFloat(0).Round(DecimalPrecision),
					"1112223333": decimal.NewFromFloat(28.8).Round(DecimalPrecision),
					"1112224444": decimal.NewFromFloat(7.2).Round(DecimalPrecision),
				},
				MinuteQty: map[string]int{
					"1112220000": 0,
					"1112223333": 4,
					"1112224444": 1,
				},
				MinutePercent: map[string]decimal.Decimal{
					"1112220000": decimal.NewFromFloat(0),
					"1112223333": decimal.NewFromFloat(0.8),
					"1112224444": decimal.NewFromFloat(0.2),
				},
				MessageCosts: map[string]decimal.Decimal{
					"1112220000": decimal.NewFromFloat(0).Round(DecimalPrecision),
					"1112223333": decimal.NewFromFloat(7.54014).Round(DecimalPrecision),
					"1112224444": decimal.NewFromFloat(2.45986).Round(DecimalPrecision),
				},
				MessagePercent: map[string]decimal.Decimal{
					"1112220000": decimal.NewFromFloat(0),
					"1112223333": decimal.NewFromFloat(0.754014),
					"1112224444": decimal.NewFromFloat(0.245986),
				},
				MessageQty: map[string]int{
					"1112220000": 0,
					"1112223333": 4696,
					"1112224444": 1532,
				},
				MegabyteCosts: map[string]decimal.Decimal{
					"1112220000": decimal.NewFromFloat(0).Round(DecimalPrecision),
					"1112223333": decimal.NewFromFloat(16.729372).Round(DecimalPrecision),
					"1112224444": decimal.NewFromFloat(6.270628).Round(DecimalPrecision),
				},
				MegabytePercent: map[string]decimal.Decimal{
					"1112220000": decimal.NewFromFloat(0),
					"1112223333": decimal.NewFromFloat(0.727364),
					"1112224444": decimal.NewFromFloat(0.272636),
				},
				MegabyteQty: map[string]int{
					"1112220000": 0,
					"1112223333": 8001,
					"1112224444": 2999,
				},
				SharedCosts: map[string]decimal.Decimal{
					"1112223333": decimal.NewFromFloat(18.283333),
					"1112224444": decimal.NewFromFloat(18.283333),
					"1112220000": decimal.NewFromFloat(18.283334),
				},
			},
		},
	}

	for _, c := range cases {
		got, err := parseMaps(c.min, c.msg, c.meg, c.bil)
		if err != nil {
			t.Errorf("parseMaps(%v, %v, %v, %v) err, %v", c.min, c.msg, c.meg, c.bil, err)
		}
		if !cmp.Equal(got, c.want) {
			t.Errorf("parseMaps(%v, %v, %v, %v) == %v, want %v", c.min, c.msg, c.meg, c.bil, got, c.want)
		}
	}
}
