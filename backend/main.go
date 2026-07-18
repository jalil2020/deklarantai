// Deklarant AI backend serveri.
package main

import (
	"log"
	"net/http"
	"os"

	"deklarant-ai/backend/internal/api"
	"deklarant-ai/backend/internal/chat"
	"deklarant-ai/backend/internal/hscode"
	"deklarant-ai/backend/internal/laws"
	"deklarant-ai/backend/internal/llm"
)

func main() {
	dataPath := getenv("HSCODE_DATA", "data/hscodes.json")
	lawsPath := getenv("LAWS_DATA", "data/laws.json")
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

	llmClient := llm.New()
	if llmClient.Available() {
		log.Printf("AI yoqilgan (Claude)")
	} else {
		log.Printf("AI o'chirilgan: ANTHROPIC_API_KEY sozlanmagan (chat ishlamaydi)")
	}

	chatSvc := chat.New(llmClient, codes, lawStore)
	srv := api.New(codes, lawStore, chatSvc, llmClient)

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
