// Package duty O'zbekiston bojxona to'lovlarini hisoblaydi.
package duty

import "math"

// Request — hisoblash uchun kirish ma'lumotlari.
type Request struct {
	CustomsValue float64 `json:"customs_value"` // bojxona qiymati (so'mda)
	ImportDuty   float64 `json:"import_duty"`   // import boj stavkasi, %
	Excise       float64 `json:"excise"`        // aksiz stavkasi, %
	VAT          float64 `json:"vat"`           // QQS stavkasi, %
	Quantity     float64 `json:"quantity"`      // miqdor (ixtiyoriy, ma'lumot uchun)
}

// LineItem — hisob-kitobning bitta qatori.
type LineItem struct {
	Name   string  `json:"name"`
	Rate   float64 `json:"rate"`   // stavka, % (agar bo'lsa)
	Base   float64 `json:"base"`   // hisoblash bazasi
	Amount float64 `json:"amount"` // to'lov summasi
}

// Result — to'liq hisob-kitob natijasi.
type Result struct {
	Items []LineItem `json:"items"`
	Total float64    `json:"total"`
}

// clearanceFee — bojxona yig'imi (soddalashtirilgan qat'iy summa, demo).
const clearanceFee = 490000.0 // ~ shartli yig'im, so'mda

// Calculate — O'zbekiston metodikasi bo'yicha ketma-ket hisoblaydi:
//
//	Import boj = Bojxona qiymati × boj%
//	Aksiz      = (Bojxona qiymati + Import boj) × aksiz%
//	QQS        = (Bojxona qiymati + Import boj + Aksiz) × QQS%
//	Jami       = Bojxona yig'imi + Import boj + Aksiz + QQS
func Calculate(r Request) Result {
	var items []LineItem
	var total float64

	// Bojxona yig'imi (qat'iy).
	items = append(items, LineItem{
		Name:   "Bojxona yig'imi",
		Rate:   0,
		Base:   r.CustomsValue,
		Amount: round(clearanceFee),
	})
	total += clearanceFee

	// Import boj.
	importDuty := r.CustomsValue * r.ImportDuty / 100
	items = append(items, LineItem{
		Name:   "Import boji",
		Rate:   r.ImportDuty,
		Base:   r.CustomsValue,
		Amount: round(importDuty),
	})
	total += importDuty

	// Aksiz.
	exciseBase := r.CustomsValue + importDuty
	excise := exciseBase * r.Excise / 100
	items = append(items, LineItem{
		Name:   "Aksiz solig'i",
		Rate:   r.Excise,
		Base:   round(exciseBase),
		Amount: round(excise),
	})
	total += excise

	// QQS.
	vatBase := r.CustomsValue + importDuty + excise
	vat := vatBase * r.VAT / 100
	items = append(items, LineItem{
		Name:   "QQS",
		Rate:   r.VAT,
		Base:   round(vatBase),
		Amount: round(vat),
	})
	total += vat

	return Result{Items: items, Total: round(total)}
}

func round(v float64) float64 {
	return math.Round(v*100) / 100
}
