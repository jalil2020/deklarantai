package hscode

import (
	"fmt"
	"strings"
	"testing"
)

var evalCases = []struct{ query, want string }{
	{"наушниклар", "851830"},
	{"наушники", "851830"},
	{"naushnik", "851830"},
	{"микрофон", "851810"},
	{"traktor", "8701"},
	{"muzlatgich", "8418"},
	{"kompyuter", "8471"},
	{"noutbuk", "8471"},
	{"Chevrolet Cobalt import qilsam qancha", "8703"},
	{"Koreyadan 2019-yilgi Hyundai Sonata, 2.0 litr import qilaman", "8703"},
	{"Dongfeng musor tashuvchi mashina, 3 dona, Xitoydan, 2024-yil", "8705"},
	{"musor tashish mashinasi bojini hisobla", "8705"},
	{"axlat mashinasi import qilsam qancha to'lov", "8705"},
	{"yo'l tozalash mashinasi", "8705"},
	{"plastik chiqindi import qilsam qancha to'lov chiqadi", "39"},
	{"muzlatgich import qilsam qancha to'lov chiqadi", "8418"},
	{"10 000 dollarlik traktor import qilsam qancha to'lov chiqadi", "8701"},
	{"dori vositalari (bez ekstraktlari) import qilmoqchiman. Qanday hujjatlar kerak?", "30"},

	// IZAFAT (-i qo'shimchasi) — o'zbekchada tovar nomi deyarli DOIM
	// shu shaklda aytiladi: "havo konditsioneri", "kir yuvish mashinasi".
	//
	// Bu holatlar jonli sinovda topildi: "havo konditsioneri" 8415 ni
	// umuman qaytarmadi, o'rniga yog'-moy guruhini berdi. Sabab —
	// "konditsioneri" bazadagi "konditsionerlar" ga substring bo'lib
	// tushmaydi, qo'shimcha kesuvchi esa bo'g'iz "-i" ni bilmasdi.
	{"havo konditsioneri", "8415"},
	{"konditsioner", "8415"},
	{"kir yuvish mashinasi", "8450"},
	{"muzlatgich kamerasi", "8418"},
}

// parentWeight qiymati o'lchovga asoslangan. Bu test uni QO'RIQLAYDI:
// vazn o'zgartirilsa yoki qidiruv buzilsa, eval to'plami darrov aytadi.
//
// top-5 talab qilinadi (asl testlar ham shunday), top-1 esa ma'lumot
// uchun bosiladi — u sifat o'zgarishini erta ko'rsatadi.
func TestParentWeightIsMeasured(t *testing.T) {
	s, _ := Load("../../data/hscodes.json")
	orig := parentWeight
	defer func() { parentWeight = orig }()

	for _, w := range []float64{parentWeight} {
		parentWeight = w
		top1, top5 := 0, 0
		var fails []string
		for _, c := range evalCases {
			got := s.Search(c.query, 5)
			if len(got) > 0 && strings.HasPrefix(got[0].Code.Code, c.want) {
				top1++
			}
			ok := false
			for _, m := range got {
				if strings.HasPrefix(m.Code.Code, c.want) {
					ok = true
					break
				}
			}
			if ok {
				top5++
			} else {
				g := "—"
				if len(got) > 0 {
					g = got[0].Code.Code
				}
				fails = append(fails, fmt.Sprintf("%.18s→%s", c.query, g))
			}
		}
		fmt.Printf("vazn=%-2.0f top1=%2d/%d  top5=%2d/%d   %s\n",
			w, top1, len(evalCases), top5, len(evalCases), strings.Join(fails, "  "))

		// TALAB: tanlangan vaznda hamma holat top-5 da bo'lishi kerak.
		// Vazn o'zgartirilsa yoki qidiruv buzilsa, shu yerda ushlanadi.
		if top5 != len(evalCases) {
			t.Errorf("vazn=%.0f: top5 %d/%d — yiqilgan: %s",
				w, top5, len(evalCases), strings.Join(fails, "  "))
		}
	}
}
