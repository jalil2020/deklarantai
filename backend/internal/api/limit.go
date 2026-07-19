package api

import (
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Cheklovlar. Hammasi muhit o'zgaruvchisi bilan sozlanadi.
//
// NEGA KERAK: har bir chat so'rovi pul turadi (~0,1–0,3 dollar). Cheklovsiz
// bitta skript bir kechada butun byudjetni yeb qo'yishi mumkin. Bu
// autentifikatsiya O'RNINI BOSMAYDI — u kelgunicha eng kam himoya.
const (
	defaultRatePerMin = 10        // bitta IP dan daqiqasiga
	defaultDailyQuota = 100       // bitta IP dan kuniga
	defaultMaxBody    = 8 << 20   // 8 MB — rasm base64 bilan kelishi mumkin
	cleanupEvery      = time.Hour // eski yozuvlarni tozalash oralig'i
)

// envInt — muhit o'zgaruvchisidan manfiy bo'lmagan butun son.
// 0 qiymati cheklovni O'CHIRADI, noto'g'ri qiymat esa sukutni qoldiradi.
func envInt(name string, def int) int {
	v, err := strconv.Atoi(os.Getenv(name))
	if err != nil || v < 0 {
		return def
	}
	return v
}

// limiter — IP bo'yicha tezlik va kunlik kvota hisoblagichi.
//
// Xotirada saqlanadi: bitta nusxa uchun yetarli. Bir nechta server
// ishlaganda umumiy hisoblagich (masalan Redis) kerak bo'ladi.
type limiter struct {
	ratePerMin int
	dailyQuota int

	mu          sync.Mutex
	seen        map[string]*counter
	lastCleanup time.Time
	now         func() time.Time // testda vaqtni boshqarish uchun
}

type counter struct {
	minute   time.Time
	inMinute int
	day      time.Time
	inDay    int
}

func newLimiter() *limiter {
	return &limiter{
		ratePerMin: envInt("RATE_PER_MIN", defaultRatePerMin),
		dailyQuota: envInt("DAILY_QUOTA", defaultDailyQuota),
		seen:       map[string]*counter{},
		now:        time.Now,
	}
}

// allow — so'rovga ruxsat berilishini tekshiradi.
// Qaytaradi: ruxsat, sabab (rad etilganda) va kutish kerak bo'lgan soniya.
func (l *limiter) allow(ip string) (bool, string, int) {
	if l.ratePerMin <= 0 && l.dailyQuota <= 0 {
		return true, "", 0
	}
	now := l.now()
	minute := now.Truncate(time.Minute)
	day := now.Truncate(24 * time.Hour)

	l.mu.Lock()
	defer l.mu.Unlock()

	// Tozalash alohida goroutine bilan emas, shu yerda vaqti-vaqti bilan
	// bajariladi: goroutine har bir server nusxasida oqib qolardi
	// (testlarda o'nlab server yaratiladi), bu esa oddiy va yetarli.
	if l.lastCleanup.IsZero() {
		l.lastCleanup = now
	} else if now.Sub(l.lastCleanup) >= cleanupEvery {
		l.lastCleanup = now
		l.cleanupLocked(now)
	}

	c := l.seen[ip]
	if c == nil {
		c = &counter{}
		l.seen[ip] = c
	}
	if !c.minute.Equal(minute) {
		c.minute, c.inMinute = minute, 0
	}
	if !c.day.Equal(day) {
		c.day, c.inDay = day, 0
	}

	if l.dailyQuota > 0 && c.inDay >= l.dailyQuota {
		// Kun oxirigacha qancha qolganini aytamiz — foydalanuvchi qachon
		// qayta urinishni bilsin.
		wait := int(day.Add(24 * time.Hour).Sub(now).Seconds())
		return false, "kunlik chegara tugadi", wait
	}
	if l.ratePerMin > 0 && c.inMinute >= l.ratePerMin {
		wait := int(minute.Add(time.Minute).Sub(now).Seconds()) + 1
		return false, "juda tez-tez so'rov yuborilmoqda", wait
	}

	c.inMinute++
	c.inDay++
	return true, "", 0
}

// cleanup — bir kundan ortiq tegilmagan yozuvlarni o'chiradi.
// Busiz xotira IP lar soniga qarab cheksiz o'sardi.
func (l *limiter) cleanup() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.cleanupLocked(l.now())
}

// cleanupLocked — mu ushlab turilgan holda chaqiriladi.
func (l *limiter) cleanupLocked(now time.Time) {
	cutoff := now.Add(-24 * time.Hour)
	for ip, c := range l.seen {
		if c.day.Before(cutoff) {
			delete(l.seen, ip)
		}
	}
}

// clientIP — so'rov manbai.
//
// Proksi orqasida ishlaganda X-Forwarded-For dagi BIRINCHI manzil olinadi.
// DIQQAT: bu sarlavhani mijoz o'zi qalbakilashtira oladi, shuning uchun
// TRUST_PROXY yoqilganda va faqat ishonchli proksi orqasida ishlatilishi
// kerak. Aks holda to'g'ridan-to'g'ri ulanish manzili olinadi.
func clientIP(r *http.Request, trustProxy bool) string {
	if trustProxy {
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			if i := strings.IndexByte(xff, ','); i > 0 {
				return strings.TrimSpace(xff[:i])
			}
			return strings.TrimSpace(xff)
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// withLimits — so'rov hajmi va tezlik cheklovlarini qo'llaydi.
//
// Cheklov faqat QIMMAT yo'llarga (chat) qo'llanadi: health va kalkulyator
// arzon va ular cheklansa, oddiy foydalanish buziladi.
func (s *Server) withLimits(next http.Handler) http.Handler {
	maxBody := int64(envInt("MAX_BODY_BYTES", defaultMaxBody))
	trustProxy := os.Getenv("TRUST_PROXY") == "1"

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Hajm chegarasi HAMMA POST so'rovga: cheklovsiz bitta so'rov
		// bilan serverning xotirasini to'ldirish mumkin edi.
		if r.Method == http.MethodPost {
			r.Body = http.MaxBytesReader(w, r.Body, maxBody)
		}

		if strings.HasPrefix(r.URL.Path, "/api/chat") {
			if ok, reason, wait := s.limiter.allow(clientIP(r, trustProxy)); !ok {
				w.Header().Set("Retry-After", strconv.Itoa(wait))
				writeErr(w, http.StatusTooManyRequests, reason)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}
