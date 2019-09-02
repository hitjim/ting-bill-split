package main

import (
	"encoding/csv"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"

	"github.com/BurntSushi/toml"
	"github.com/jung-kurt/gofpdf"
	"github.com/shopspring/decimal"
)

type bill struct {
	Description    string   `toml:"description"`
	DeviceIds      []string `toml:"deviceIds"`
	ShortStrawID   string   `toml:"shortStrawId"`
	Total          float64  `toml:"total"`
	Devices        float64  `toml:"devices"`
	Minutes        float64  `toml:"minutes"`
	Messages       float64  `toml:"messages"`
	Megabytes      float64  `toml:"megabytes"`
	ExtraMinutes   float64  `toml:"extraMinutes"`
	ExtraMessages  float64  `toml:"extraMessages"`
	ExtraMegabytes float64  `toml:"extraMegabytes"`
	Fees           float64  `toml:"fees"`
}

// Used to contain all subtotals for a monthly bill.
// MinuteCosts, MessageCosts, MegabyteCosts are maps of decimal.Decimal totals.
// They are split by bill.DeviceIds and calculated by usage in parseMaps.
// SharedCosts reflect the rest of the items not based on usage, which get split evenly across all deviceIds
// TODO: finish these comments
type billSplit struct {
	MinuteCosts     map[string]decimal.Decimal
	MinuteQty       map[string]int
	MinutePercent   map[string]decimal.Decimal
	MessageCosts    map[string]decimal.Decimal
	MessageQty      map[string]int
	MessagePercent  map[string]decimal.Decimal
	MegabyteCosts   map[string]decimal.Decimal
	MegabyteQty     map[string]int
	MegabytePercent map[string]decimal.Decimal
	SharedCosts     map[string]decimal.Decimal
}

type totals struct {
	TotalMinutes   int
	TotalMessages  int
	TotalMegabytes int
}

func parseMaps(min map[string]int, msg map[string]int, meg map[string]int, bil bill) (billSplit, error) {
	bs := billSplit{
		make(map[string]decimal.Decimal),
		make(map[string]int),
		make(map[string]decimal.Decimal),
		make(map[string]decimal.Decimal),
		make(map[string]int),
		make(map[string]decimal.Decimal),
		make(map[string]decimal.Decimal),
		make(map[string]int),
		make(map[string]decimal.Decimal),
		make(map[string]decimal.Decimal),
	}
	var usedMin, usedMsg, usedMeg int
	DecimalPrecision := int32(6)

	bilMinutes := decimal.NewFromFloat(bil.Minutes + bil.ExtraMinutes)
	bilMessages := decimal.NewFromFloat(bil.Messages + bil.ExtraMessages)
	bilMegabytes := decimal.NewFromFloat(bil.Megabytes + bil.ExtraMegabytes)
	delta := decimal.NewFromFloat(bil.Devices + bil.Fees).Round(DecimalPrecision)
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

	totalMin := decimal.New(int64(usedMin), DecimalPrecision)
	totalMsg := decimal.New(int64(usedMsg), DecimalPrecision)
	totalMeg := decimal.New(int64(usedMeg), DecimalPrecision)

	for _, id := range bil.DeviceIds {
		subMin := decimal.New(int64(min[id]), DecimalPrecision)
		bs.MinutePercent[id] = subMin.Div(totalMin)
		bs.MinuteCosts[id] = bs.MinutePercent[id].Mul(bilMinutes)
		// It's possible for a device to still be on the bill, but not show any usage data
		if value, exists := min[id]; exists {
			bs.MinuteQty[id] = value
		} else {
			bs.MinuteQty[id] = 0
		}

		subMsg := decimal.New(int64(msg[id]), DecimalPrecision)
		bs.MessagePercent[id] = subMsg.DivRound(totalMsg, DecimalPrecision)
		bs.MessageCosts[id] = bs.MessagePercent[id].Mul(bilMessages)
		// It's possible for a device to still be on the bill, but not show any usage data
		if value, exists := msg[id]; exists {
			bs.MessageQty[id] = value
		} else {
			bs.MessageQty[id] = 0
		}

		subMeg := decimal.New(int64(meg[id]), DecimalPrecision)
		bs.MegabytePercent[id] = subMeg.DivRound(totalMeg, DecimalPrecision)
		bs.MegabyteCosts[id] = bs.MegabytePercent[id].Mul(bilMegabytes)
		// It's possible for a device to still be on the bill, but not show any usage data
		if value, exists := meg[id]; exists {
			bs.MegabyteQty[id] = value
		} else {
			bs.MegabyteQty[id] = 0
		}

		bs.SharedCosts[id] = delta.DivRound(deviceQty, DecimalPrecision)
	}

	minSubSum := decimal.New(0, DecimalPrecision)
	for _, sub := range bs.MinuteCosts {
		minSubSum = minSubSum.Add(sub)
	}

	minSubExtra := bilMinutes.Sub(minSubSum)
	if minSubExtra.GreaterThan(decimal.New(0, DecimalPrecision)) {
		fmt.Printf("Remainder minutes cost of $%s to deviceId %s\n", minSubExtra.String(), bil.ShortStrawID)
		bs.MinuteCosts[bil.ShortStrawID].Add(minSubExtra)
	} else {
		fmt.Println("There was no remainder cost when splitting minutes.")
	}

	msgSubSum := decimal.New(0, DecimalPrecision)
	for _, sub := range bs.MessageCosts {
		msgSubSum = msgSubSum.Add(sub)
	}

	msgSubExtra := bilMessages.Sub(msgSubSum)
	if msgSubExtra.GreaterThan(decimal.New(0, DecimalPrecision)) {
		fmt.Printf("Remainder messages cost of $%s to deviceId %s\n", msgSubExtra.String(), bil.ShortStrawID)
		bs.MessageCosts[bil.ShortStrawID].Add(msgSubExtra)
	} else {
		fmt.Println("There was no remainder cost when splitting messages.")
	}

	megSubSum := decimal.New(0, DecimalPrecision)
	for _, sub := range bs.MegabyteCosts {
		megSubSum = megSubSum.Add(sub)
	}

	megSubExtra := bilMessages.Sub(megSubSum)
	if megSubExtra.GreaterThan(decimal.New(0, DecimalPrecision)) {
		fmt.Printf("Remainder megabytes cost of $%s to deviceId %s\n", megSubExtra.String(), bil.ShortStrawID)
		bs.MegabyteCosts[bil.ShortStrawID].Add(megSubExtra)
	} else {
		fmt.Println("There was no remainder cost when splitting megabytes.")
	}

	deltaSubSum := decimal.New(0, DecimalPrecision)
	for _, sub := range bs.SharedCosts {
		deltaSubSum = deltaSubSum.Add(sub)
	}

	deltaSubExtra := delta.Sub(deltaSubSum)
	if deltaSubExtra.GreaterThan(decimal.New(0, DecimalPrecision)) {
		fmt.Printf("Remainder delta cost of $%s added to deviceId %s\n", deltaSubExtra.String(), bil.ShortStrawID)
		bs.SharedCosts[bil.ShortStrawID] = bs.SharedCosts[bil.ShortStrawID].Add(deltaSubExtra)
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
		if len(args) == 2 {
			newDirName = args[1]
		}
		if _, err := os.Stat(newDirName); os.IsNotExist(err) {
			fmt.Println("Creating a directory for a new billing period.")
			os.MkdirAll(newDirName, os.ModeDir)
			createBillFile(newDirName)
			fmt.Printf("\n1. Enter values for the bill.toml file in new directory `%s`\n", newDirName)
			fmt.Println("2. Add csv files for minutes, message, megabytes in the new directory")
			fmt.Printf("3. run `ting-bill-split %s`\n", newDirName)
		}
	}
}

func createBillFile(path string) {
	path += "/bill.toml"
	f, err := os.Create(path)

	if err != nil {
		panic(err)
	}

	newBill := bill{
		Description:    "Ting Bill YYYY-MM-DD",
		DeviceIds:      []string{"1112223333", "2229998888", "etc"},
		ShortStrawID:   "1112223333",
		Total:          0.00,
		Devices:        0.00,
		Minutes:        0.00,
		Messages:       0.00,
		Megabytes:      0.00,
		ExtraMinutes:   0.00,
		ExtraMessages:  0.00,
		ExtraMegabytes: 0.00,
		Fees:           0.00,
	}

	if err := toml.NewEncoder(f).Encode(newBill); err != nil {
		log.Fatalf("Error encoding TOML: %s", err)
	}
}

// For a fileName string, return true if it contains the nameTerm anywhere.
// If an empty string is provided for `ext`, no extension matching is performed.
// Otherwise additional file extension matching is performed.
// TODO LATER - maybe use path/filepath.Match
func isFileMatch(fileName string, nameTerm string, ext string) bool {
	r := regexp.MustCompile(`(?i)^[\w-]*` + nameTerm + `[\w-]*$`)

	if ext != "" {
		r = regexp.MustCompile(`(?i)^[\w-]*` + nameTerm + `[\w-]*(\.` + ext + `)$`)
	}

	return r.MatchString(fileName)
}

func parseDir(path string) {
	var billFile *os.File
	var minFile *os.File
	var msgFile *os.File
	var megFile *os.File

	files, err := ioutil.ReadDir(path)

	if err != nil {
		log.Fatal(err)
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		if billFile == nil && isFileMatch(file.Name(), "bill", "toml") {
			billFile, err = os.Open(file.Name())
			if err != nil {
				log.Fatal(err)
			}
		}

		if minFile == nil && isFileMatch(file.Name(), "minutes", "csv") {
			minFile, err = os.Open(file.Name())
			if err != nil {
				log.Fatal(err)
			}
		}

		if msgFile == nil && isFileMatch(file.Name(), "messages", "csv") {
			msgFile, err = os.Open(file.Name())
			if err != nil {
				log.Fatal(err)
			}
		}

		if megFile == nil && isFileMatch(file.Name(), "megabytes", "csv") {
			megFile, err = os.Open(file.Name())
			if err != nil {
				log.Fatal(err)
			}
		}
	}

	if billFile == nil || minFile == nil || msgFile == nil || megFile == nil {
		fmt.Println("Unable to open necessary files.")

		if billFile == nil {
			fmt.Println("Bill file not found.")
			return
		}

		if minFile == nil {
			fmt.Println("Minutes file not found.")
			return
		}

		if msgFile == nil {
			fmt.Println("Messages file not found.")
			return
		}

		if megFile == nil {
			fmt.Println("Megabytes file not found.")
			return
		}
	} else {
		fmt.Printf("\nRunning calculations based on files in directory: %s\n\n", path)

		billData, err := parseBill(billFile)
		if err != nil {
			log.Fatal(err)
		}

		minMap, err := parseMinutes(minFile)
		if err != nil {
			log.Fatal(err)
		}

		msgMap, err := parseMessages(msgFile)
		if err != nil {
			log.Fatal(err)
		}

		megMap, err := parseMegabytes(megFile)
		if err != nil {
			log.Fatal(err)
		}

		//TODO take in each map and return a billSplit
		split, err := parseMaps(minMap, msgMap, megMap, billData)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(split)

		pdfFilePath := filepath.Join(path, billData.Description+".pdf")

		invoiceNames, err := generatePDF(split, billData, pdfFilePath)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("Invoice generation complete: %s", invoiceNames)
	}
}

func generatePDF(bs billSplit, b bill, filePath string) (string, error) {
	fmt.Printf("Generating invoice %s\n\n", filePath)
	RoundPrecision := int32(2)

	// weightedTableHeading := []string{"Cost Type", "Minutes", "Messages", "Data"}
	// sharedTableHeading := []string{"Cost Type", "Amount"}
	// costsSplitHeading := []string{"Phone Number", "Nickname", "$Min", "$Msg", "$Data"}

	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()
	pdf.SetFont("Arial", "B", 10)
	pdf.SetXY(10, 20)

	// Table 0: Heading - 7 rows
	// heading: Invoice filename w/date, Device qty, Bill Total, Split Total
	//   (for comparison), Usage subtotal, Devices Subtotal, Tax+Reg subtotal"
	headingTable := func(b bill, bs billSplit) {
		pageHeading := []string{"Invoice with date", "Devices Qty", "$Total", "$Calc", "$Usage", "$Devices", "$Tax+Reg"}
		w := []float64{40.0, 25.0, 20.0, 20.0, 20.0, 20.0, 20.0}

		// Print heading
		for j, str := range pageHeading {
			pdf.CellFormat(w[j], 7, str, "1", 0, "C", false, 0, "")
		}
		pdf.Ln(-1)

		// Prep data
		minCosts := decimal.New(0, 1)
		msgCosts := decimal.New(0, 1)
		megCosts := decimal.New(0, 1)
		shrCosts := decimal.New(0, 1)

		for _, v := range bs.MinuteCosts {
			minCosts = minCosts.Add(v)
		}

		for _, v := range bs.MessageCosts {
			msgCosts = msgCosts.Add(v)
		}
		for _, v := range bs.MegabyteCosts {
			megCosts = megCosts.Add(v)
		}
		for _, v := range bs.SharedCosts {
			shrCosts = shrCosts.Add(v)
		}

		calcCost := decimal.Sum(minCosts, msgCosts, megCosts, shrCosts).Round(RoundPrecision)
		usgCost := decimal.Sum(minCosts, msgCosts, megCosts).Round(RoundPrecision)

		values := []string{
			b.Description,
			strconv.Itoa(len(b.DeviceIds)),
			strconv.FormatFloat(b.Total, 'f', 2, 64),
			calcCost.StringFixed(2),
			usgCost.StringFixed(2),
			strconv.FormatFloat(b.Devices, 'f', 2, 64),
			strconv.FormatFloat(b.Fees, 'f', 2, 64),
		}

		pdf.SetX(10)

		// Print data
		for i, str := range values {
			pdf.CellFormat(w[i], 7, str, "1", 0, "C", false, 0, "")
		}
		pdf.Ln(-1)
	}
	headingTable(b, bs)

	// Table 1: Usage - 8 rows
	// heading: number, nickname?, min, msg, data (KB), min%, msg%, data%
	// Then entries for each number
	// then entry for "Total" under nickname, and rest of sums
	usageTable := func(b bill, bs billSplit) {
		type usageTableVals struct {
			owner      string
			minutes    string
			messages   string
			data       string
			percentMin string
			percentMsg string
			percentMeg string
		}

		usageTableHeading := []string{"Phone Number", "Owner", "Minutes", "Messages", "Data (KB)", "Min%", "Msg%", "Data%"}
		w := []float64{40.0, 30.0, 25.0, 25.0, 25.0, 15.0, 15.0, 15.0}
		pdf.SetXY(10, pdf.GetY()+5)

		// Print heading
		for i, str := range usageTableHeading {
			pdf.CellFormat(w[i], 7, str, "1", 0, "C", false, 0, "")
		}
		pdf.Ln(-1)

		// Prep data
		values := make(map[string]usageTableVals)

		for _, id := range b.DeviceIds {
			values[id] = usageTableVals{
				"fart",
				strconv.Itoa(bs.MinuteQty[id]),
				strconv.Itoa(bs.MessageQty[id]),
				strconv.Itoa(bs.MegabyteQty[id]),
				bs.MinutePercent[id].StringFixed(RoundPrecision),
				bs.MessagePercent[id].StringFixed(RoundPrecision),
				bs.MegabytePercent[id].StringFixed(RoundPrecision),
			}
		}

		// TODO: turn this into some kind of getMapKeys if it gets too crazy
		ids := make([]string, len(values))
		i := 0
		for k := range values {
			ids[i] = k
			i++
		}

		pdf.SetXY(10, pdf.GetY())

		//Print data
		var wi int
		for _, id := range ids {
			wi = 0
			rowVals := values[id]
			pdf.CellFormat(w[wi], 7, id, "1", 0, "C", false, 0, "")
			wi++
			pdf.CellFormat(w[wi], 7, rowVals.owner, "1", 0, "C", false, 0, "")
			wi++
			pdf.CellFormat(w[wi], 7, rowVals.minutes, "1", 0, "R", false, 0, "")
			wi++
			pdf.CellFormat(w[wi], 7, rowVals.messages, "1", 0, "R", false, 0, "")
			wi++
			pdf.CellFormat(w[wi], 7, rowVals.data, "1", 0, "R", false, 0, "")
			wi++
			pdf.CellFormat(w[wi], 7, rowVals.percentMin, "1", 0, "C", false, 0, "")
			wi++
			pdf.CellFormat(w[wi], 7, rowVals.percentMsg, "1", 0, "C", false, 0, "")
			wi++
			pdf.CellFormat(w[wi], 7, rowVals.percentMeg, "1", 0, "C", false, 0, "")
			pdf.SetXY(10, pdf.GetY()+7)
		}
		pdf.Ln(-1)
	}
	usageTable(b, bs)

	// Table 2: Weighted costs - 3 rows
	// heading: Weighted Costs: Minutes, Messages, Data
	// Base: $x, $y, $z
	// Extra: etc
	// Total: etc

	// Table 3: Shared costs - 2 rows
	// heading: Type, Amount

	// Table 4: Costs split - 7
	// heading: number, Nickname, Min, Msg, Data, Shared, Total
	// entry for each number

	// weightedTable := func(yPosition float64, cHeadings []string, bs billSplit) {
	// 	// Column widths
	// 	w := []float64{40.0, 35.0, 40.0, 45.0}
	// 	wSum := 0.0
	// 	for _, v := range w {
	// 		wSum += v
	// 	}
	// 	left := (210 - wSum) / 2
	// 	// Header
	// 	pdf.SetY(yPosition)
	// 	pdf.SetX(left)
	// 	for j, str := range header {
	// 		pdf.CellFormat(w[j], 7, str, "1", 0, "C", false, 0, "")
	// 	}
	// 	pdf.Ln(-1)
	// 	// Data
	// 	for _, c := range countryList {
	// 		pdf.SetX(left)
	// 		pdf.CellFormat(w[0], 6, c.nameStr, "LR", 0, "", false, 0, "")
	// 		pdf.CellFormat(w[1], 6, c.capitalStr, "LR", 0, "", false, 0, "")
	// 		pdf.CellFormat(w[2], 6, c.smellStr, "LR", 0, "R", false, 0, "")
	// 		pdf.CellFormat(w[3], 6, c.birdStr, "LR", 0, "R", false, 0, "")
	// 		pdf.Ln(-1)
	// 	}
	// 	pdf.SetX(left)
	// 	pdf.CellFormat(wSum, 0, "", "T", 0, "", false, 0, "")
	// }

	err := pdf.OutputFileAndClose(filePath)

	// TODO LATER - add dates to bill. For now, entering manually in the "description" field in bill.toml
	// Future: generate a range off min/max dates in usage files?
	// Or maybe just have a new field in toml?

	// TODO make this take a path in, for dir-mode splitting

	return filePath, err
}

func main() {
	fmt.Printf("Ting Bill Splitter\n\n")

	billPtr := flag.String("bill", "", "filename for bill toml - ex: -bill=\"bill.toml\"")
	minPtr := flag.String("minutes", "", "filename for minutes csv - ex: -minutes=\"minutes.csv\"")
	msgPtr := flag.String("messages", "", "filename for messages csv - ex: -messages=\"messages.csv\"")
	megPtr := flag.String("megabytes", "", "filename for megabytes csv - ex: -megabytes=\"megabytes.csv\"")

	flag.Parse()
	args := flag.Args()
	targetDir := "."

	if len(args) > 0 {
		fmt.Println("Running in batch mode")

		command := args[0]

		switch command {
		case "new":
			createNewBillingDir(args)
		case "dir":
			if len(args) > 1 {
				targetDir = args[1]
			}
			parseDir(targetDir)
			// TODO - make parseDir return a split, since non-dir parsing uses a split
			//   then handle all actions after the if/else
		default:
			fmt.Println("Use `ting-bill-split new` or `new <billing-directory>` to create a new billing directory")
			fmt.Println("Use `ting-bill-split dir <billing-directory>` to run on a directory containing a `bill.toml`, and CSV files for minutes, messages, and megabytes usage.")
			fmt.Println("  Each of these files must contain their type somewhere in the filename - i.e. `YYYYMMDD-messages.csv` or `messages-potatosalad.csv` or whatever.")
			fmt.Printf("\n... or `-h` for flag options")
		}
	} else {
		fmt.Println("Running with with individual file assignments")

		badParam := false
		paramMap := map[string]*string{
			"bill":      billPtr,
			"minutes":   minPtr,
			"messages":  msgPtr,
			"megabytes": megPtr,
		}

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

		msgMap, err := parseMessages(msgFile)
		if err != nil {
			log.Fatal(err)
		}

		megMap, err := parseMegabytes(megFile)
		if err != nil {
			log.Fatal(err)
		}

		//TODO take in each map and return a billSplit
		split, err := parseMaps(minMap, msgMap, megMap, billData)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(split)
	}
}
