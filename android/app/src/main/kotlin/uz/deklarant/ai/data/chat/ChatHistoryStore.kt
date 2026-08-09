package uz.deklarant.ai.data.chat

import android.content.Context
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import kotlinx.serialization.Serializable
import kotlinx.serialization.json.Json
import uz.deklarant.ai.domain.model.ChatImage
import uz.deklarant.ai.domain.model.ChatMessage
import java.io.File

/**
 * Suhbatni qurilmada saqlaydi.
 *
 * NEGA KERAK: tarix `ChatScreenModel` ichida, ya'ni XOTIRADA edi —
 * ilova yopilsa yoki tizim uni fonda o'ldirsa, butun suhbat yo'qolardi.
 * Deklarant uchun bu og'ir: bitta hisob-kitob uzun savol (kod, miqdor,
 * faktura, transport, davlat) va uzun javob — ularni qaytadan yozish
 * kerak bo'lardi.
 *
 * NEGA FAYL, SharedPreferences EMAS: xabarlarda base64 rasm bo'ladi
 * (bitta invoys ~1,6 MB). SharedPreferences butun XML ni xotirada
 * ushlaydi va har yozishda qayta serializatsiya qiladi — bunday hajm
 * uchun mos emas. AuthStore da SharedPreferences qoladi: u yerda
 * bitta qisqa token.
 *
 * NEGA Room EMAS: bitta ro'yxatni saqlash uchun ma'lumotlar bazasi,
 * annotatsiya ishlovchisi va migratsiyalar oqlanmaydi. So'rov ham,
 * indeks ham kerak emas — tarix butunlay o'qiladi va butunlay yoziladi.
 */
class ChatHistoryStore(context: Context) {

    private val file = File(context.applicationContext.filesDir, FILE_NAME)

    private val json = Json {
        ignoreUnknownKeys = true
        encodeDefaults = true
    }

    /** Saqlangan suhbat. Fayl yo'q yoki buzuq bo'lsa — bo'sh ro'yxat. */
    suspend fun load(): List<ChatMessage> = withContext(Dispatchers.IO) {
        if (!file.exists()) return@withContext emptyList()
        runCatching {
            json.decodeFromString<List<StoredMessage>>(file.readText())
                .map(StoredMessage::toDomain)
        }.getOrElse {
            // Buzuq fayl butun chatni ishdan chiqarmasligi kerak.
            // Uni o'chirib, toza boshlaymiz.
            file.delete()
            emptyList()
        }
    }

    /**
     * Suhbatni yozadi.
     *
     * Yozishdan oldin QISQARTIRILADI (pastdagi izohga qarang) — aks holda
     * bir necha invoys surati bilan fayl o'nlab megabaytga chiqardi.
     */
    suspend fun save(messages: List<ChatMessage>) = withContext(Dispatchers.IO) {
        runCatching {
            if (messages.isEmpty()) {
                file.delete()
                return@runCatching
            }
            val text = json.encodeToString(trim(messages).map(StoredMessage::from))
            // Vaqtinchalik faylga yozib, keyin ko'chiramiz: yozish
            // o'rtasida ilova o'ldirilsa, YARIM fayl qolib ketardi va
            // keyingi ochilishda tarix butunlay yo'qolardi.
            val tmp = File(file.parentFile, "$FILE_NAME.tmp")
            tmp.writeText(text)
            if (!tmp.renameTo(file)) {
                file.writeText(text)
                tmp.delete()
            }
        }
        Unit
    }

    /** Tarixni o'chiradi (foydalanuvchi tozalaganda va chiqishda). */
    suspend fun clear() = withContext(Dispatchers.IO) {
        file.delete()
        Unit
    }

    /**
     * Saqlash uchun qisqartirish.
     *
     * Ikki chegara, ATAYLAB har xil o'lchovda:
     *
     *	xabarlar soni — eski suhbat cheksiz o'smasin
     *	rasmlar soni  — hajm shu yerda to'planadi (bitta rasm ~1,6 MB)
     *
     * Rasm HAJM bo'yicha emas, SONI bo'yicha cheklanadi: hajm bo'yicha
     * hisoblansa, bitta katta surat oldidagi butun yozishmani siqib
     * chiqarardi. Bu xatoga backend tarixini qisqartirishda ham yo'l
     * qo'yilgan edi va test aynan shuni ushlagan.
     *
     * Eski rasm o'chirilganda xabar matniga eslatma qo'shiladi — aks
     * holda foydalanuvchi "men-ku surat yuborgandim" deb hayron bo'lardi.
     */
    private fun trim(messages: List<ChatMessage>): List<ChatMessage> {
        val recent = messages.takeLast(MAX_MESSAGES)

        // Oxirgi rasmli xabarlarning o'rnini belgilaymiz.
        val keepImagesAt = recent
            .withIndex()
            .filter { it.value.images.isNotEmpty() }
            .map { it.index }
            .takeLast(MAX_IMAGE_MESSAGES)
            .toSet()

        return recent.mapIndexed { i, m ->
            when {
                m.images.isEmpty() || i in keepImagesAt -> m
                else -> m.copy(
                    images = emptyList(),
                    content = (m.content + " [surat saqlanmadi]").trim(),
                )
            }
        }
    }

    private companion object {
        const val FILE_NAME = "chat-history.json"

        /** Saqlanadigan xabarlar soni. */
        const val MAX_MESSAGES = 60

        /** Suratli xabarlar soni — hajm shu yerda to'planadi. */
        const val MAX_IMAGE_MESSAGES = 2
    }
}

// ------------------------------------------------------- saqlash shakli

/**
 * Diskdagi shakl.
 *
 * Domen modeli ATAYLAB annotatsiyalanmaydi: `domain` toza Kotlin
 * bo'lib qoladi va serializatsiya kutubxonasini bilmaydi. Fayl
 * formati o'zgarsa, domen o'zgarmaydi.
 */
@Serializable
private data class StoredMessage(
    val role: String,
    val content: String,
    val images: List<StoredImage> = emptyList(),
) {
    fun toDomain() = ChatMessage(
        role = if (role == "user") ChatMessage.Role.User else ChatMessage.Role.Assistant,
        content = content,
        images = images.map { ChatImage(it.mediaType, it.data) },
    )

    companion object {
        fun from(m: ChatMessage) = StoredMessage(
            role = if (m.role == ChatMessage.Role.User) "user" else "assistant",
            content = m.content,
            images = m.images.map { StoredImage(it.mediaType, it.data) },
        )
    }
}

@Serializable
private data class StoredImage(val mediaType: String, val data: String)
