package duty

import "testing"

func TestCalculate(t *testing.T) {
	r := Calculate(Request{
		CustomsValue: 50_000_000,
		ImportDuty:   10,
		Excise:       0,
		VAT:          12,
		Quantity:     1,
	})

	// Import boj = 50M × 10% = 5M
	// Aksiz     = (50M + 5M) × 0% = 0
	// QQS       = (50M + 5M + 0) × 12% = 6.6M
	// Jami      = 490000 + 5M + 0 + 6.6M = 12,090,000
	want := 12_090_000.0
	if r.Total != want {
		t.Errorf("Total = %.0f; kutilgan %.0f", r.Total, want)
	}
	if len(r.Items) != 4 {
		t.Fatalf("Items soni = %d; kutilgan 4", len(r.Items))
	}
	if r.Items[1].Amount != 5_000_000 {
		t.Errorf("Import boji = %.0f; kutilgan 5000000", r.Items[1].Amount)
	}
	if r.Items[3].Amount != 6_600_000 {
		t.Errorf("QQS = %.0f; kutilgan 6600000", r.Items[3].Amount)
	}
}

func TestCalculateWithExcise(t *testing.T) {
	r := Calculate(Request{
		CustomsValue: 10_000_000,
		ImportDuty:   15,
		Excise:       20,
		VAT:          12,
	})
	// Import = 1.5M; Aksiz = (10M+1.5M)×20% = 2.3M; QQS = (10M+1.5M+2.3M)×12% = 1.656M
	// Jami = 490000 + 1.5M + 2.3M + 1.656M = 5,946,000
	want := 5_946_000.0
	if r.Total != want {
		t.Errorf("Total = %.0f; kutilgan %.0f", r.Total, want)
	}
}
