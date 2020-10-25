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

	"github.com/hitjim/ting-bill-split/internal/tingbill"
	"github.com/hitjim/ting-bill-split/internal/tingpdf"

	"github.com/BurntSushi/toml"
	"github.com/shopspring/decimal"
)

func parseMaps(min map[string]int, msg map[string]int, meg map[string]int, bil tingbill.Bill) (tingbill.BillSplit, error) {
	bs := tingbill.BillSplit{
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
	delta := decimal.NewFromFloat(bil.DevicesCost + bil.Fees).Round(DecimalPrecision)
	deviceQty := decimal.New(int64(len(bil.Devices)), 0)

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

	deviceIds := bil.DeviceIds()

	for _, id := range deviceIds {
		subMin := decimal.New(int64(min[id]), DecimalPrecision)
		bs.MinutePercent[id] = subMin.Div(totalMin)
		bs.MinuteCosts[id] = bs.MinutePercent[id].Mul(bilMinutes)
		// It's possible for a device to still be on the Bill, but not show any usage data
		if value, exists := min[id]; exists {
			bs.MinuteQty[id] = value
		} else {
			bs.MinuteQty[id] = 0
		}

		subMsg := decimal.New(int64(msg[id]), DecimalPrecision)
		bs.MessagePercent[id] = subMsg.DivRound(totalMsg, DecimalPrecision)
		bs.MessageCosts[id] = bs.MessagePercent[id].Mul(bilMessages)
		// It's possible for a device to still be on the Bill, but not show any usage data
		if value, exists := msg[id]; exists {
			bs.MessageQty[id] = value
		} else {
			bs.MessageQty[id] = 0
		}

		subMeg := decimal.New(int64(meg[id]), DecimalPrecision)
		bs.MegabytePercent[id] = subMeg.DivRound(totalMeg, DecimalPrecision)
		bs.MegabyteCosts[id] = bs.MegabytePercent[id].Mul(bilMegabytes)
		// It's possible for a device to still be on the Bill, but not show any usage data
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

func parseBill(r io.Reader) (tingbill.Bill, error) {
	var b tingbill.Bill
	if _, err := toml.DecodeReader(r, &b); err != nil {
		return tingbill.Bill{}, err
	}

	ids := b.DeviceIds()

	// Check to see if a shortStrawId was set. If not, set it to first one we find.
	// Ordering is random, so deal with it.
	// TODO - make this more testable
	phoneIndex := sliceIndex(len(ids), func(i int) bool { return ids[i] == b.ShortStrawID })

	if phoneIndex < 0 {
		b.ShortStrawID = ids[0]
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
			os.MkdirAll(newDirName, os.ModePerm)
			createBillFile(newDirName)
			fmt.Printf("\n1. Enter values for the bill.toml file in new directory `%s`\n", newDirName)
			fmt.Println("2. Add csv files for minutes, message, megabytes in the new directory")
			fmt.Printf("3. run `ting-bill-split dir %s`\n", newDirName)
		} else {
			fmt.Println("Directory already exists.")
		}
	}
}

func createBillFile(path string) {
	path += "/bill.toml"
	f, err := os.Create(path)

	if err != nil {
		panic(err)
	}

	// TODO - instead of encoding the struct, maybe define the newBill to ensure we
	// don't forget to create requisite parts in the toml, but then use the newBill
	// to "hand-craft" the example toml. So we can group values in a sensible way
	// and provide helpful comment text
	newBill := tingbill.Bill{
		Description: "Ting Bill Split YYYY-MM-DD",
		Devices: []tingbill.Device{
			tingbill.Device{
				DeviceID: "1112223333",
				Owner:    "owner1",
			},
			tingbill.Device{
				DeviceID: "2229998888",
				Owner:    "owner2",
			},
			tingbill.Device{
				DeviceID: "3331119999",
				Owner:    "owner1",
			},
		},
		ShortStrawID:   "1112223333",
		Total:          0.00,
		DevicesCost:    0.00,
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
			fmt.Printf("Directory \"%v\" found, continuing...\n", file.Name())
			continue
		}

		if billFile == nil && isFileMatch(file.Name(), "bill", "toml") {
			billFile, err = os.Open(filepath.Join(path, file.Name()))
			if err != nil {
				log.Fatal(err)
			}
		}

		if minFile == nil && isFileMatch(file.Name(), "minutes", "csv") {
			minFile, err = os.Open(filepath.Join(path, file.Name()))
			if err != nil {
				log.Fatal(err)
			}
		}

		if msgFile == nil && isFileMatch(file.Name(), "messages", "csv") {
			msgFile, err = os.Open(filepath.Join(path, file.Name()))
			if err != nil {
				log.Fatal(err)
			}
		}

		if megFile == nil && isFileMatch(file.Name(), "megabytes", "csv") {
			megFile, err = os.Open(filepath.Join(path, file.Name()))
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

		//TODO take in each map and return a BillSplit
		split, err := parseMaps(minMap, msgMap, megMap, billData)
		if err != nil {
			log.Fatal(err)
		}

		pdfFilePath := filepath.Join(path, billData.Description+".pdf")

		invoiceName, err := tingpdf.GeneratePDF(split, billData, pdfFilePath)
		if err != nil {
			fmt.Printf("Failed to generate invoice at path: %v\n\n", pdfFilePath)
			log.Fatal(err)
		}
		fmt.Printf("Invoice generation complete: %s\n\n", invoiceName)
	}
}

func printUsageHelp() {
	fmt.Println("Use `ting-bill-split new` or `ting-bill-split new <billing-directory>` to create a new billing directory")
	fmt.Println("\nUse `ting-bill-split dir <billing-directory>` to run on a directory containing a `bill.toml`, and CSV files for minutes, messages, and megabytes usage.")
	fmt.Println("  Each of these files must contain their type somewhere in the filename - i.e. `YYYYMMDD-messages.csv` or `messages-potatosalad.csv` or whatever.")
}

func main() {
	fmt.Printf("\nTING BILL SPLIT\n")
	fmt.Println("***************")

	billPtr := flag.String("bill", "", "filename for bill toml - ex: -bill=\"bill.toml\"")
	minPtr := flag.String("minutes", "", "filename for minutes csv - ex: -minutes=\"minutes.csv\"")
	msgPtr := flag.String("messages", "", "filename for messages csv - ex: -messages=\"messages.csv\"")
	megPtr := flag.String("megabytes", "", "filename for megabytes csv - ex: -megabytes=\"megabytes.csv\"")

	flag.Parse()
	args := flag.Args()
	targetDir := "."

	if len(args) == 0 {
		fmt.Printf("`help`: usage guide for running batch mode against a single directory (recommended)\n")
		fmt.Printf("`-h`: usage guide for running with individual file flags\n\n")
	}

	if len(args) > 0 {
		fmt.Println("BATCH MODE")

		command := args[0]

		switch command {
		case "new":
			createNewBillingDir(args)
		case "dir":
			workingDir, err := os.Getwd()
			if err != nil {
				fmt.Println("Could not get working directory")
				log.Fatal(err)
			}

			if len(args) > 1 {
				targetDir = args[1]
			}

			fullTargetDir := ""

			// Handle absolute and relative paths, respectively
			if filepath.IsAbs(targetDir) {
				fullTargetDir = targetDir
			} else {
				fullTargetDir = filepath.Join(workingDir, targetDir)
			}

			if filepath.IsAbs(fullTargetDir) {
				parseDir(fullTargetDir)
				// TODO - make parseDir return a split, since non-dir parsing uses a split
				//   then handle all actions after the if/else
			} else {
				fmt.Printf("Bill directory %v is invalid\n\n", fullTargetDir)
			}
		case "help":
			printUsageHelp()
		default:
			fmt.Printf("\nInvalid arguments provided\n")
			printUsageHelp()
		}
	} else {
		badParam := false
		paramMap := map[string]*string{
			"bill":      billPtr,
			"minutes":   minPtr,
			"messages":  msgPtr,
			"megabytes": megPtr,
		}

		flagUsed := false
		for _, v := range paramMap {
			if *v != "" {
				flagUsed = true
			}
		}

		if flagUsed {
			fmt.Println("RUNNING WITH INDIVIDUAL FILE FLAGS")
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

			//TODO take in each map and return a BillSplit
			split, err := parseMaps(minMap, msgMap, megMap, billData)
			if err != nil {
				log.Fatal(err)
			}
			fmt.Println(split)
		}
	}
}
