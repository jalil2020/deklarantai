// Package llm Anthropic Claude Messages API bilan ishlaydigan minimal klient.
// Matn va rasm (vision) xabarlarini qo'llab-quvvatlaydi.
package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	defaultAPIURL = "https://api.anthropic.com/v1/messages"
	apiVersion    = "2023-06-01"
	defaultModel  = "claude-opus-4-8"
	// O'rta model SUKUT BO'YICHA O'CHIQ (bo'sh = asosiy modelga qaytadi).
	//
	// ⚠️ NEGA O'CHIQ: u yoqilgan holda ishga tushirilgach, foydalanuvchi
	// javob sifati pasayganini sezdi. O'lchov sababni tasdiqladi — 25 ta
	// haqiqiy savoldan 19 tasi (76%) Opus dan Sonnet ga tushib ketgan,
	// chunki yo'naltirish qoidasi juda keng edi (chat.modelFor ga qarang).
	//
	// Qoida toraytirildi, lekin sukut baribir SIFAT tomonida qoldi:
	// bu ilova pul va jarima haqida javob beradi, shuning uchun
	// arzonlashtirish OCHIQ TANLOV bo'lishi kerak, jimgina sukut emas.
	// Yoqish: ANTHROPIC_MID_MODEL=claude-sonnet-5
	defaultMidModel = ""
	// Arzon model ham SUKUT BO'YICHA O'CHIQ (bo'sh = asosiy modelga
	// qaytadi). Ya'ni sukut holatda HAMMA so'rov ANTHROPIC_MODEL ga
	// ketadi — bo'sh gap ham.
	//
	// NEGA: bu ilova pul va jarima haqida javob beradi. Modelni
	// jimgina almashtirish javob sifatini o'zgartiradi va buni
	// foydalanuvchi sezadi (bir marta shunday bo'lgan — modelFor
	// izohiga qarang). Arzonlashtirish OCHIQ TANLOV bo'lsin.
	//
	// Bo'sh gapning narxi baribir kichik: unga kontekst umuman
	// izlanmaydi (chat.withRetrieval), ya'ni "salom" 8 token kirish.
	// Yoqish: ANTHROPIC_FAST_MODEL=claude-haiku-4-5-20251001
	defaultFastModel = ""
	defaultMaxTokens = 2048
)

// ErrNoAPIKey — API kaliti sozlanmagan.
var ErrNoAPIKey = errors.New("ANTHROPIC_API_KEY sozlanmagan")

// Client — LLM klienti. Ikki provayderni boshqaradi:
//
//	Claude (Anthropic) — rasm va hisob-kitob. Vision faqat shu yerda bor.
//	GLM (Z.ai)         — qolgan matnli savollar, ~4 barobar arzon.
//
// GLM sozlanmagan bo'lsa, arzon so'rovlar Claude ning arzon modeliga
// (Haiku) qaytadi — ya'ni GLM_API_KEY siz ham dastur avvalgidek ishlaydi.
type Client struct {
	apiKey    string
	model     string
	midModel  string // bazaga tayangan ma'lumot savollari
	fastModel string // GLM yo'q bo'lsa ishlatiladigan zaxira arzon model
	url       string
	maxTokens int
	cacheTTL  string // "" (5 daqiqa) yoki "1h"
	http      *http.Client
	glm       *glmClient
}

// Model — javob uchun qaysi darajani ishlatish.
type Model int

const (
	// Full — Claude Opus. Rasm, boj/QQS/aksiz hisob-kitobi — xato narxi
	// yuqori bo'lgan hamma savol.
	Full Model = iota
	// Mid — o'rta daraja (Sonnet). Javob BIZNING BAZA blokiga tayangan
	// ma'lumot savollari: "bu kod nimani anglatadi", "qaysi modda",
	// "qanday hujjat kerak".
	//
	// NEGA XAVFSIZ: bunday savolda model o'ylamaydi — kontekstda berilgan
	// faktni o'qib qayta ifodalaydi. Xavf arifmetikada va "qaysi kod
	// to'g'ri keladi" degan hukmda to'planadi, ular esa Full da qoladi.
	Mid
	// Cheap — arzon daraja (GLM-5.2, u yo'q bo'lsa Haiku). Bazadan hech
	// narsa topilmagan qisqa savol: salomlashish, "nima qila olasan".
	// Hisob-kitob, rasm va bazaga tayangan javob bu yerga TUSHMAYDI.
	Cheap
)

// String — jurnal va test uchun o'qiladigan nom.
func (m Model) String() string {
	switch m {
	case Full:
		return "Full"
	case Mid:
		return "Mid"
	case Cheap:
		return "Cheap"
	}
	return "noma'lum"
}

// anthropicModel — Anthropic so'rovi uchun model nomi.
//
// modelName dan ATAYLAB alohida: modelName GLM nomini ham qaytarishi
// mumkin, uni esa Anthropic so'rovining "model" maydoniga qo'yib bo'lmaydi
// (API "unknown model" xatosi berardi). Bitta funksiya ikkala vazifani
// bajarsa, shu xato jimgina kirib kelardi.
func (c *Client) anthropicModel(m Model) string {
	switch {
	case m == Cheap && c.fastModel != "":
		return c.fastModel
	case m == Mid && c.midModel != "":
		return c.midModel
	}
	return c.model
}

// modelName — sarf hisobotida ko'rsatiladigan model nomi (provayder bilan).
func (c *Client) modelName(m Model) string {
	if c.useGLM(m) {
		return c.glm.model
	}
	return c.anthropicModel(m)
}

// pickModel — YAKUNIY qaror: so'ralgan daraja va xabar tarkibiga qarab.
//
// QAT'IY QOIDA: rasm bo'lsa, doim Full. GLM-5.2 rasmni umuman qabul
// qilmaydi (Z.ai: "Input Modalities: Text"), shuning uchun bu tekshiruv
// chaqiruvchiga emas, shu yerga qo'yilgan — yo'naltirishda xato qilinsa
// ham rasm hech qachon matn-only provayderga ketmaydi.
func pickModel(m Model, history []Message) Model {
	for _, msg := range history {
		if len(msg.Images) > 0 {
			return Full
		}
	}
	return m
}

// useGLM — bu so'rov GLM ga ketadimi.
func (c *Client) useGLM(m Model) bool {
	return m == Cheap && c.glm.available()
}

// New — muhit o'zgaruvchilaridan klient yaratadi.
//
//	ANTHROPIC_API_KEY — majburiy (bo'lmasa chat o'chiq)
//	ANTHROPIC_MODEL   — ixtiyoriy
//	ANTHROPIC_API_URL — ixtiyoriy: so'rovlar korporativ shlyuz orqali
//	                    yuborilsa yoki testda soxta server ishlatilsa
func New() *Client {
	model := os.Getenv("ANTHROPIC_MODEL")
	if model == "" {
		model = defaultModel
	}
	url := os.Getenv("ANTHROPIC_API_URL")
	if url == "" {
		url = defaultAPIURL
	}
	mid := os.Getenv("ANTHROPIC_MID_MODEL")
	if mid == "" {
		mid = defaultMidModel
	}
	fast := os.Getenv("ANTHROPIC_FAST_MODEL")
	if fast == "" {
		fast = defaultFastModel
	}
	maxTokens := envInt("ANTHROPIC_MAX_TOKENS", defaultMaxTokens)
	// Faqat "1h" tan olinadi: noma'lum qiymat API xatosi bo'lardi,
	// shuning uchun jimgina sukutga (5 daqiqa) qaytamiz.
	cacheTTL := ""
	if os.Getenv("ANTHROPIC_CACHE_TTL") == cacheTTL1h {
		cacheTTL = cacheTTL1h
	}
	return &Client{
		apiKey:    os.Getenv("ANTHROPIC_API_KEY"),
		model:     model,
		midModel:  mid,
		fastModel: fast,
		url:       url,
		maxTokens: maxTokens,
		cacheTTL:  cacheTTL,
		http:      &http.Client{Timeout: 120 * time.Second},
		glm:       newGLM(maxTokens),
	}
}

// GLMAvailable — arzon provayder sozlanganmi (jurnal uchun).
func (c *Client) GLMAvailable() bool { return c.glm.available() }

// Tiers — amaldagi model darajalari, ishga tushishda jurnalga yozish uchun.
//
// NEGA KERAK: Mid va Cheap darajalari sukut bo'yicha o'chiq va asosiy
// modelga qaytadi. Bu holat jurnalda KO'RINIB TURISHI shart — aks holda
// kimdir muhitga ANTHROPIC_MID_MODEL qo'yib qo'ysa, javoblar boshqa
// modeldan kela boshlaydi va buni faqat sifat pasayganda sezish mumkin.
// Aynan shunday hodisa bir marta yuz bergan.
func (c *Client) Tiers() string {
	// ⚠️ GLM ham HISOBGA OLINADI. U `anthropicModel` dan OLDIN turadi
	// (useGLM), ya'ni ANTHROPIC_FAST_MODEL bo'sh bo'lsa ham arzon
	// darajani o'ziga tortib oladi. Buni e'tiborsiz qoldirgan edik va
	// auditda chiqdi: GLM yoqiq holatda ham satr "hamma so'rov: opus"
	// deb yozardi — ya'ni aynan adashtirish uchun qo'yilgan satr
	// adashtirardi.
	glm := ""
	if c.glm.available() {
		glm = c.glm.model
	}
	if c.midModel == "" && c.fastModel == "" && glm == "" {
		return "hamma so'rov: " + c.model
	}
	var b strings.Builder
	fmt.Fprintf(&b, "asosiy: %s", c.model)
	if c.midModel != "" {
		fmt.Fprintf(&b, " | ⚠️ ma'lumot savollari: %s", c.midModel)
	}
	// GLM arzon darajani fastModel dan OLDIN oladi, shuning uchun
	// ikkalasi sozlangan bo'lsa ham amalda ishlaydigani GLM.
	switch {
	case glm != "":
		fmt.Fprintf(&b, " | ⚠️ bo'sh gap: %s (GLM)", glm)
	case c.fastModel != "":
		fmt.Fprintf(&b, " | ⚠️ bo'sh gap: %s", c.fastModel)
	}
	return b.String()
}

// GLMModel — arzon provayder modeli nomi.
func (c *Client) GLMModel() string {
	if !c.glm.available() {
		return ""
	}
	return c.glm.model
}

// envInt — muhit o'zgaruvchisidan musbat butun son; noto'g'ri bo'lsa sukut.
func envInt(name string, def int) int {
	if v, err := strconv.Atoi(os.Getenv(name)); err == nil && v > 0 {
		return v
	}
	return def
}

// Available — API kaliti mavjudligini bildiradi.
func (c *Client) Available() bool {
	return c.apiKey != ""
}

// Image — suhbatga biriktirilgan rasm (base64, "data:" prefiksisiz).
type Image struct {
	MediaType string `json:"media_type"` // masalan "image/jpeg", "image/png"
	Data      string `json:"data"`       // base64 kodlangan tarkib
}

// Message — suhbat xabari (matn va ixtiyoriy rasmlar).
type Message struct {
	Role    string  `json:"role"` // "user" yoki "assistant"
	Content string  `json:"content"`
	Images  []Image `json:"images,omitempty"`
}

// DIQQAT: `temperature` bu yerda ATAYLAB yo'q.
//
// Javob barqarorligini oshirish uchun temperature=0.2 qo'yib ko'rildi, ammo
// API xato qaytardi: "`temperature` is deprecated for this model"
// (claude-opus-4-8). Shuning uchun javob turg'unligi so'rov parametri bilan
// emas, kontekstni aniq yozish bilan ta'minlanadi — masalan aksiz bo'shlig'i
// chat.formatMatches da ochiq ogohlantirish sifatida chiqariladi.
type apiRequest struct {
	Model     string       `json:"model"`
	MaxTokens int          `json:"max_tokens"`
	System    []systemMsg  `json:"system,omitempty"`
	Messages  []apiMessage `json:"messages"`
	Stream    bool         `json:"stream,omitempty"`
}

// systemMsg — tizim ko'rsatmasi bloki. Massiv ko'rinishi kesh uchun zarur:
// oddiy matn bo'lsa, cache_control ni biriktirib bo'lmaydi.
type systemMsg struct {
	Type         string        `json:"type"` // "text"
	Text         string        `json:"text"`
	CacheControl *cacheControl `json:"cache_control,omitempty"`
}

type cacheControl struct {
	Type string `json:"type"`          // "ephemeral"
	TTL  string `json:"ttl,omitempty"` // "" (5 daqiqa) yoki "1h"
}

// Kesh muddati.
//
// Sukut — 5 daqiqa. MUAMMO shunda: kesh BARCHA foydalanuvchilar uchun
// umumiy (tizim ko'rsatmasi bir xil), lekin 5 daqiqada bironta so'rov
// kelmasa u yo'qoladi va keyingi so'rov keshni QAYTA YOZADI — bu esa
// oddiy kirishdan 1,25 barobar qimmat. Ya'ni siyrak trafikda kesh
// foyda emas, zarar keltiradi.
//
// "1h" buni tuzatadi: soatiga bitta so'rov bo'lsa ham kesh tirik
// qoladi. Yozish qimmatroq (2 barobar), o'qish esa o'sha-o'sha 0,1
// barobar. Shuning uchun ATAYLAB sozlanadigan qilingan — trafik
// oyiga bir necha so'rov bo'lsa, 5 daqiqa arzonroq.
const cacheTTL1h = "1h"

// extendedCacheBeta — 1 soatlik kesh uchun kerak bo'ladigan sarlavha.
const extendedCacheBeta = "extended-cache-ttl-2025-04-11"

// Tizim ko'rsatmasi ~6 000 belgi va HAR SO'ROVDA bir xil. Kesh bo'lmasa,
// u har safar qaytadan to'lanadi. Kesh o'qish narxi asl narxning ~10%i.
//
// Kesh ATAYLAB shu yerda: ko'rsatma barqaror, retrieval bloklari esa
// oxirgi FOYDALANUVCHI xabariga qo'shiladi — ular o'zgaruvchan bo'lgani
// uchun keshga tushmaydi va keshni buzmaydi ham.
const minCacheable = 2048 // belgi; bundan qisqa matnni keshlashning ma'nosi yo'q

// Cacheable — shu ko'rsatma keshlanadimi.
//
// Tashqariga chiqarilgan, chunki ko'rsatmani YOZADIGAN paket (chat) uni
// tekshira olishi kerak: ko'rsatma qisqarib ketsa kesh JIMGINA o'chadi —
// xato bo'lmaydi, faqat har so'rov qimmatlashadi.
func Cacheable(system string) bool { return len(system) >= minCacheable }

// buildRequest — so'rov tanasini yig'adi. Complete va Stream uchun umumiy,
// shunda ikkalasi bir xil model, limit va kontekst bilan ishlaydi.
func (c *Client) buildRequest(model Model, system string, history []Message, stream bool) ([]byte, error) {
	msgs := make([]apiMessage, 0, len(history))
	for _, m := range history {
		msgs = append(msgs, apiMessage{Role: m.Role, Content: buildContent(m)})
	}

	var sys []systemMsg
	if system != "" {
		blk := systemMsg{Type: "text", Text: system}
		if len(system) >= minCacheable {
			blk.CacheControl = &cacheControl{Type: "ephemeral", TTL: c.cacheTTL}
		}
		sys = []systemMsg{blk}
	}

	return json.Marshal(apiRequest{
		Model:     c.anthropicModel(model),
		MaxTokens: c.maxTokens,
		System:    sys,
		Messages:  msgs,
		Stream:    stream,
	})
}

// newHTTPRequest — sarlavhalari qo'yilgan POST so'rov.
func (c *Client) newHTTPRequest(ctx context.Context, body []byte) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("anthropic-version", apiVersion)
	if c.cacheTTL == cacheTTL1h {
		req.Header.Set("anthropic-beta", extendedCacheBeta)
	}
	return req, nil
}

// apiMessage.Content matn (string) yoki bloklar ro'yxati (any) bo'lishi mumkin.
type apiMessage struct {
	Role    string `json:"role"`
	Content any    `json:"content"`
}

type textBlock struct {
	Type string `json:"type"` // "text"
	Text string `json:"text"`
}

type imageBlock struct {
	Type   string      `json:"type"` // "image"
	Source imageSource `json:"source"`
}

type imageSource struct {
	Type      string `json:"type"` // "base64"
	MediaType string `json:"media_type"`
	Data      string `json:"data"`
}

type apiResponse struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	// StopReason — "end_turn", "max_tokens", "stop_sequence"…
	StopReason string   `json:"stop_reason"`
	Usage      apiUsage `json:"usage"`
	Error      *struct {
		Message string `json:"message"`
	} `json:"error"`
}

// stopMaxTokens — javob uzunlik chegarasida to'xtaganini bildiradi.
const stopMaxTokens = "max_tokens"

// truncatedNote — kesilgan javob oxiriga qo'shiladigan ogohlantirish.
//
// NEGA KERAK: chegaraga urilgan javob JIMGINA kesiladi — API xato
// bermaydi, matn shunchaki gap o'rtasida tugaydi. Jonli sinovda boj
// hisobi aynan shunday kesildi va oxirida turgan "kelib chiqish
// sertifikati bo'lmasa boj IKKI BAROBAR" ogohlantirishi yo'qoldi.
// Foydalanuvchi javobni to'liq deb o'qib, kam to'lov hisoblab qolardi.
//
// Markdown ajratgichi bilan boshlanadi — ikkala mijoz ham (web va
// Android) uni alohida blok qilib chizadi.
const truncatedNote = "\n\n---\n\n⚠️ **Javob uzunlik chegarasiga yetdi va TO'LIQ EMAS.** " +
	"Yuqoridagi ro'yxat yoki hisob oxirigacha yozilmagan bo'lishi mumkin — " +
	"savolni bo'lib bering yoki qaysi qismi kerakligini aniq ayting."

// buildContent — xabarni API formatiga o'giradi.
// Rasm bo'lmasa oddiy matn (string), aks holda bloklar ro'yxati qaytariladi.
func buildContent(m Message) any {
	if len(m.Images) == 0 {
		return m.Content
	}
	blocks := make([]any, 0, len(m.Images)+1)
	for _, img := range m.Images {
		blocks = append(blocks, imageBlock{
			Type: "image",
			Source: imageSource{
				Type:      "base64",
				MediaType: img.MediaType,
				Data:      img.Data,
			},
		})
	}
	if m.Content != "" {
		blocks = append(blocks, textBlock{Type: "text", Text: m.Content})
	}
	return blocks
}

// Complete — to'liq javob, asosiy modelda (Claude).
func (c *Client) Complete(ctx context.Context, system string, history []Message) (string, error) {
	return c.CompleteWith(ctx, Full, system, history)
}

// CompleteWith — daraja ko'rsatilgan holda to'liq javob.
//
// Stream bilan bir xil yo'naltirish: oqimli va oqimsiz javob bir xil
// provayderga tushishi kerak, aks holda foydalanuvchi bir xil savolga
// ikki xil sifatdagi javob olardi.
func (c *Client) CompleteWith(ctx context.Context, model Model, system string, history []Message) (string, error) {
	model = pickModel(model, history)

	if c.useGLM(model) {
		return c.glm.Complete(ctx, system, history)
	}
	if !c.Available() {
		return "", ErrNoAPIKey
	}

	body, err := c.buildRequest(model, system, history, false)
	if err != nil {
		return "", err
	}
	req, err := c.newHTTPRequest(ctx, body)
	if err != nil {
		return "", err
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	var out apiResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return "", fmt.Errorf("javobni o'qishda xato: %w", err)
	}
	if out.Error != nil {
		return "", fmt.Errorf("API xatosi: %s", out.Error.Message)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API status %d: %s", resp.StatusCode, string(raw))
	}

	c.reportUsage(model, out.Usage)

	var text strings.Builder
	for _, blk := range out.Content {
		if blk.Type == "text" {
			text.WriteString(blk.Text)
		}
	}
	// Chegarada to'xtagan javob — ochiq aytamiz (izohi truncatedNote da).
	if out.StopReason == stopMaxTokens {
		text.WriteString(truncatedNote)
	}
	return text.String(), nil
}

// ---------------------------------------------------------------- streaming

// StreamFunc — javobning har bir bo'lagi kelganda chaqiriladi.
// Xato qaytarsa, oqim to'xtatiladi (masalan foydalanuvchi ketib qolgan).
type StreamFunc func(chunk string) error

// streamEvent — SSE hodisasining bizga kerakli qismi.
//
// Anthropic oqimi bir necha turdagi hodisa yuboradi (message_start,
// content_block_start, ping, message_stop...). Bizga faqat matn bo'laklari
// kerak: type="content_block_delta", delta.type="text_delta".
type streamEvent struct {
	Type    string   `json:"type"`
	Usage   apiUsage `json:"usage"`
	Message *struct {
		Usage apiUsage `json:"usage"`
	} `json:"message"`
	Delta struct {
		Type string `json:"type"`
		Text string `json:"text"`
		// message_delta hodisasida keladi — oqimda tugash sababi
		// SHU YERDA, javob tanasida emas.
		StopReason string `json:"stop_reason"`
	} `json:"delta"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

// Stream — javobni bo'lak-bo'lak qaytaradi.
//
// NEGA KERAK: to'liq javob 23–49 soniya oladi. Foydalanuvchi shuncha vaqt
// bo'sh ekranga qarab turmasligi uchun matn yozilayotganda ko'rsatiladi.
//
// Xatolar ikki joyda chiqishi mumkin: oqim BOSHLANISHIDAN oldin (HTTP
// status va JSON xato tanasi) va oqim ICHIDA (event: error). Ikkalasi ham
// qayta ishlanadi — aks holda foydalanuvchi yarim javob olib, nima
// bo'lganini bilmay qolardi.
func (c *Client) Stream(ctx context.Context, model Model, system string, history []Message, onChunk StreamFunc) error {
	// Rasm bo'lsa daraja Full ga majburlanadi (GLM rasmni bilmaydi).
	model = pickModel(model, history)

	if c.useGLM(model) {
		return c.glm.Stream(ctx, system, history, onChunk)
	}
	if !c.Available() {
		return ErrNoAPIKey
	}
	body, err := c.buildRequest(model, system, history, true)
	if err != nil {
		return err
	}
	req, err := c.newHTTPRequest(ctx, body)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "text/event-stream")

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Oqim boshlanmasdan xato qaytgan bo'lsa — tana oddiy JSON.
	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		var out apiResponse
		if json.Unmarshal(raw, &out) == nil && out.Error != nil {
			return fmt.Errorf("API xatosi: %s", out.Error.Message)
		}
		return fmt.Errorf("API status %d: %s", resp.StatusCode, string(raw))
	}

	sc := bufio.NewScanner(resp.Body)
	// Bo'laklar kichik, lekin bitta hodisa uzun bo'lib qolsa oqim
	// jim uzilib qolmasligi uchun buferni kengaytiramiz.
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	// Oqimda sarf ikki joyda keladi: message_start da kirish tokenlari
	// (kesh ma'lumoti ham shu yerda), message_delta da chiqish tokenlari.
	var usage apiUsage
	var got bool
	var stopReason string
	for sc.Scan() {
		line := sc.Text()
		// SSE: bizga faqat "data:" qatorlari kerak, "event:" va bo'sh
		// qatorlar tashlab yuboriladi.
		data, ok := strings.CutPrefix(line, "data: ")
		if !ok {
			continue
		}
		if data == "[DONE]" {
			break
		}
		var ev streamEvent
		if err := json.Unmarshal([]byte(data), &ev); err != nil {
			continue // noma'lum hodisa — e'tiborsiz qoldiramiz
		}
		if ev.Error != nil {
			return fmt.Errorf("API xatosi: %s", ev.Error.Message)
		}
		if ev.Message != nil && ev.Message.Usage.InputTokens > 0 {
			usage.InputTokens = ev.Message.Usage.InputTokens
			usage.CacheCreationInputTokens = ev.Message.Usage.CacheCreationInputTokens
			usage.CacheReadInputTokens = ev.Message.Usage.CacheReadInputTokens
		}
		if ev.Usage.OutputTokens > 0 {
			usage.OutputTokens = ev.Usage.OutputTokens
		}
		if ev.Delta.StopReason != "" {
			stopReason = ev.Delta.StopReason
		}
		if ev.Type != "content_block_delta" || ev.Delta.Type != "text_delta" || ev.Delta.Text == "" {
			continue
		}
		got = true
		if err := onChunk(ev.Delta.Text); err != nil {
			return err
		}
	}
	// Chegarada to'xtagan javob — oxiriga ogohlantirish qo'shamiz.
	// Bu ham oddiy bo'lak sifatida ketadi, ya'ni mijoz tomonida
	// alohida ishlov kerak emas (izohi truncatedNote da).
	if stopReason == stopMaxTokens && got {
		if err := onChunk(truncatedNote); err != nil {
			return err
		}
	}
	c.reportUsage(model, usage)

	if err := sc.Err(); err != nil {
		return fmt.Errorf("oqim uzildi: %w", err)
	}
	// Hech narsa kelmasa, chaqiruvchi buni sukut deb o'ylamasligi kerak.
	if !got {
		return errors.New("API bo'sh javob qaytardi")
	}
	return nil
}

// ---------------------------------------------------------------- hisob

// Usage — bitta so'rovning token sarfi.
//
// NEGA KERAK: xarajatni ko'rmasdan boshqarib bo'lmaydi. Ayniqsa
// CacheRead — kesh ishlayotganini shu ko'rsatadi. Kesh ishlamasa,
// CacheRead doim 0 bo'ladi va tizim ko'rsatmasi har safar to'liq
// to'lanadi.
type Usage struct {
	Model        string
	InputTokens  int
	OutputTokens int
	CacheWrite   int // keshga yozilgan (birinchi so'rovda)
	CacheRead    int // keshdan o'qilgan (arzon)
}

// OnUsage — har so'rovdan keyin chaqiriladi (nil bo'lishi mumkin).
// main.go da jurnalga yozish uchun o'rnatiladi.
var OnUsage func(Usage)

type apiUsage struct {
	InputTokens              int `json:"input_tokens"`
	OutputTokens             int `json:"output_tokens"`
	CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
	CacheReadInputTokens     int `json:"cache_read_input_tokens"`
}

func (c *Client) reportUsage(model Model, u apiUsage) {
	reportUsage(c.modelName(model), u)
}

// reportUsage — model NOMI bilan hisobot. Ikkala provayder ham shu orqali
// yozadi, shunda admin paneli ularni bir xil ustunlarda ko'rsatadi.
func reportUsage(model string, u apiUsage) {
	if OnUsage == nil {
		return
	}
	OnUsage(Usage{
		Model:        model,
		InputTokens:  u.InputTokens,
		OutputTokens: u.OutputTokens,
		CacheWrite:   u.CacheCreationInputTokens,
		CacheRead:    u.CacheReadInputTokens,
	})
}
