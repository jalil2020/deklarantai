// Deklarant AI backend serveri.
package main

import (
	"log"
	"net/http"
	"os"

	"deklarant-ai/backend/internal/api"
	"deklarant-ai/backend/internal/chat"
	"deklarant-ai/backend/internal/hscode"
	"deklarant-ai/backend/internal/llm"
)

func main() {
	dataPath := getenv("HSCODE_DATA", "data/hscodes.json")
	addr := ":" + getenv("PORT", "8080")

	codes, err := hscode.Load(dataPath)
	if err != nil {
		log.Fatalf("TIF TN bazasini yuklab bo'lmadi (%s): %v", dataPath, err)
	}
	log.Printf("TIF TN bazasi yuklandi: %d ta kod", len(codes.All()))

	llmClient := llm.New()
	if llmClient.Available() {
		log.Printf("AI yoqilgan (Claude)")
	} else {
		log.Printf("AI o'chirilgan: ANTHROPIC_API_KEY sozlanmagan (chat ishlamaydi)")
	}

	chatSvc := chat.New(llmClient, codes)
	srv := api.New(codes, chatSvc, llmClient)

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
