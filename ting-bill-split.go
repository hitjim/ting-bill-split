package main

import (
	"encoding/csv"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"

	"github.com/BurntSushi/toml"
	"github.com/shopspring/decimal"
)

type bill struct {
	Minutes      float64  `toml:"minutes"`
	Messages     float64  `toml:"messages"`
	Megabytes    float64  `toml:"megabytes"`
	Devices      float64  `toml:"devices"`
	Extras       float64  `toml:"extras"`
	Fees         float64  `toml:"fees"`
	DeviceIds    []string `toml:"deviceIds"`
	ShortStrawID string   `toml:"shortStrawId"`
	Total        float64  `toml:"total"`
}

// Used to contain all subtotals for a monthly bill.
// MinSubs, MsgSubs, MegSubs are maps of decimal.Decimal totals.
// They are split by bill.DeviceIds and calculated by usage in parseMaps.
// DeltaSubs reflect the rest of the items not based on usage, which get split evenly across all deviceIds
type billSplit struct {
	MinSubs   map[string]decimal.Decimal
	MsgSubs   map[string]decimal.Decimal
	MegSubs   map[string]decimal.Decimal
	DeltaSubs map[string]decimal.Decimal
}

func parseMaps(min map[string]int, msg map[string]int, meg map[string]int, bil bill) (billSplit, error) {
	bs := billSplit{
		make(map[string]decimal.Decimal),
		make(map[string]decimal.Decimal),
		make(map[string]decimal.Decimal),
		make(map[string]decimal.Decimal),
	}
	var usedMin, usedMsg, usedMeg int
	DecimalPrecision := int32(6)
	RoundPrecision := int32(2)

	bilMinutes := decimal.NewFromFloat(bil.Minutes)
	bilMessages := decimal.NewFromFloat(bil.Messages)
	bilMegabytes := decimal.NewFromFloat(bil.Megabytes)
	delta := decimal.NewFromFloat(bil.Devices + bil.Extras + bil.Fees)
	deviceQty := decimal.New(int64(len(bil.DeviceIds)), 0)

	// Calculate usage totals
	for _, v := range min {
		usedMin += v
	}

	for _, v := range msg {
		usedMsg += v
	}

	for _, v := range meg {
		usedMeg += v
	}

	for _, id := range bil.DeviceIds {
		subMin := decimal.New(int64(min[id]), DecimalPrecision)
		totalMin := decimal.New(int64(usedMin), DecimalPrecision)
		percentMin := subMin.DivRound(totalMin, DecimalPrecision)
		bs.MinSubs[id] = percentMin.Mul(bilMinutes).Round(RoundPrecision)

		subMsg := decimal.New(int64(msg[id]), DecimalPrecision)
		totalMsg := decimal.New(int64(usedMsg), DecimalPrecision)
		percentMsg := subMsg.DivRound(totalMsg, DecimalPrecision)
		bs.MsgSubs[id] = percentMsg.Mul(bilMessages).Round(RoundPrecision)

		subMeg := decimal.New(int64(meg[id]), DecimalPrecision)
		totalMeg := decimal.New(int64(usedMeg), DecimalPrecision)
		percentMeg := subMeg.DivRound(totalMeg, DecimalPrecision)
		bs.MegSubs[id] = percentMeg.Mul(bilMegabytes).Round(RoundPrecision)

		bs.DeltaSubs[id] = delta.DivRound(deviceQty, RoundPrecision)
	}

	minSubSum := decimal.New(0, RoundPrecision)
	for _, sub := range bs.MinSubs {
		minSubSum = minSubSum.Add(sub)
	}

	minSubExtra := bilMinutes.Sub(minSubSum)
	if minSubExtra.GreaterThan(decimal.New(0, RoundPrecision)) {
		fmt.Printf("Remainder minutes cost of $%s to deviceId %s\n", minSubExtra.String(), bil.ShortStrawID)
		bs.MinSubs[bil.ShortStrawID].Add(minSubExtra)
	} else {
		fmt.Println("There was no remainder cost when splitting minutes.")
	}

	msgSubSum := decimal.New(0, RoundPrecision)
	for _, sub := range bs.MsgSubs {
		msgSubSum = msgSubSum.Add(sub)
	}

	msgSubExtra := bilMessages.Sub(msgSubSum)
	if msgSubExtra.GreaterThan(decimal.New(0, RoundPrecision)) {
		fmt.Printf("Remainder messages cost of $%s to deviceId %s\n", msgSubExtra.String(), bil.ShortStrawID)
		bs.MsgSubs[bil.ShortStrawID].Add(msgSubExtra)
	} else {
		fmt.Println("There was no remainder cost when splitting messages.")
	}

	megSubSum := decimal.New(0, RoundPrecision)
	for _, sub := range bs.MegSubs {
		megSubSum = megSubSum.Add(sub)
	}

	megSubExtra := bilMessages.Sub(megSubSum)
	if megSubExtra.GreaterThan(decimal.New(0, RoundPrecision)) {
		fmt.Printf("Remainder megabytes cost of $%s to deviceId %s\n", megSubExtra.String(), bil.ShortStrawID)
		bs.MegSubs[bil.ShortStrawID].Add(megSubExtra)
	} else {
		fmt.Println("There was no remainder cost when splitting megabytes.")
	}

	deltaSubSum := decimal.New(0, RoundPrecision)
	for _, sub := range bs.DeltaSubs {
		deltaSubSum = deltaSubSum.Add(sub)
	}

	deltaSubExtra := delta.Sub(deltaSubSum)
	if deltaSubExtra.GreaterThan(decimal.New(0, RoundPrecision)) {
		fmt.Printf("Remainder delta cost of $%s added to deviceId %s\n", deltaSubExtra.String(), bil.ShortStrawID)
		bs.DeltaSubs[bil.ShortStrawID] = bs.DeltaSubs[bil.ShortStrawID].Add(deltaSubExtra)
	} else {
		fmt.Println("There was no remainder cost when splitting delta.")
	}

	return bs, nil
}

func parseBill(r io.Reader) (bill, error) {
	var b bill
	if _, err := toml.DecodeReader(r, &b); err != nil {
		return bill{}, err
	}

	phoneIndex := sliceIndex(len(b.DeviceIds), func(i int) bool { return b.DeviceIds[i] == b.ShortStrawID })

	if phoneIndex < 0 {
		b.ShortStrawID = b.DeviceIds[0]
	}

	return b, nil
}

func checkParam(param string, ptr *string, badParam *bool) {
	if *ptr == "" {
		*badParam = true
		fmt.Printf("%s parameter is bad\n", param)
	}
}

func sliceIndex(limit int, predicate func(i int) bool) int {
	for i := 0; i < limit; i++ {
		if predicate(i) {
			return i
		}
	}
	return -1
}

func parseMinutes(minReader io.Reader) (map[string]int, error) {
	m := make(map[string]int)
	r := csv.NewReader(minReader)

	// Get index of important fields
	header, err := r.Read()

	if err == io.EOF {
		fmt.Println("minutes.csv is empty!")
		return m, err
	}

	if err != nil {
		fmt.Println("Error parsing minutes.csv")
		return m, err
	}

	phoneIndex := sliceIndex(len(header), func(i int) bool { return header[i] == "Phone" })

	if phoneIndex < 0 {
		return m, errors.New(`missing "Phone" header in minutes csv file`)
	}

	minIndex := sliceIndex(len(header), func(i int) bool { return header[i] == "Duration (min)" })

	if minIndex < 0 {
		return m, errors.New(`missing "Duration (min)" header in minutes csv file`)
	}

	for {
		record, err := r.Read()

		if err != nil {
			if err == io.EOF {
				break
			}
			return m, err
		}

		min, err := strconv.Atoi(record[minIndex])
		if err != nil {
			return m, err
		}

		phone := record[phoneIndex]

		m[phone] += min
	}

	return m, nil
}

func parseMessages(msgReader io.Reader) (map[string]int, error) {
	m := make(map[string]int)
	r := csv.NewReader(msgReader)

	// Get index of important fields
	header, err := r.Read()

	if err == io.EOF {
		fmt.Println("messages.csv is empty!")
		return m, err
	}

	if err != nil {
		fmt.Println("Error parsing messages.csv")
		return m, err
	}

	phoneIndex := sliceIndex(len(header), func(i int) bool { return header[i] == "Phone" })

	if phoneIndex < 0 {
		return m, errors.New(`missing "Phone" header in messages csv file`)
	}

	for {
		record, err := r.Read()

		if err != nil {
			if err == io.EOF {
				break
			}
			return m, err
		}

		phone := record[phoneIndex]

		m[phone]++
	}

	return m, nil
}

func parseMegabytes(megReader io.Reader) (map[string]int, error) {
	m := make(map[string]int)
	r := csv.NewReader(megReader)

	// Get index of important fields
	header, err := r.Read()

	if err == io.EOF {
		fmt.Println("megabytes.csv is empty!")
		return m, err
	}

	if err != nil {
		fmt.Println("Error parsing megabytes.csv")
		return m, err
	}

	phoneIndex := sliceIndex(len(header), func(i int) bool { return header[i] == "Device" })

	if phoneIndex < 0 {
		return m, errors.New(`missing "Device" header in megabytes csv file`)
	}

	kbIndex := sliceIndex(len(header), func(i int) bool { return header[i] == "Kilobytes" })

	if kbIndex < 0 {
		return m, errors.New(`missing "Kilobytes" header in megabytes csv file`)
	}

	for {
		record, err := r.Read()

		if err != nil {
			if err == io.EOF {
				break
			}
			return m, err
		}

		kb, err := strconv.Atoi(record[kbIndex])
		if err != nil {
			return m, err
		}

		phone := record[phoneIndex]

		m[phone] += kb
	}

	return m, nil
}

func createNewBillingDir(args []string) {
	newDirName := "new-billing-period"
	if len(args) > 2 {
		fmt.Println("Syntax: `new <dir-name>`")
	} else {
		newDirName = args[1]
		fmt.Printf("newDirName is: %s", newDirName)
		if _, err := os.Stat(newDirName); os.IsNotExist(err) {
			fmt.Println("Creating a directory for a new billing period.")
			os.MkdirAll(newDirName, os.ModeDir)
		}
	}
}

func createBillsFile(path string) {
	newBills := bill{
		Minutes:      0.00,
		Messages:     0.00,
		Megabytes:    0.00,
		Devices:      0.00,
		Extras:       0.00,
		Fees:         0.00,
		DeviceIds:    []string{},
		ShortStrawID: "",
		Total:        0.00,
	}

	if err := toml.NewEncoder(os.Stdout).Encode(newBills); err != nil {
		log.Fatalf("Error encoding TOML: %s", err)
	}
}

func main() {
	fmt.Printf("Ting Bill Splitter\n\n")

	billPtr := flag.String("bills", "", "filename for bills toml - ex: -bills=\"bills.toml\"")
	minPtr := flag.String("minutes", "", "filename for minutes csv - ex: -minutes=\"minutes.csv\"")
	msgPtr := flag.String("messages", "", "filename for messages csv - ex: -messages=\"messages.csv\"")
	megPtr := flag.String("megabytes", "", "filename for megabytes csv - ex: -megabytes=\"megabytes.csv\"")

	badParam := false
	paramMap := map[string]*string{
		"bills":     billPtr,
		"minutes":   minPtr,
		"messages":  msgPtr,
		"megabytes": megPtr,
	}

	flag.Parse()
	args := flag.Args()

	if len(args) > 0 {
		fmt.Println("Running in batch mode")
		if args[0] == "new" {
			createNewBillingDir(args)
		} else {
			fmt.Println("Use `new` to create a new billing directory")
			fmt.Println("... or `-h` for flag options")
		}
	} else {
		fmt.Println("Running with with individual file assignments")
		for k, v := range paramMap {
			checkParam(k, v, &badParam)
		}

		if badParam {
			os.Exit(1)
		}

		billFile, err := os.Open(*billPtr)
		if err != nil {
			log.Fatal(err)
		}

		billData, err := parseBill(billFile)
		if err != nil {
			log.Fatal(err)
		}

		// TODO remove this once we do something useful with it.
		fmt.Println("billData ... something something ... *wanders off*")
		fmt.Println(billData)

		minFile, err := os.Open(*minPtr)
		if err != nil {
			log.Fatal(err)
		}

		msgFile, err := os.Open(*msgPtr)
		if err != nil {
			log.Fatal(err)
		}

		megFile, err := os.Open(*megPtr)
		if err != nil {
			log.Fatal(err)
		}

		minMap, err := parseMinutes(minFile)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(minMap)

		msgMap, err := parseMessages(msgFile)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(msgMap)

		megMap, err := parseMegabytes(megFile)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(megMap)

		//TODO take in each map and return a billSplit
		split, err := parseMaps(minMap, msgMap, megMap, billData)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(split)
	}
}
