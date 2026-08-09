package uz.deklarant.ai.domain.repository

import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.StateFlow
import uz.deklarant.ai.domain.model.BrowseNode
import uz.deklarant.ai.domain.model.BrowsePage
import uz.deklarant.ai.domain.model.ChatImage
import uz.deklarant.ai.domain.model.ChatMessage
import uz.deklarant.ai.domain.model.ChatMode
import uz.deklarant.ai.domain.model.LawArticle
import uz.deklarant.ai.domain.model.LawDoc
import uz.deklarant.ai.domain.model.LawText
import uz.deklarant.ai.domain.model.Role
import uz.deklarant.ai.domain.model.RoleInfo
import uz.deklarant.ai.domain.model.User

// Repozitoriy INTERFEYSLARI domenda turadi, amalga oshirilishi esa
// data qatlamida (Dependency Inversion).
//
// Ekran modellari faqat shu interfeyslarni biladi — shuning uchun
// testda soxta amalga oshirish berish uchun hech narsani o'zgartirish
// kerak emas.

/**
 * Kirish va ro'yxatdan o'tish.
 *
 * NEGA StateFlow: kirgan foydalanuvchi ILOVA BO'YLAB kerak — chat
 * ekrani uni talab qiladi, menyu esa ismini ko'rsatadi. Har biri
 * alohida so'rasa, ular bir-biridan orqada qolardi.
 */
interface AuthRepository {
    val user: StateFlow<User?>

    /** Saqlangan token hali amaldami — ilova ochilishida chaqiriladi. */
    suspend fun restore()

    suspend fun login(login: String, password: String): User
    suspend fun register(login: String, password: String, name: String, role: Role): User
    suspend fun logout()

    /** Ro'yxatdan o'tishda tanlanadigan rollar (kvota va uslub bilan). */
    suspend fun roles(): List<RoleInfo>
}

interface ChatRepository {
    /**
     * Javobni BO'LAK-BO'LAK qaytaradi.
     *
     * NEGA Flow: to'liq javob 20–50 soniya oladi. Foydalanuvchi shuncha
     * vaqt bo'sh ekranga qaramasligi uchun matn yozilayotganda ko'rsatiladi.
     */
    fun stream(history: List<ChatMessage>, mode: ChatMode): Flow<String>
}

interface CodeRepository {
    /** Ierarxiya bo'ylab ko'rish. Hech qaysi parametr berilmasa — bo'limlar. */
    suspend fun browse(section: String? = null, group: String? = null, heading: String? = null): BrowsePage

    /**
     * Matnli qidiruv — kod raqami ham, tovar nomi ham.
     *
     * `limit` taklif ro'yxati uchun: sukut 5 ta variant kamlik qiladi.
     */
    suspend fun search(query: String, limit: Int = 5): List<BrowseNode>
}

interface LawRepository {
    suspend fun docs(): List<LawDoc>
    suspend fun articles(doc: Int): List<LawArticle>
    suspend fun article(doc: Int, index: Int): LawText
}

/**
 * Rasmni suhbatga tayyorlaydi.
 *
 * NEGA DOMENDA INTERFEYS: rasm o'qish va siqish Android'ga xos ish
 * (ContentResolver, Bitmap, EXIF). Ekran modeli ularni bilishi shart
 * emas — u faqat "manzildan rasm yasab ber" deydi.
 *
 * `uri` — satr, Android turi emas: domen platformadan xoli qoladi va
 * testda oddiy satr bilan ishlash mumkin.
 */
interface ImageRepository {
    suspend fun load(uri: String): ChatImage
}
