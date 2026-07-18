// Package api HTTP endpointlarini va marshrutlashni ta'minlaydi.
package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"deklarant-ai/backend/internal/chat"
	"deklarant-ai/backend/internal/duty"
	"deklarant-ai/backend/internal/hscode"
	"deklarant-ai/backend/internal/laws"
	"deklarant-ai/backend/internal/llm"
)

// Server — barcha bog'liqliklarni ushlab turadi.
type Server struct {
	codes *hscode.Store
	laws  *laws.Store // nil bo'lishi mumkin
	chat  *chat.Service
	llm   *llm.Client
}

// New — server yaratadi.
func New(codes *hscode.Store, lawStore *laws.Store, chatSvc *chat.Service, llmClient *llm.Client) *Server {
	return &Server{codes: codes, laws: lawStore, chat: chatSvc, llm: llmClient}
}

// Routes — barcha marshrutlarni ro'yxatdan o'tkazadi.
func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", s.handleHealth)
	mux.HandleFunc("POST /api/hscode/search", s.handleHSSearch)
	mux.HandleFunc("POST /api/duty/calculate", s.handleDutyCalc)
	mux.HandleFunc("POST /api/chat", s.handleChat)
	return withCORS(mux)
}

// ---- Health ----

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	out := map[string]any{
		"status":       "ok",
		"ai_available": s.llm.Available(),
		"codes":        len(s.codes.All()),
		// Bazaning kelib chiqishi — foydalanuvchi "nima bor va qachongi holat"
		// deb so'raganda ko'rsatish uchun.
		"base": s.codes.Meta(),
	}
	if s.laws != nil {
		out["laws"] = s.laws.Meta()
	}
	writeJSON(w, http.StatusOK, out)
}

// ---- HS kod qidirish ----

type hsSearchRequest struct {
	Query string `json:"query"`
	UseAI bool   `json:"use_ai"`
}

type hsSearchResponse struct {
	Matches   []hscode.Match `json:"matches"`
	AIComment string         `json:"ai_comment,omitempty"`
	Source    string         `json:"source"` // "keyword" yoki "ai"
}

func (s *Server) handleHSSearch(w http.ResponseWriter, r *http.Request) {
	var req hsSearchRequest
	if err := decode(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "so'rovni o'qib bo'lmadi")
		return
	}
	if strings.TrimSpace(req.Query) == "" {
		writeErr(w, http.StatusBadRequest, "qidiruv so'rovi bo'sh")
		return
	}

	matches := s.codes.Search(req.Query, 5)
	resp := hsSearchResponse{Matches: matches, Source: "keyword"}

	// AI yoqilgan va mavjud bo'lsa, izoh qo'shamiz.
	if req.UseAI && s.llm.Available() {
		if comment, err := s.aiHSComment(r.Context(), req.Query, matches); err == nil {
			resp.AIComment = comment
			resp.Source = "ai"
		}
	}

	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) aiHSComment(ctx context.Context, query string, matches []hscode.Match) (string, error) {
	var b strings.Builder
	b.WriteString("Foydalanuvchi tovari: \"")
	b.WriteString(query)
	b.WriteString("\"\n\nBazadan topilgan mos kodlar:\n")
	if len(matches) == 0 {
		b.WriteString("(bazada mos kod topilmadi)\n")
	}
	for _, m := range matches {
		b.WriteString(fmt.Sprintf("- %s — %s (boj %g%%, QQS %g%%)\n",
			m.Code.Code, m.Code.PathUZ, m.Code.ImportDuty, m.Code.VAT))
	}
	b.WriteString("\nEng mos TIF TN kodni tavsiya qil va nima uchun ekanligini 2-3 gapda tushuntir. " +
		"Agar bazada mos kod bo'lmasa, umumiy TIF TN guruhi (birinchi 4 raqam) bo'yicha yo'nalish ber.")

	system := "Sen O'zbekiston TIF TN (tovar nomenklaturasi) bo'yicha ekspertsan. Qisqa, o'zbek tilida javob ber."
	return s.llm.Complete(ctx, system, []llm.Message{{Role: "user", Content: b.String()}})
}

// ---- Boj hisoblash ----

func (s *Server) handleDutyCalc(w http.ResponseWriter, r *http.Request) {
	var req duty.Request
	if err := decode(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "so'rovni o'qib bo'lmadi")
		return
	}
	if req.CustomsValue < 0 {
		writeErr(w, http.StatusBadRequest, "bojxona qiymati manfiy bo'lishi mumkin emas")
		return
	}
	writeJSON(w, http.StatusOK, duty.Calculate(req))
}

// ---- Chat ----

type chatRequest struct {
	Messages []llm.Message `json:"messages"`
}

func (s *Server) handleChat(w http.ResponseWriter, r *http.Request) {
	var req chatRequest
	if err := decode(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "so'rovni o'qib bo'lmadi")
		return
	}
	if len(req.Messages) == 0 {
		writeErr(w, http.StatusBadRequest, "xabar yo'q")
		return
	}
	if !s.chat.Available() {
		writeErr(w, http.StatusServiceUnavailable,
			"AI xizmati sozlanmagan. ANTHROPIC_API_KEY muhit o'zgaruvchisini o'rnating.")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()

	reply, err := s.chat.Reply(ctx, req.Messages)
	if err != nil {
		writeErr(w, http.StatusBadGateway, "AI javob bermadi: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"reply": reply})
}

// ---- Yordamchilar ----

func decode(r *http.Request, v any) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(v)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
