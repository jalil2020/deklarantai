// Logo belgisini asl rasmdan kesib, kerakli o'lchamlarga tayyorlaydi.
//
// NEGA DASTUR: belgi qo'lda qayta chizilmaydi — dizayner bergan rasmning
// O'ZI ishlatiladi. Qo'lda kesish/kichraytirish takrorlanmaydi va har
// safar boshqacha chiqadi; shu tufayli barcha ko'chirmalar bitta buyruq
// bilan qaytadan yasaladi.
//
//	go run tools/logo/main.go <asl-rasm.png>
//
// Uch ish qiladi:
//  1. Matnni kesib tashlaydi — belgi bilan matn orasidagi bo'sh satrlar
//     chizig'i bo'yicha (matn joyi qo'lda kiritilmaydi).
//  2. Oq fonni shaffofga o'giradi — FAQAT chetdan quyilib. Robot boshi
//     ichidagi oq rang saqlanadi, chunki u belgining bir qismi.
//  3. Maydon o'rtachasi bilan kichraytiradi — kichik ikonkada tishlar
//     chiqmasin.
package main

import (
	"fmt"
	"image"
	"image/png"
	"math"
	"os"
	"path/filepath"
)

// Fon chegaralari. lumOpaque dan qorong'i piksel — belgi; lumClear dan
// yorug'i — toza fon; oralig'i tekislangan chet (yumshoq o'tish).
const (
	lumOpaque = 232.0
	lumClear  = 250.0
)

// Adaptive ikonkada belgi 108dp maydonning shuncha ulushini egallaydi.
//
// 0.52 QO'LDA TANLANMAGAN, hisoblangan: belgi markazidan eng uzoq nuqta —
// pastki-o'ngdagi tasdiq doirasi, u belgi o'lchamining 0.627 radiusida.
// Dumaloq niqob 108dp dan 72dp ni ko'rsatadi, ya'ni radius 36dp:
//
//	0.627 x safeZone x 108 <= 36  =>  safeZone <= 0.53
//
// 0.62 da tasdiq belgisi kesilib qolgan edi.
const safeZone = 0.52

type target struct {
	path string
	size int
	// pad — belgi atrofidagi bo'sh joy ulushi (adaptive ikonka uchun).
	pad float64
	// bg — orqa fon; nil bo'lsa shaffof qoladi.
	bg *[3]uint8
}

var white = [3]uint8{0xFF, 0xFF, 0xFF}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "foydalanish: go run tools/logo/main.go <asl-rasm.png>")
		os.Exit(2)
	}
	src, err := load(os.Args[1])
	check(err)

	mark := cut(src)
	fmt.Printf("belgi kesildi: %dx%d\n", mark.Bounds().Dx(), mark.Bounds().Dy())

	fgPad := (1 - safeZone) / 2
	targets := []target{
		// Android — launcher ikonkasi (adaptive, minSdk 26 shuning uchun
		// eski PNG mipmap lar kerak emas).
		{"android/app/src/main/res/mipmap-mdpi/ic_launcher_foreground.png", 108, fgPad, nil},
		{"android/app/src/main/res/mipmap-hdpi/ic_launcher_foreground.png", 162, fgPad, nil},
		{"android/app/src/main/res/mipmap-xhdpi/ic_launcher_foreground.png", 216, fgPad, nil},
		{"android/app/src/main/res/mipmap-xxhdpi/ic_launcher_foreground.png", 324, fgPad, nil},
		{"android/app/src/main/res/mipmap-xxxhdpi/ic_launcher_foreground.png", 432, fgPad, nil},
		// Android — ilova ichidagi belgi (chat bo'sh ekranida).
		{"android/app/src/main/res/drawable-nodpi/logo_mark.png", 512, 0, nil},

		// Web.
		{"frontend/public/logo.png", 512, 0, nil},
		{"frontend/public/favicon.png", 64, 0, nil},
		// iOS shaffoflikni qora rangga to'ldiradi — oq fon beramiz.
		{"frontend/public/apple-touch-icon.png", 180, 0.06, &white},
	}

	root, err := repoRoot()
	check(err)

	for _, t := range targets {
		out := square(mark, t.size, t.pad, t.bg)
		path := filepath.Join(root, filepath.FromSlash(t.path))
		check(os.MkdirAll(filepath.Dir(path), 0o755))
		check(save(path, out))
		fmt.Printf("  %-64s %dpx\n", t.path, t.size)
	}
}

// cut — matnni kesib tashlaydi va belgi atrofidagi oq joyni olib,
// shaffof fonli RGBA qaytaradi.
func cut(src image.Image) *image.RGBA {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()

	lum := make([]float64, w*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			r, g, bl, _ := src.At(b.Min.X+x, b.Min.Y+y).RGBA()
			// Eng qorong'i kanal — rangli piksel (masalan to'q ko'k)
			// fon deb hisoblanmasligi uchun.
			lum[y*w+x] = float64(min3(r, g, bl)) / 257
		}
	}

	rowEmpty := func(y int) bool {
		for x := 0; x < w; x++ {
			if lum[y*w+x] < lumOpaque {
				return false
			}
		}
		return true
	}

	// Belgi tugagan joy: mazmun boshlangandan keyingi birinchi uzun
	// bo'sh satrlar oralig'i. Matn shundan pastda qoladi.
	//
	// gapRows — belgi ichidagi tasodifiy bo'sh satrdan farqlash uchun.
	const gapRows = 20
	cutY, seen, gap := h, false, 0
	for y := 0; y < h; y++ {
		if rowEmpty(y) {
			if seen {
				gap++
				if gap >= gapRows {
					cutY = y - gap + 1
					break
				}
			}
			continue
		}
		seen, gap = true, 0
	}

	// Chetdan quyish: faqat tashqi oq fon shaffof bo'ladi.
	clear := make([]bool, w*h)
	stack := make([]int, 0, w*4)
	push := func(x, y int) {
		if x < 0 || y < 0 || x >= w || y >= cutY {
			return
		}
		i := y*w + x
		if clear[i] || lum[i] < lumOpaque {
			return
		}
		clear[i] = true
		stack = append(stack, i)
	}
	for x := 0; x < w; x++ {
		push(x, 0)
		push(x, cutY-1)
	}
	for y := 0; y < cutY; y++ {
		push(0, y)
		push(w-1, y)
	}
	for len(stack) > 0 {
		i := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		x, y := i%w, i/w
		push(x-1, y)
		push(x+1, y)
		push(x, y-1)
		push(x, y+1)
	}

	// Belgi chegarasi — shaffof bo'lmagan piksellar bo'yicha.
	minX, minY, maxX, maxY := w, cutY, -1, -1
	for y := 0; y < cutY; y++ {
		for x := 0; x < w; x++ {
			if clear[y*w+x] && lum[y*w+x] >= lumClear {
				continue
			}
			if x < minX {
				minX = x
			}
			if x > maxX {
				maxX = x
			}
			if y < minY {
				minY = y
			}
			if y > maxY {
				maxY = y
			}
		}
	}
	if maxX < minX || maxY < minY {
		panic("belgi topilmadi: rasm butunlay oq")
	}

	out := image.NewRGBA(image.Rect(0, 0, maxX-minX+1, maxY-minY+1))
	for y := minY; y <= maxY; y++ {
		for x := minX; x <= maxX; x++ {
			i := y*w + x
			r, g, bl, a := src.At(b.Min.X+x, b.Min.Y+y).RGBA()
			alpha := float64(a) / 257
			if clear[i] {
				// Tekislangan chet: to'liq oq — ko'rinmas, qorong'iroq —
				// ko'rinadi. Shu tufayli chegara silliq chiqadi.
				k := (lumClear - lum[i]) / (lumClear - lumOpaque)
				alpha *= clamp01(k)
			}
			o := out.PixOffset(x-minX, y-minY)
			// premultiplied alpha — image/png shuni kutadi.
			out.Pix[o+0] = scale(r, alpha)
			out.Pix[o+1] = scale(g, alpha)
			out.Pix[o+2] = scale(bl, alpha)
			out.Pix[o+3] = uint8(alpha + 0.5)
		}
	}
	return out
}

// square — belgini kvadrat maydonga markazlab, maydon o'rtachasi bilan
// kichraytiradi.
//
// Maydon o'rtachasi (bilinear emas): 640px dan 48px ga tushirilganda
// bilinear piksellarni tashlab ketadi va ingichka chiziqlar yo'qoladi.
func square(src *image.RGBA, size int, pad float64, bg *[3]uint8) *image.RGBA {
	inner := int(float64(size) * (1 - 2*pad))
	if inner < 1 {
		inner = 1
	}
	sw, sh := src.Bounds().Dx(), src.Bounds().Dy()

	// Nisbat saqlanadi: belgi cho'zilmasin.
	scaleF := math.Min(float64(inner)/float64(sw), float64(inner)/float64(sh))
	dw, dh := int(float64(sw)*scaleF), int(float64(sh)*scaleF)
	offX, offY := (size-dw)/2, (size-dh)/2

	dst := image.NewRGBA(image.Rect(0, 0, size, size))
	if bg != nil {
		for i := 0; i < len(dst.Pix); i += 4 {
			dst.Pix[i+0], dst.Pix[i+1], dst.Pix[i+2], dst.Pix[i+3] = bg[0], bg[1], bg[2], 0xFF
		}
	}

	for y := 0; y < dh; y++ {
		y0, y1 := boxRange(y, dh, sh)
		for x := 0; x < dw; x++ {
			x0, x1 := boxRange(x, dw, sw)

			var sr, sg, sb, sa, n float64
			for sy := y0; sy < y1; sy++ {
				for sx := x0; sx < x1; sx++ {
					o := src.PixOffset(sx, sy)
					sr += float64(src.Pix[o+0])
					sg += float64(src.Pix[o+1])
					sb += float64(src.Pix[o+2])
					sa += float64(src.Pix[o+3])
					n++
				}
			}
			if n == 0 {
				continue
			}
			sr, sg, sb, sa = sr/n, sg/n, sb/n, sa/n

			o := dst.PixOffset(offX+x, offY+y)
			if bg == nil {
				dst.Pix[o+0], dst.Pix[o+1], dst.Pix[o+2], dst.Pix[o+3] =
					uint8(sr+0.5), uint8(sg+0.5), uint8(sb+0.5), uint8(sa+0.5)
				continue
			}
			// Fon ustiga qo'yamiz (manba premultiplied).
			k := 1 - sa/255
			dst.Pix[o+0] = uint8(sr + float64(bg[0])*k + 0.5)
			dst.Pix[o+1] = uint8(sg + float64(bg[1])*k + 0.5)
			dst.Pix[o+2] = uint8(sb + float64(bg[2])*k + 0.5)
			dst.Pix[o+3] = 0xFF
		}
	}
	return dst
}

// boxRange — chiqish pikseliga to'g'ri keladigan manba oralig'i.
func boxRange(i, dstLen, srcLen int) (int, int) {
	a := i * srcLen / dstLen
	b := (i + 1) * srcLen / dstLen
	if b <= a {
		b = a + 1
	}
	if b > srcLen {
		b = srcLen
	}
	return a, b
}

func load(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return png.Decode(f)
}

func save(path string, img image.Image) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}

// repoRoot — go.mod emas, `tools` papkasi bo'yicha: ildizda go.mod yo'q.
func repoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "tools", "logo")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("loyiha ildizi topilmadi (tools/logo yo'q)")
		}
		dir = parent
	}
}

func min3(a, b, c uint32) uint32 {
	if b < a {
		a = b
	}
	if c < a {
		a = c
	}
	return a
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func scale(v uint32, alpha float64) uint8 {
	return uint8(float64(v)/257*(alpha/255) + 0.5)
}

func check(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "xato:", err)
		os.Exit(1)
	}
}
