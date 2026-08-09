package uz.deklarant.ai.data.remote

import io.ktor.client.HttpClient
import io.ktor.client.engine.okhttp.OkHttp
import io.ktor.client.plugins.HttpTimeout
import io.ktor.client.plugins.contentnegotiation.ContentNegotiation
import io.ktor.serialization.kotlinx.json.json
import kotlinx.serialization.json.Json

/**
 * HTTP klient — bitta nusxa, butun ilova uchun.
 *
 * NEGA BITTA: har repozitoriy o'z klientini yasasa, ulanish havzasi va
 * ip pullari takrorlanardi. Klient og'ir obyekt, uni qayta ishlatish
 * kerak.
 */
internal object HttpClientFactory {

    /**
     * ignoreUnknownKeys — backend yangi maydon qo'shsa, ESKI ilova
     * ishlashdan to'xtamasin. Aks holda serverni yangilash barcha
     * o'rnatilgan ilovalarni buzardi.
     */
    val json: Json = Json {
        ignoreUnknownKeys = true
        explicitNulls = false
    }

    fun create(): HttpClient = HttpClient(OkHttp) {
        expectSuccess = false // status kodni O'ZIMIZ tekshiramiz

        install(ContentNegotiation) { json(json) }

        install(HttpTimeout) {
            // Ulanish tez bo'lishi kerak — server yo'q bo'lsa darrov bilamiz.
            connectTimeoutMillis = 10_000
            // Javob esa UZOQ: AI javobi 20–50 soniya oladi, oqim esa
            // undan ham uzoqroq davom etishi mumkin.
            requestTimeoutMillis = 180_000
            socketTimeoutMillis = 180_000
        }
    }
}
