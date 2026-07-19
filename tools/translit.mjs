// O'zbek kirill → lotin transliteratsiyasi.
//
// manba bazasidagi title_uz maydonlari kirillda ("бошқалар"), bizning
// interfeys esa lotinda ("boshqalar"). Shu sabab o'girish kerak.
//
// Nozik joylar:
//   е  — so'z boshida va unlidan keyin "ye", aks holda "e"
//   ц  — so'z boshida "s", aks holda "ts"
//   ь  — tushiriladi
//   ъ  — tutuq belgisi (ʼ)
//   ў/ғ — oʻ / gʻ (modifikator apostrof, oddiy ' emas)

const MAP = {
  а: 'a', б: 'b', в: 'v', г: 'g', д: 'd', ж: 'j', з: 'z', и: 'i',
  й: 'y', к: 'k', л: 'l', м: 'm', н: 'n', о: 'o', п: 'p', р: 'r',
  с: 's', т: 't', у: 'u', ф: 'f', х: 'x', ч: 'ch', ш: 'sh', щ: 'sh',
  ы: 'i', э: 'e', ю: 'yu', я: 'ya', ё: 'yo',
  қ: 'q', ғ: 'gʻ', ҳ: 'h', ў: 'oʻ',
  ь: '', ъ: 'ʼ',
}

const CYRILLIC = /[а-яёқғҳў]/i
const VOWELS = new Set('аеёиоуўыэюя')

// "ц" ning o'zbekcha lotin yozuvidagi qoidasi:
//   so'z boshida        → s   (цемент → sement)
//   UNDOSHdan keyin     → s   (акциз → aksiz, функция → funksiya, станция → stansiya)
//   unlidan keyin       → ts  (декларация → deklaratsiya, позиция → pozitsiya)
// Bu muhim: "aktsiz" deb yozsak, foydalanuvchining "aksiz" so'rovi topmaydi.
function tsRule(prev) {
  if (prev === '') return 's'
  return VOWELS.has(prev) ? 'ts' : 's'
}

/** Bitta so'zni (faqat kichik harflarda) o'giradi. */
function wordToLatin(lower) {
  let out = ''
  for (let i = 0; i < lower.length; i++) {
    const ch = lower[i]
    const prev = i > 0 ? lower[i - 1] : ''
    if (ch === 'е') out += prev === '' || VOWELS.has(prev) ? 'ye' : 'e'
    else if (ch === 'ц') out += tsRule(prev)
    else out += MAP[ch] ?? ch
  }
  return out
}

/** So'zning katta-kichikligini natijaga ko'chiradi. */
function applyCase(src, latin) {
  const letters = [...src].filter((c) => /\p{L}/u.test(c))
  const allUpper = letters.length > 1 && letters.every((c) => c === c.toLocaleUpperCase())
  if (allUpper) return latin.toLocaleUpperCase()
  if (src[0] === src[0].toLocaleUpperCase() && src[0] !== src[0].toLocaleLowerCase())
    return latin.charAt(0).toLocaleUpperCase() + latin.slice(1)
  return latin
}

/** Kirill matnni lotinga o'giradi. Lotin, raqam va belgilar tegilmaydi. */
export function toLatin(src) {
  if (!src) return ''
  // Matnni so'z va so'z-bo'lmagan bo'laklarga ajratamiz.
  return src.replace(/[\p{L}\p{M}]+/gu, (word) => {
    if (!CYRILLIC.test(word)) return word // lotincha so'z — tegmaymiz
    return applyCase(word, wordToLatin(word.toLocaleLowerCase()))
  })
}
