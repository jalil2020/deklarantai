package uz.deklarant.ai.data.remote

import io.ktor.client.HttpClient
import io.ktor.client.call.body
import io.ktor.client.request.post
import kotlinx.coroutines.sync.Mutex
import kotlinx.coroutines.sync.withLock
import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable
import uz.deklarant.ai.data.auth.AuthStore

/** Belgi shu sarlavhada yuboriladi (backend: internal/api/auth.go). */
internal const val CLIENT_HEADER = "X-API-Key"

/**
 * Mijoz belgisi — chat endpointlariga kirish uchun.
 *
 * Ikki manba:
 *
 *	staticKey — qurish vaqtida berilgan doimiy kalit (`-PapiKey`).
 *	            Backend uni API_KEYS ro'yxatidan taniydi.
 *	/api/session — anonim token. Kalit berilmaganda ishlatiladi.
 *
 * Token ATAYLAB diskka saqlanmaydi: uni olish bitta arzon so'rov, va
 * saqlangan token baribir muddati bilan eskirardi — ya'ni saqlash
 * yangilash mantiqini yo'qotmasdan hech narsa qo'shmaydi.
 */
internal class ClientToken(
    private val client: HttpClient,
    private val baseUrl: String,
    private val staticKey: String,
    private val auth: AuthStore,
) {
    // Ilova ochilishida bir necha so'rov bir vaqtda ketishi mumkin —
    // qulf tokenning bir marta olinishini kafolatlaydi.
    private val mutex = Mutex()
    private var cached: String? = null

    suspend fun get(refresh: Boolean = false): String {
        // Kirgan foydalanuvchi tokeni ENG USTUN: chat faqat shu bilan
        // ishlaydi va kvota ham unga bog'langan.
        auth.token.takeIf { it.isNotEmpty() }?.let { return it }
        if (staticKey.isNotEmpty()) return staticKey
        return mutex.withLock {
            if (!refresh) {
                cached?.let { return@withLock it }
            }
            val issued: SessionDto = client.post("$baseUrl/api/session").body()
            issued.token.also { cached = it }
        }
    }

    /**
     * Anonim tokenni yangilash mantiqiymi.
     *
     * Doimiy kalit yoki kirgan foydalanuvchi bo'lsa — yo'q: ularda 401
     * "token eskirdi" degani emas, "ruxsat yo'q" degani.
     */
    val refreshable: Boolean get() = staticKey.isEmpty() && auth.token.isEmpty()
}

@Serializable
internal data class SessionDto(
    val token: String,
    @SerialName("expires_at") val expiresAt: String? = null,
)
