package uz.deklarant.ai.domain.model

// Domen modellari — TOZA Kotlin. Android ham, tarmoq ham, JSON ham
// bu yerga kirmaydi.
//
// NEGA: shu tufayli tarmoq javobi shakli o'zgarsa (DTO), ekran kodi
// tegilmaydi — o'zgarish data qatlamida to'xtaydi.

/** Javob uslubi. Faktlar ikkalasida bir xil, farq faqat tushuntirishda. */
enum class ChatMode(val wire: String) {
    Declarant("deklarant"),
    Business("tadbirkor"),
}

/** Suhbatga biriktirilgan rasm (base64, "data:" prefiksisiz). */
data class ChatImage(
    val mediaType: String,
    val data: String,
)

data class ChatMessage(
    val role: Role,
    val content: String,
    val images: List<ChatImage> = emptyList(),
) {
    enum class Role { User, Assistant }
}

/** TIF TN ierarxiyasining bitta tuguni: bo'lim, guruh, pozitsiya yoki kod. */
data class BrowseNode(
    val id: String,
    val title: String,
    val count: Int = 0,
    val isLeaf: Boolean = false,
    val importDuty: Double = 0.0,
    val vat: Double = 0.0,
    val unit: String = "",
    /**
     * Kombinatsiyalangan stavkaning QAT'IY qismi — dollarda, bitta
     * birlik uchun: «10%, lekin 1 kg uchun 0,5 dollardan kam emas».
     * 13 142 koddan 1 555 tasida bor.
     *
     * null = yo'q. Faqat qidiruv natijasida to'ladi; ierarxiya bargida
     * bu ma'lumot saqlanmaydi.
     */
    val specific: Double? = null,
    val specificUnit: String? = null,
)

/** Yuqoriga qaytish zanjirining bo'g'ini. */
data class Crumb(
    val level: BrowseLevel,
    val id: String,
    val title: String,
)

/**
 * Ierarxiya darajasi.
 *
 * Sanab o'tilgan tur ishlatiladi, satr emas: noto'g'ri daraja nomi
 * kompilyatsiyada ushlanadi, ish vaqtida emas.
 */
enum class BrowseLevel { Sections, Groups, Headings, Codes }

data class BrowsePage(
    val level: BrowseLevel,
    val parent: String?,
    val items: List<BrowseNode>,
    val path: List<Crumb>,
)

/** Qonun korpusidagi hujjat. */
data class LawDoc(
    val id: Int,
    val name: String,
    val date: String?,
    val lex: String?,
    val chunks: Int,
)

/** Hujjat ichidagi modda (ro'yxat uchun — faqat matn boshi). */
data class LawArticle(
    val doc: Int,
    val index: Int,
    val title: String,
    val preview: String,
)

/** Moddaning to'liq matni. */
data class LawText(
    val docName: String,
    val date: String?,
    val lex: String?,
    val title: String,
    val text: String,
)
