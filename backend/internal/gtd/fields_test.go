package gtd

import "strings"

import "testing"

// Grafalar ma'lumotnomasi izchil bo'lishi kerak.
func TestFieldsConsistent(t *testing.T) {
	seen := map[string]bool{}
	for _, f := range ImportFields {
		if f.No == "" || f.Name == "" {
			t.Errorf("bo'sh graf: %+v", f)
		}
		if seen[f.No] {
			t.Errorf("takroriy graf raqami: %s", f.No)
		}
		seen[f.No] = true
		// Auto grafada manba KO'RSATILISHI shart — aks holda model
		// nimaga tayanishini bilmaydi.
		if f.Fill == Auto && f.Src == "" {
			t.Errorf("graf %s avto, lekin manba yo'q", f.No)
		}
	}
	// Eng muhim avto grafalar bor bo'lishi shart: kod (33), to'lovlar (47).
	for _, want := range []string{"31", "33", "34", "45", "47"} {
		if !seen[want] {
			t.Errorf("muhim graf %s yo'q", want)
		}
	}
}

// Kod tanlash va to'lov hisobi — bizning kuchimiz, avto bo'lishi shart.
func TestKeyFieldsAreAuto(t *testing.T) {
	byNo := map[string]Field{}
	for _, f := range ImportFields {
		byNo[f.No] = f
	}
	for _, no := range []string{"33", "34", "45", "47"} {
		if byNo[no].Fill != Auto {
			t.Errorf("graf %s (%s) avto bo'lishi kerak", no, byNo[no].Name)
		}
	}
	if n := AutoCount(); n < 10 {
		t.Errorf("avto grafalar %d ta; kamida 10 kutilgan", n)
	}
	t.Logf("avto: %d / %d graf", AutoCount(), len(ImportFields))
}

// Prompt bloki barcha grafani va tushuntirishni o'z ichiga olishi kerak.
func TestPromptBlock(t *testing.T) {
	p := PromptBlock()
	for _, want := range []string{"[33] TIF TN kod", "[47] To'lovlar hisobi", "avto", "foydalanuvchi"} {
		if !strings.Contains(p, want) {
			t.Errorf("prompt blokida %q yo'q", want)
		}
	}
}
