// Package rates Markaziy bank valyuta kurslarini oladi.
//
// Manba: cbu.uz ochiq API — https://cbu.uz/uz/arkhiv-kursov-valyut/json/
//
// NEGA KERAK: bojxona qiymati so'mga aylantiriladi va kurs shunda hal
// qiluvchi. Ilgari foydalanuvchi kursni qo'lda kiritardi, kiritmasa esa
// AI o'zi TAXMIN qilardi ("kurs ~12 600 deb hisobladim") — bu jim va
// xavfli xato edi.
//
// DIQQAT: kurs SANAGA bog'liq. Bojxona qiymati deklaratsiya ro'yxatga
// olingan kundagi kurs bo'yicha hisoblanadi, "bugungi" bo'yicha emas.
package rates

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	defaultURL = "https://cbu.uz/uz/arkhiv-kursov-valyut/json"
	// Tashqi xizmat chat javobini kechiktirmasligi kerak.
	defaultTimeout = 5 * time.Second
)

// Rate — bitta valyuta kursi.
type Rate struct {
	Code   string  `json:"code"`    // raqamli kod (840), GTD da shu ishlatiladi
	Ccy    string  `json:"ccy"`     // USD
	NameUZ string  `json:"name_uz"` //
	Value  float64 `json:"value"`   // 1 BIRLIK necha so'm (nominal hisobga olingan)
	Date   string  `json:"date"`    // kurs sanasi, YYYY-MM-DD
}

// cbuRate — cbu.uz javobining bizga kerakli qismi.
type cbuRate struct {
	Code    string `json:"Code"`
	Ccy     string `json:"Ccy"`
	NameUZ  string `json:"CcyNm_UZ"`
	Nominal string `json:"Nominal"`
	Rate    string `json:"Rate"`
	Date    string `json:"Date"` // dd.mm.yyyy
}

// Client — kurslarni oladi va keshlaydi.
type Client struct {
	url  string
	http *http.Client

	mu    sync.RWMutex
	cache map[string]entry
}

type entry struct {
	rates   map[string]Rate // kalit: valyuta kodi (USD) va raqamli kod (840)
	expires time.Time       // nol bo'lsa — muddatsiz (tarixiy kurs o'zgarmaydi)
}

// New — mijoz yaratadi. url bo'sh bo'lsa cbu.uz ishlatiladi
// (testda soxta server manzili beriladi).
func New(url string) *Client {
	if url == "" {
		url = defaultURL
	}
	return &Client{
		url:   strings.TrimRight(url, "/"),
		http:  &http.Client{Timeout: defaultTimeout},
		cache: map[string]entry{},
	}
}

// On — berilgan sanadagi barcha kurslarni qaytaradi.
// Sana nol bo'lsa — eng oxirgi (bugungi) kurslar.
func (c *Client) On(ctx context.Context, day time.Time) (map[string]Rate, error) {
	key := "latest"
	url := c.url + "/"
	if !day.IsZero() {
		key = day.Format("2006-01-02")
		url = c.url + "/all/" + key + "/"
	}

	if r, ok := c.cached(key); ok {
		return r, nil
	}

	raw, err := c.fetch(ctx, url)
	if err != nil {
		return nil, err
	}
	out := make(map[string]Rate, len(raw)*2)
	for _, r := range raw {
		rate, err := convert(r)
		if err != nil {
			continue // bitta valyuta buzuq bo'lsa, qolganlari ishlasin
		}
		out[strings.ToUpper(rate.Ccy)] = rate
		out[rate.Code] = rate
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("kurslar bo'sh qaytdi")
	}

	c.store(key, out, day.IsZero())
	return out, nil
}

// Get — bitta valyutaning kursini qaytaradi.
// ccy — "USD" yoki raqamli kod "840".
func (c *Client) Get(ctx context.Context, ccy string, day time.Time) (Rate, error) {
	all, err := c.On(ctx, day)
	if err != nil {
		return Rate{}, err
	}
	r, ok := all[strings.ToUpper(strings.TrimSpace(ccy))]
	if !ok {
		return Rate{}, fmt.Errorf("valyuta topilmadi: %s", ccy)
	}
	return r, nil
}

func (c *Client) fetch(ctx context.Context, url string) ([]cbuRate, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("kurs olinmadi: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("cbu.uz status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	var out []cbuRate
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("kurs javobini o'qib bo'lmadi: %w", err)
	}
	return out, nil
}

// convert — cbu.uz yozuvini bizning ko'rinishga o'giradi.
//
// DIQQAT: "Nominal" — kotirovka nechta birlik uchun ekanini bildiradi.
// Ko'pchilikda 1, lekin IDR, IRR va VND da 10. Bo'lmasak, bu valyutalar
// o'n barobar xato chiqadi.
func convert(r cbuRate) (Rate, error) {
	v, err := strconv.ParseFloat(strings.TrimSpace(r.Rate), 64)
	if err != nil {
		return Rate{}, err
	}
	nominal := 1.0
	if n, err := strconv.ParseFloat(strings.TrimSpace(r.Nominal), 64); err == nil && n > 0 {
		nominal = n
	}
	return Rate{
		Code:   r.Code,
		Ccy:    r.Ccy,
		NameUZ: r.NameUZ,
		Value:  v / nominal,
		Date:   normalizeDate(r.Date),
	}, nil
}

// normalizeDate — "17.07.2026" → "2026-07-17".
func normalizeDate(s string) string {
	if t, err := time.Parse("02.01.2006", strings.TrimSpace(s)); err == nil {
		return t.Format("2006-01-02")
	}
	return s
}

func (c *Client) cached(key string) (map[string]Rate, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	e, ok := c.cache[key]
	if !ok {
		return nil, false
	}
	if !e.expires.IsZero() && time.Now().After(e.expires) {
		return nil, false
	}
	return e.rates, true
}

// store — keshga yozadi.
//
// Tarixiy kurs hech qachon o'zgarmaydi, shuning uchun muddatsiz saqlanadi.
// "Bugungi" kurs esa ertaga boshqacha bo'ladi — u kun oxirigacha yashaydi.
func (c *Client) store(key string, rates map[string]Rate, isLatest bool) {
	var expires time.Time
	if isLatest {
		now := time.Now()
		expires = time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, now.Location())
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cache[key] = entry{rates: rates, expires: expires}
}
