package tingcsv

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"

	"github.com/hitjim/ting-bill-split/internal/tingbill"
	"github.com/shopspring/decimal"
)

// GenerateCSV accepts a tingbill.BillSplit, tingbill.Bill, filepath string, and returns a string
// containing a filepath for the newly generated Ting Bill Split CSV, and an error.
// The tingbill.Bill should be the same one that generated the tingbill.BillSplit
// The first value in a table's header row will **Have Asterisks**
func GenerateCSV(bs tingbill.BillSplit, b tingbill.Bill, filePath string) (string, error) {
	fmt.Printf("\nGenerating invoice CSV...\n")
	const RoundPrecision = int32(2)

	// Bail right away if we can't write the CSV file
	csvFile, err := os.Create(filePath)
	if err != nil {
		return filePath, err
	}
	defer csvFile.Close()

	writer := csv.NewWriter(csvFile)
	defer writer.Flush()

	// Table 0: Heading - 7 columns, 2 rows
	// heading: Invoice filename w/date, Device qty, Bill Total, Split Total
	//   (for comparison), Usage subtotal, Devices Subtotal, Tax+Reg subtotal"

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

	records := [][]string{
		{"**Invoice with date**", "Devices Qty", "$Total", "$Calc", "$Usage", "$Devices", "$Tax+Reg"},
		{
			b.Description,
			strconv.Itoa(len(b.Devices)),
			strconv.FormatFloat(b.Total, 'f', 2, 64),
			calcCost.StringFixed(2),
			usgCost.StringFixed(2),
			strconv.FormatFloat(b.DevicesCost, 'f', 2, 64),
			strconv.FormatFloat(b.Fees, 'f', 2, 64),
		},
	}

	// Table 1: Usage - 8 columns, <deviceID qty>+1 rows
	// heading: number, nickname?, min, msg, data (KB), min%, msg%, data%
	// Then entries for each number
	// then entry for "Total" under nickname, and rest of sums
	records = append(records, []string{"**Phone Number**", "Owner", "Minutes", "Messages", "Data (KB)", "Min%", "Msg%", "Data%"})

	// Prep data
	ids := b.DeviceIds()

	for _, id := range ids {
		records = append(records, []string{
			id,
			b.OwnerByID(id),
			strconv.Itoa(bs.MinuteQty[id]),
			strconv.Itoa(bs.MessageQty[id]),
			strconv.Itoa(bs.MegabyteQty[id]),
			bs.MinutePercent[id].StringFixed(RoundPrecision),
			bs.MessagePercent[id].StringFixed(RoundPrecision),
			bs.MegabytePercent[id].StringFixed(RoundPrecision),
		})
	}

	// Table 2: Weighted Cost Type - 4 columns, 4 rows (+1 for cell to right of final column)
	// heading: Weighted: Minutes, Messages, Data
	// Base: $x, $y, $z
	// Extra: etc
	// Total: etc (sum of Min, Msg, Data gets tacked on as extra cell/col on final row)

	// Prep data
	totalMin := b.Minutes + b.ExtraMinutes
	totalMsg := b.Messages + b.ExtraMessages
	totalMeg := b.Megabytes + b.ExtraMegabytes
	wTotal := strconv.FormatFloat(totalMin+totalMsg+totalMeg, 'f', 2, 64)

	records = append(records, []string{"**Weighted**", "Minutes", "Messages", "Data"},
		[]string{
			"Base",
			strconv.FormatFloat(b.Minutes, 'f', 2, 64),
			strconv.FormatFloat(b.Messages, 'f', 2, 64),
			strconv.FormatFloat(b.Megabytes, 'f', 2, 64),
		},
		[]string{
			"Extra",
			strconv.FormatFloat(b.ExtraMinutes, 'f', 2, 64),
			strconv.FormatFloat(b.ExtraMessages, 'f', 2, 64),
			strconv.FormatFloat(b.ExtraMegabytes, 'f', 2, 64),
		},
		[]string{
			"Total",
			strconv.FormatFloat(totalMin, 'f', 2, 64),
			strconv.FormatFloat(totalMsg, 'f', 2, 64),
			strconv.FormatFloat(totalMeg, 'f', 2, 64),
			wTotal,
		},
	)

	// Table 3: Shared costs - 2 columns, 4 rows
	// heading: Shared, Amount
	// Devices: $
	// Tax & Reg: $
	// Total: $
	sTotal := strconv.FormatFloat(b.DevicesCost+b.Fees, 'f', 2, 64)

	records = append(records, []string{"**Shared**", "Amount"},
		[]string{
			"Devices",
			strconv.FormatFloat(b.DevicesCost, 'f', 2, 64),
		},
		[]string{
			"Tax & Reg",
			strconv.FormatFloat(b.Fees, 'f', 2, 64),
		},
		[]string{
			"Total",
			sTotal,
		},
	)

	// Table 4: Costs split - 7 columns, <deviceID qty>+1 rows
	// heading: number, Nickname, Min, Msg, Data, Shared, Total
	// entry for each number
	records = append(records, []string{"**Phone Number**", "Owner", "$Min", "$Msg", "$Data", "$Shared", "$Total"})

	for _, id := range ids {
		userTotal := decimal.Sum(bs.MinuteCosts[id], bs.MessageCosts[id], bs.MegabyteCosts[id], bs.SharedCosts[id])

		records = append(records, []string{
			id,
			b.OwnerByID(id),
			bs.MinuteCosts[id].StringFixed(2),
			bs.MessageCosts[id].StringFixed(2),
			bs.MegabyteCosts[id].StringFixed(2),
			bs.SharedCosts[id].StringFixed(2),
			userTotal.StringFixed(2),
		})
	}

	// Records complete, write to CSV
	err = writer.WriteAll(records)

	return filePath, err
}
