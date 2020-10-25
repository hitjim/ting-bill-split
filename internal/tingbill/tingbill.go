package tingbill

import "github.com/shopspring/decimal"

func (b Bill) DeviceIds() []string {
	deviceIds := make([]string, len(b.Devices))

	for i, d := range b.Devices {
		deviceIds[i] = d.DeviceID
	}

	return deviceIds
}

func (b Bill) OwnerByID(id string) string {
	o := "Unknown"

	for _, d := range b.Devices {
		if id == d.DeviceID {
			o = d.Owner
		}
	}

	return o
}

type Device struct {
	DeviceID string
	Owner    string
}

// Used to represent the Ting-provided and user-provided info required to split Bill costs
type Bill struct {
	Description    string   `toml:"description"`
	Devices        []Device `toml:"devices"`
	ShortStrawID   string   `toml:"shortStrawId"`
	Total          float64  `toml:"total"`
	DevicesCost    float64  `toml:"devicesCost"`
	Minutes        float64  `toml:"minutes"`
	Messages       float64  `toml:"messages"`
	Megabytes      float64  `toml:"megabytes"`
	ExtraMinutes   float64  `toml:"extraMinutes"`
	ExtraMessages  float64  `toml:"extraMessages"`
	ExtraMegabytes float64  `toml:"extraMegabytes"`
	Fees           float64  `toml:"fees"`
}

// Used to contain all subtotals for a monthly Bill.
// MinuteCosts, MessageCosts, MegabyteCosts are maps of decimal.Decimal totals.
// They are split by Bill.Devices and calculated by usage in parseMaps.
// SharedCosts reflect the rest of the items not based on usage, which get split evenly across all DeviceIds
// TODO: finish these comments
type BillSplit struct {
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
