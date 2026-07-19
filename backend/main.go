// Deklarant AI backend serveri.
package main

import (
	"log"
	"net/http"
	"os"

	"deklarant-ai/backend/internal/api"
	"deklarant-ai/backend/internal/chat"
	"deklarant-ai/backend/internal/countries"
	"deklarant-ai/backend/internal/docs"
	"deklarant-ai/backend/internal/hscode"
	"deklarant-ai/backend/internal/laws"
	"deklarant-ai/backend/internal/llm"
	"deklarant-ai/backend/internal/rates"
)

func main() {
	dataPath := getenv("HSCODE_DATA", "data/hscodes.json")
	lawsPath := getenv("LAWS_DATA", "data/laws.json")
	docsPath := getenv("DOCS_DATA", "data/docs.json")
	countriesPath := getenv("COUNTRIES_DATA", "data/countries.json")
	addr := ":" + getenv("PORT", "8080")

	codes, err := hscode.Load(dataPath)
	if err != nil {
		log.Fatalf("TIF TN bazasini yuklab bo'lmadi (%s): %v", dataPath, err)
	}
	m := codes.Meta()
	log.Printf("TIF TN bazasi: %d ta kod (%s, stavkalar %s holatiga)",
		len(codes.All()), m.Nomenclature, m.RatesAsOf)

	// Qonun korpusi ixtiyoriy — bo'lmasa ilova baribir ishlaydi.
	lawStore, err := laws.Load(lawsPath)
	if err != nil {
		log.Printf("Qonun korpusi yuklanmadi (%s): %v — chat qonun matnisiz ishlaydi", lawsPath, err)
		lawStore = nil
	} else {
		lm := lawStore.Meta()
		log.Printf("Qonun korpusi: %d ta hujjatdan %d parcha", lm.Docs, lawStore.Len())
	}

	// Hujjat talablari ham ixtiyoriy.
	docStore, err := docs.Load(docsPath)
	if err != nil {
		log.Printf("Hujjat talablari yuklanmadi (%s): %v — chat ularsiz ishlaydi", docsPath, err)
		docStore = nil
	} else {
		dm := docStore.Meta()
		log.Printf("Hujjat talablari: %d ta qoida, %d tur (%s holatiga)",
			docStore.Len(), dm.Types, dm.RulesAsOf)
	}

	// Har so'rovning token sarfini jurnalga yozamiz — xarajatni ko'rmasdan
	// boshqarib bo'lmaydi. cache_read nolga teng bo'lib qolsa, kesh
	// ishlamayapti degani va tizim ko'rsatmasi har safar to'liq to'lanadi.
	llm.OnUsage = func(u llm.Usage) {
		log.Printf("sarf: model=%s kirish=%d chiqish=%d kesh(yozildi=%d o'qildi=%d)",
			u.Model, u.InputTokens, u.OutputTokens, u.CacheWrite, u.CacheRead)
	}

	// Davlatlar ma'lumotnomasi — boj kelib chiqishga bog'liq (BK 300-modda).
	countryStore, err := countries.Load(countriesPath)
	if err != nil {
		log.Printf("Davlatlar ma'lumotnomasi yuklanmadi (%s): %v — boj kelib chiqishsiz hisoblanadi", countriesPath, err)
		countryStore = nil
	} else {
		log.Printf("Davlatlar: %d ta (%d tasi erkin savdo)", countryStore.Len(), len(countryStore.FreeTrade()))
	}

	// Valyuta kurslari — Markaziy bankdan. Xizmat ishlamasa chat kursni
	// SO'RAYDI, taxmin qilmaydi.
	rateClient := rates.New(os.Getenv("CBU_API_URL"))

	llmClient := llm.New()
	if llmClient.Available() {
		log.Printf("AI yoqilgan (Claude)")
	} else {
		log.Printf("AI o'chirilgan: ANTHROPIC_API_KEY sozlanmagan (chat ishlamaydi)")
	}

	chatSvc := chat.New(llmClient, codes, lawStore, docStore, rateClient)
	srv := api.New(codes, lawStore, chatSvc, llmClient, countryStore, rateClient)

	log.Printf("Server ishga tushdi: http://localhost%s", addr)
	if err := http.ListenAndServe(addr, srv.Routes()); err != nil {
		log.Fatalf("server xatosi: %v", err)
	}
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
