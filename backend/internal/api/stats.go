package api

import (
	"maps"
	"sync"
	"time"

	"deklarant-ai/backend/internal/llm"
)

// stats — xotiradagi foydalanish statistikasi.
//
// NEGA XOTIRADA: bu yengil ko'rsatkichlar (sarf, so'rov soni), doimiy
// saqlanmasa ham bo'ladi. Server qayta ishga tushsa nolga qaytadi.
// Jiddiy hisobot kerak bo'lganda tashqi jurnal (masalan fayl yoki
// ma'lumotlar bazasi) qo'shiladi.
type stats struct {
	mu sync.Mutex

	started time.Time

	// Har yo'l bo'yicha so'rovlar soni.
	requests map[string]int64

	// Model bo'yicha jami tokenlar.
	tokens map[string]tokenTotals

	// Oxirgi sarf yozuvlari (halqa buferi) — jonli kuzatish uchun.
	recent    []usageRow
	recentCap int
	recentPos int
}

type tokenTotals struct {
	Calls      int64 `json:"calls"`
	Input      int64 `json:"input"`
	Output     int64 `json:"output"`
	CacheWrite int64 `json:"cache_write"`
	CacheRead  int64 `json:"cache_read"`
}

type usageRow struct {
	At         time.Time `json:"at"`
	Model      string    `json:"model"`
	Input      int64     `json:"input"`
	Output     int64     `json:"output"`
	CacheRead  int64     `json:"cache_read"`
	CacheWrite int64     `json:"cache_write"`
}

func newStats(now time.Time) *stats {
	return &stats{
		started:   now,
		requests:  map[string]int64{},
		tokens:    map[string]tokenTotals{},
		recent:    make([]usageRow, 100),
		recentCap: 100,
	}
}

// countRequest — bitta so'rovni hisobga oladi.
func (s *stats) countRequest(path string) {
	s.mu.Lock()
	s.requests[path]++
	s.mu.Unlock()
}

// addUsage — llm.OnUsage dan keladigan token sarfini yozadi.
func (s *stats) addUsage(u llm.Usage, now time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()

	t := s.tokens[u.Model]
	t.Calls++
	t.Input += int64(u.InputTokens)
	t.Output += int64(u.OutputTokens)
	t.CacheWrite += int64(u.CacheWrite)
	t.CacheRead += int64(u.CacheRead)
	s.tokens[u.Model] = t

	s.recent[s.recentPos] = usageRow{
		At: now, Model: u.Model,
		Input: int64(u.InputTokens), Output: int64(u.OutputTokens),
		CacheRead: int64(u.CacheRead), CacheWrite: int64(u.CacheWrite),
	}
	s.recentPos = (s.recentPos + 1) % s.recentCap
}

// snapshot — hisobot uchun statistikaning nusxasi.
type statsSnapshot struct {
	UptimeSeconds int64                  `json:"uptime_seconds"`
	Requests      map[string]int64       `json:"requests"`
	Tokens        map[string]tokenTotals `json:"tokens"`
	Recent        []usageRow             `json:"recent"`
}

func (s *stats) snapshot(now time.Time) statsSnapshot {
	s.mu.Lock()
	defer s.mu.Unlock()

	reqs := make(map[string]int64, len(s.requests))
	maps.Copy(reqs, s.requests)
	toks := make(map[string]tokenTotals, len(s.tokens))
	maps.Copy(toks, s.tokens)

	// Halqa buferini eng yangisidan eng eskisiga o'qib chiqamiz.
	var recent []usageRow
	for i := 0; i < s.recentCap; i++ {
		idx := (s.recentPos - 1 - i + s.recentCap) % s.recentCap
		if s.recent[idx].At.IsZero() {
			break
		}
		recent = append(recent, s.recent[idx])
	}

	return statsSnapshot{
		UptimeSeconds: int64(now.Sub(s.started).Seconds()),
		Requests:      reqs,
		Tokens:        toks,
		Recent:        recent,
	}
}
