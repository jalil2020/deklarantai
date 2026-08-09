package uz.deklarant.ai.data.remote

import io.ktor.client.HttpClient
import io.ktor.client.call.body
import io.ktor.client.request.get
import io.ktor.client.request.header
import io.ktor.client.request.parameter
import io.ktor.client.request.post
import io.ktor.client.request.preparePost
import io.ktor.client.request.setBody
import io.ktor.client.statement.HttpResponse
import io.ktor.client.statement.bodyAsChannel
import io.ktor.client.statement.bodyAsText
import io.ktor.http.ContentType
import io.ktor.http.HttpHeaders
import io.ktor.http.HttpStatusCode
import io.ktor.http.contentType
import io.ktor.http.isSuccess
import io.ktor.utils.io.ByteReadChannel
import io.ktor.utils.io.readUTF8Line
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.FlowCollector
import kotlinx.coroutines.flow.flow

/**
 * Backend bilan yagona aloqa nuqtasi.
 *
 * Repozitoriylar faqat shu sinf orqali tarmoqqa chiqadi — manzil yasash
 * va xato o'girish bir joyda (DRY).
 */
internal class DeklarantApi(
    private val client: HttpClient,
    private val baseUrl: String,
    private val token: ClientToken,
) {

    // ------------------------------------------------------------ chat

    /**
     * Chat javobini SSE oqimi sifatida qaytaradi.
     *
     * Backend formati: har satr `data: {"text":"..."}`.
     * Xato oqim ICHIDA ham kelishi mumkin — o'shanda HTTP status
     * allaqachon 200 bo'lgan bo'ladi, shuning uchun `error` maydonini
     * tashlab yuborish mumkin emas: foydalanuvchi yarim javob olib,
     * nima bo'lganini bilmay qolardi.
     */
    fun chatStream(body: ChatRequest): Flow<String> = flow {
        // Qayta urinish XAVFSIZ: 401 oqim boshlanishidan oldin
        // aniqlanadi, ya'ni bironta bo'lak chiqarilmagan bo'ladi.
        retryingOn401 { key -> stream(key, body) }
    }

    private suspend fun FlowCollector<String>.stream(key: String, body: ChatRequest) {
        client.preparePost("$baseUrl/api/chat/stream") {
            contentType(ContentType.Application.Json)
            header(HttpHeaders.Accept, "text/event-stream")
            header(CLIENT_HEADER, key)
            setBody(body)
        }.execute { response ->
            if (response.status == HttpStatusCode.Unauthorized) throw UnauthorizedException()
            if (!response.status.isSuccess()) {
                throw ApiException(errorMessage(response))
            }
            val channel: ByteReadChannel = response.bodyAsChannel()
            while (true) {
                val line = channel.readUTF8Line() ?: break
                val payload = line.removePrefix("data: ").takeIf { it != line } ?: continue
                if (payload == "[DONE]") break

                val event = runCatching {
                    HttpClientFactory.json.decodeFromString(StreamEvent.serializer(), payload)
                }.getOrNull() ?: continue // noma'lum hodisa — e'tiborsiz

                event.error?.let { throw ApiException(it) }
                if (event.done) return@execute
                event.text?.let { if (it.isNotEmpty()) emit(it) }
            }
        }
    }

    // ------------------------------------------------------------ kirish

    // Ro'yxatdan o'tish va kirish belgisiz ketadi — belgi aynan shu
    // yerdan olinadi.

    suspend fun register(body: AuthRequest): SessionDtoFull =
        client.post("$baseUrl/api/auth/register") {
            contentType(ContentType.Application.Json)
            setBody(body)
        }.requireOk()

    suspend fun login(body: AuthRequest): SessionDtoFull =
        client.post("$baseUrl/api/auth/login") {
            contentType(ContentType.Application.Json)
            setBody(body)
        }.requireOk()

    suspend fun me(key: String): UserDto =
        client.get("$baseUrl/api/auth/me") { header(CLIENT_HEADER, key) }.requireOk()

    suspend fun logout(key: String) {
        client.post("$baseUrl/api/auth/logout") { header(CLIENT_HEADER, key) }
    }

    suspend fun roles(): RolesDto = client.get("$baseUrl/api/auth/roles").requireOk()

    // ------------------------------------------------------------ kodlar

    suspend fun browse(section: String?, group: String?, heading: String?): BrowseDto =
        client.get("$baseUrl/api/hscode/browse") {
            // Faqat ENG CHUQUR parametr yuboriladi — backend darajani
            // shunga qarab aniqlaydi.
            when {
                heading != null -> parameter("heading", heading)
                group != null -> parameter("group", group)
                section != null -> parameter("section", section)
            }
        }.requireOk()

    suspend fun search(query: String, limit: Int): SearchResponse =
        client.post("$baseUrl/api/hscode/search") {
            contentType(ContentType.Application.Json)
            setBody(SearchRequest(query = query, limit = limit))
        }.requireOk()

    // ------------------------------------------------------------ qonunlar

    suspend fun laws(doc: Int? = null, index: Int? = null): LawsBrowseDto =
        client.get("$baseUrl/api/laws/browse") {
            doc?.let { parameter("doc", it) }
            index?.let { parameter("i", it) }
        }.requireOk()

    // ------------------------------------------------------------ yordamchi

    /**
     * Belgi bilan bajaradi; 401 kelsa BIR MARTA yangilab qayta uradi.
     *
     * NEGA KERAK: server qayta ishga tushganda (yoki token muddati
     * tugaganda) eski belgi kuchini yo'qotadi. Foydalanuvchi buni
     * "xato" ko'rinishida sezmasligi kerak.
     */
    private suspend fun <T> retryingOn401(block: suspend (String) -> T): T =
        try {
            block(token.get())
        } catch (_: UnauthorizedException) {
            if (!token.refreshable) throw ApiException("Mijoz kaliti qabul qilinmadi")
            block(token.get(refresh = true))
        }

    private suspend inline fun <reified T> HttpResponse.requireOk(): T {
        if (status == HttpStatusCode.Unauthorized) throw UnauthorizedException()
        if (!status.isSuccess()) throw ApiException(errorMessage(this))
        return body()
    }

    /**
     * Xato matnini javob tanasidan oladi.
     *
     * Backend `{"error": "..."}` qaytaradi. U o'qib bo'lmasa, hech
     * bo'lmasa status kod ko'rsatiladi — "nimadir xato" degan
     * foydasiz xabardan yaxshiroq.
     */
    private suspend fun errorMessage(response: HttpResponse): String {
        val raw = runCatching { response.bodyAsText() }.getOrNull().orEmpty()
        val parsed = runCatching {
            HttpClientFactory.json.decodeFromString(ErrorDto.serializer(), raw).error
        }.getOrNull()
        return parsed?.takeIf { it.isNotBlank() }
            ?: "Server xatosi (${response.status.value})"
    }
}

/** API dan kelgan, foydalanuvchiga ko'rsatiladigan xato. */
internal open class ApiException(message: String) : Exception(message)

/**
 * Mijoz belgisi tan olinmadi (401).
 *
 * ApiException dan meros: qayta urinish ham yordam bermasa, xabar
 * baribir foydalanuvchiga ko'rsatiladigan holatda bo'ladi.
 */
internal class UnauthorizedException : ApiException("Mijoz belgisi qabul qilinmadi")
