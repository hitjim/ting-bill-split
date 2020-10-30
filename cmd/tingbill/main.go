package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"

	"github.com/hitjim/ting-bill-split/internal/tingcsv"

	"github.com/hitjim/ting-bill-split/internal/tingbill"
	"github.com/hitjim/ting-bill-split/internal/tingparse"
	"github.com/hitjim/ting-bill-split/internal/tingpdf"

	"github.com/BurntSushi/toml"
)

func checkParam(param string, ptr *string, badParam *bool) {
	if *ptr == "" {
		*badParam = true
		fmt.Printf("%s parameter is bad\n", param)
	}
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
			if err := os.MkdirAll(newDirName, os.ModePerm); err != nil {
				log.Fatal("Failed to create new billing directory: ", err)
			}
			createBillFile(newDirName)
			fmt.Printf("\n1. Enter values for the bill.toml file in new directory `%s`\n", newDirName)
			fmt.Println("2. Add csv files for minutes, message, megabytes in the new directory")
			fmt.Printf("3. run `tingbill dir %s`\n", newDirName)
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

		billData, err := tingparse.ParseBill(billFile)
		if err != nil {
			log.Fatal(err)
		}

		minMap, err := tingparse.ParseMinutes(minFile)
		if err != nil {
			log.Fatal(err)
		}

		msgMap, err := tingparse.ParseMessages(msgFile)
		if err != nil {
			log.Fatal(err)
		}

		megMap, err := tingparse.ParseMegabytes(megFile)
		if err != nil {
			log.Fatal(err)
		}

		split, err := tingparse.CalculateSplit(minMap, msgMap, megMap, billData)
		if err != nil {
			log.Fatal(err)
		}

		pdfFilePath := filepath.Join(path, billData.Description+".pdf")
		invoiceName, err := tingpdf.GeneratePDF(split, billData, pdfFilePath)
		if err != nil {
			fmt.Printf("Failed to generate PDF invoice at path: %v\n\n", pdfFilePath)
			log.Fatal(err)
		}
		fmt.Printf("PDF invoice generation complete: %s\n", invoiceName)

		csvFilePath := filepath.Join(path, billData.Description+"_report.csv")
		invoiceName, err = tingcsv.GenerateCSV(split, billData, csvFilePath)
		if err != nil {
			fmt.Printf("Failed to generate CSV record at path: %v\n\n", csvFilePath)
			log.Fatal(err)
		}
		fmt.Printf("CSV invoice generation complete: %s\n\n", invoiceName)
	}
}

func printUsageHelp() {
	fmt.Println("Use `tingbill new` or `tingbill new <billing-directory>` to create a new billing directory")
	fmt.Println("\nUse `tingbill dir <billing-directory>` to run on a directory containing a `bill.toml`, and CSV files for minutes, messages, and megabytes usage.")
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
				//   then handle all actions after the if/else. Maybe?
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

			billData, err := tingparse.ParseBill(billFile)
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

			minMap, err := tingparse.ParseMinutes(minFile)
			if err != nil {
				log.Fatal(err)
			}

			msgMap, err := tingparse.ParseMessages(msgFile)
			if err != nil {
				log.Fatal(err)
			}

			megMap, err := tingparse.ParseMegabytes(megFile)
			if err != nil {
				log.Fatal(err)
			}

			split, err := tingparse.CalculateSplit(minMap, msgMap, megMap, billData)
			if err != nil {
				log.Fatal(err)
			}
			fmt.Println(split)
		}
	}
}
