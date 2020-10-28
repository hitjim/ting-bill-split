package tingpdf

import (
	"fmt"
	"strconv"

	"github.com/hitjim/ting-bill-split/internal/tingbill"
	"github.com/jung-kurt/gofpdf"
	"github.com/shopspring/decimal"
)

// GeneratePDF accepts a tingbill.BillSplit, tingbill.Bill, filepath string, and returns a string
// containing a filepath for the newly generated Ting Bill Split PDF, and an error.
// The tingbill.Bill should be the same one that generated the tingbill.BillSplit.
func GeneratePDF(bs tingbill.BillSplit, b tingbill.Bill, filePath string) (string, error) {
	fmt.Printf("\nGenerating invoice PDF...\n")
	RoundPrecision := int32(2)

	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()
	pdf.SetFont("Arial", "B", 10)
	pdf.SetXY(10, 20)

	// Table 0: Heading - 7 rows
	// heading: Invoice filename w/date, Device qty, Bill Total, Split Total
	//   (for comparison), Usage subtotal, Devices Subtotal, Tax+Reg subtotal"
	headingTable := func(b tingbill.Bill, bs tingbill.BillSplit) {
		pageHeading := []string{"Invoice with date", "Devices Qty", "$Total", "$Calc", "$Usage", "$Devices", "$Tax+Reg"}
		w := []float64{65.0, 25.0, 20.0, 20.0, 20.0, 20.0, 20.0}

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
			strconv.Itoa(len(b.Devices)),
			strconv.FormatFloat(b.Total, 'f', 2, 64),
			calcCost.StringFixed(2),
			usgCost.StringFixed(2),
			strconv.FormatFloat(b.DevicesCost, 'f', 2, 64),
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
	usageTable := func(b tingbill.Bill, bs tingbill.BillSplit) {
		type usageTableVals struct {
			id         string
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
		ids := b.DeviceIds()

		for _, id := range ids {
			values[id] = usageTableVals{
				id,
				b.OwnerByID(id),
				strconv.Itoa(bs.MinuteQty[id]),
				strconv.Itoa(bs.MessageQty[id]),
				strconv.Itoa(bs.MegabyteQty[id]),
				bs.MinutePercent[id].StringFixed(RoundPrecision),
				bs.MessagePercent[id].StringFixed(RoundPrecision),
				bs.MegabytePercent[id].StringFixed(RoundPrecision),
			}
		}

		// TODO: turn this into some kind of getMapKeys if it gets too crazy

		// Print data
		pdf.SetXY(10, pdf.GetY())
		var wi int
		valuesBound := len(values) - 1

		for i, id := range ids {
			wi = 0
			row := values[id]

			pdf.CellFormat(w[wi], 7, row.id, "1", 0, "C", false, 0, "")
			wi++
			pdf.CellFormat(w[wi], 7, row.owner, "1", 0, "C", false, 0, "")
			wi++
			pdf.CellFormat(w[wi], 7, row.minutes, "1", 0, "R", false, 0, "")
			wi++
			pdf.CellFormat(w[wi], 7, row.messages, "1", 0, "R", false, 0, "")
			wi++
			pdf.CellFormat(w[wi], 7, row.data, "1", 0, "R", false, 0, "")
			wi++
			pdf.CellFormat(w[wi], 7, row.percentMin, "1", 0, "C", false, 0, "")
			wi++
			pdf.CellFormat(w[wi], 7, row.percentMsg, "1", 0, "C", false, 0, "")
			wi++
			pdf.CellFormat(w[wi], 7, row.percentMeg, "1", 0, "C", false, 0, "")
			if i < valuesBound {
				pdf.SetXY(10, pdf.GetY()+7)
			}
		}
		pdf.Ln(-1)
	}
	usageTable(b, bs)

	// Table 2: Weighted costs - 3 rows
	// heading: Weighted Costs: Minutes, Messages, Data
	// Base: $x, $y, $z
	// Extra: etc
	// Total: etc
	weightedTable := func(b tingbill.Bill) {
		type weightedTableVals struct {
			name     string
			minutes  string
			messages string
			data     string
		}

		wtheading := []string{"Cost Type", "Minutes", "Messages", "Data"}
		pdf.SetXY(10, pdf.GetY()+5)

		// Print heading
		for _, str := range wtheading {
			pdf.CellFormat(25.0, 7, str, "1", 0, "C", false, 0, "")
		}
		pdf.Ln(-1)

		// Prep data
		totalMin := b.Minutes + b.ExtraMinutes
		totalMsg := b.Messages + b.ExtraMessages
		totalMeg := b.Megabytes + b.ExtraMegabytes
		wTotal := strconv.FormatFloat(totalMin+totalMsg+totalMeg, 'f', 2, 64)

		values := []weightedTableVals{
			{
				name:     "Base",
				minutes:  strconv.FormatFloat(b.Minutes, 'f', 2, 64),
				messages: strconv.FormatFloat(b.Messages, 'f', 2, 64),
				data:     strconv.FormatFloat(b.Megabytes, 'f', 2, 64),
			},
			{
				name:     "Extra",
				minutes:  strconv.FormatFloat(b.ExtraMinutes, 'f', 2, 64),
				messages: strconv.FormatFloat(b.ExtraMessages, 'f', 2, 64),
				data:     strconv.FormatFloat(b.ExtraMegabytes, 'f', 2, 64),
			},
			{
				name:     "Total",
				minutes:  strconv.FormatFloat(totalMin, 'f', 2, 64),
				messages: strconv.FormatFloat(totalMsg, 'f', 2, 64),
				data:     strconv.FormatFloat(totalMeg, 'f', 2, 64),
			},
		}

		// Print data
		pdf.SetXY(10, pdf.GetY())
		valuesBound := len(values) - 1

		for i, row := range values {
			pdf.CellFormat(25.0, 7, row.name, "1", 0, "C", false, 0, "")
			pdf.CellFormat(25.0, 7, row.minutes, "1", 0, "C", false, 0, "")
			pdf.CellFormat(25.0, 7, row.messages, "1", 0, "C", false, 0, "")
			pdf.CellFormat(25.0, 7, row.data, "1", 0, "C", false, 0, "")

			if i < valuesBound {
				pdf.SetXY(10, pdf.GetY()+7)
			} else {
				pdf.CellFormat(25.0, 7, wTotal, "1", 0, "C", false, 0, "")
			}
		}
		pdf.Ln(-1)
	}
	weightedTable(b)

	// Table 3: Shared costs - 2
	// TODO LATER - handle all the tax and reg costs in bill file?
	// heading: Type, Amount
	sharedTable := func(b tingbill.Bill) {
		type sharedTableVals struct {
			costType string
			amount   string
		}

		stheading := []string{"Type", "Amount"}
		pdf.SetXY(10, pdf.GetY()+5)

		// Print heading
		for _, str := range stheading {
			pdf.CellFormat(25.0, 7, str, "1", 0, "C", false, 0, "")
		}
		pdf.Ln(-1)

		// Prep data
		sTotal := strconv.FormatFloat(b.DevicesCost+b.Fees, 'f', 2, 64)

		values := []sharedTableVals{
			{
				costType: "Devices",
				amount:   strconv.FormatFloat(b.DevicesCost, 'f', 2, 64),
			},
			{
				costType: "Tax & Reg",
				amount:   strconv.FormatFloat(b.Fees, 'f', 2, 64),
			},
			{
				costType: "Total",
				amount:   sTotal,
			},
		}

		// Print data
		pdf.SetXY(10, pdf.GetY())
		valuesBound := len(values) - 1

		for i, row := range values {
			pdf.CellFormat(25.0, 7, row.costType, "1", 0, "L", false, 0, "")
			pdf.CellFormat(25.0, 7, row.amount, "1", 0, "R", false, 0, "")

			if i < valuesBound {
				pdf.SetXY(10, pdf.GetY()+7)
			}
		}
		pdf.Ln(-1)
	}
	sharedTable(b)

	// Table 4: Costs split - 7
	// heading: number, Nickname, Min, Msg, Data, Shared, Total
	// entry for each number
	splitTable := func(bs tingbill.BillSplit) {
		type splitTableVals struct {
			id       string
			owner    string
			minutes  string
			messages string
			data     string
			shared   string
			total    string
		}

		splitTableHeading := []string{"Phone Number", "Owner", "$Min", "$Msg", "$Data", "$Shared", "$Total"}
		w := []float64{35.0, 30.0, 25.0, 25.0, 25.0, 25.0, 25.0}
		pdf.SetXY(10, pdf.GetY()+5)

		// Print heading
		for i, str := range splitTableHeading {
			pdf.CellFormat(w[i], 7, str, "1", 0, "C", false, 0, "")
		}
		pdf.Ln(-1)

		// Prep data
		values := make(map[string]splitTableVals)

		ids := b.DeviceIds()

		for _, id := range ids {
			userTotal := decimal.Sum(bs.MinuteCosts[id], bs.MessageCosts[id], bs.MegabyteCosts[id], bs.SharedCosts[id])
			values[id] = splitTableVals{
				id,
				b.OwnerByID(id),
				bs.MinuteCosts[id].StringFixed(2),
				bs.MessageCosts[id].StringFixed(2),
				bs.MegabyteCosts[id].StringFixed(2),
				bs.SharedCosts[id].StringFixed(2),
				userTotal.StringFixed(2),
			}
		}

		// Print data
		pdf.SetXY(10, pdf.GetY())
		var wi int
		valuesBound := len(values) - 1

		for i, id := range ids {
			wi = 0
			row := values[id]

			pdf.CellFormat(w[wi], 7, row.id, "1", 0, "C", false, 0, "")
			wi++
			pdf.CellFormat(w[wi], 7, row.owner, "1", 0, "C", false, 0, "")
			wi++
			pdf.CellFormat(w[wi], 7, row.minutes, "1", 0, "R", false, 0, "")
			wi++
			pdf.CellFormat(w[wi], 7, row.messages, "1", 0, "R", false, 0, "")
			wi++
			pdf.CellFormat(w[wi], 7, row.data, "1", 0, "R", false, 0, "")
			wi++
			pdf.CellFormat(w[wi], 7, row.shared, "1", 0, "R", false, 0, "")
			wi++
			pdf.CellFormat(w[wi], 7, row.total, "1", 0, "R", false, 0, "")
			wi++
			if i < valuesBound {
				pdf.SetXY(10, pdf.GetY()+7)
			}
		}
		pdf.Ln(-1)
	}
	splitTable(bs)

	err := pdf.OutputFileAndClose(filePath)

	// TODO - add dates to bill. For now, entering manually in the "description" field in bill.toml
	// Future: generate a range off min/max dates in usage files?
	// Or maybe just have a new field in toml?

	return filePath, err
}
