package uz.deklarant.ai.di

import android.content.Context
import io.ktor.client.HttpClient
import uz.deklarant.ai.BuildConfig
import uz.deklarant.ai.data.auth.AuthStore
import uz.deklarant.ai.data.chat.ChatHistoryStore
import uz.deklarant.ai.data.image.AndroidImageRepository
import uz.deklarant.ai.data.remote.ClientToken
import uz.deklarant.ai.data.remote.DeklarantApi
import uz.deklarant.ai.data.remote.HttpClientFactory
import uz.deklarant.ai.data.repository.AuthRepositoryImpl
import uz.deklarant.ai.data.repository.ChatRepositoryImpl
import uz.deklarant.ai.data.repository.CodeRepositoryImpl
import uz.deklarant.ai.data.repository.LawRepositoryImpl
import uz.deklarant.ai.domain.repository.AuthRepository
import uz.deklarant.ai.domain.repository.ChatRepository
import uz.deklarant.ai.domain.repository.CodeRepository
import uz.deklarant.ai.domain.repository.ImageRepository
import uz.deklarant.ai.domain.repository.LawRepository

/**
 * Bog'liqliklar yig'iladigan yagona joy (composition root).
 *
 * NEGA FREYMVORK EMAS (Hilt/Koin): bu ilovada uchta repozitoriy va
 * bitta HTTP klient bor. Ular uchun DI kutubxonasi qo'shish — annotatsiya
 * ishlovchisi, qurish vaqti va o'rganish narxini olib keladi, evaziga esa
 * hech narsa bermaydi. Konstruktor orqali uzatish shu miqyosda soddaroq
 * va ochiqroq (KISS).
 *
 * Bog'liqlikni teskari qilish (DIP) baribir saqlanadi: ekran modellari
 * INTERFEYSga bog'lanadi, bu yerda esa amalga oshirilishi ulanadi.
 * Testda boshqa amalga oshirishni berish uchun shu sinf almashtiriladi.
 */
class AppContainer(
    context: Context,
    baseUrl: String = BuildConfig.API_BASE_URL,
) {
    private val httpClient: HttpClient = HttpClientFactory.create()
    private val url = baseUrl.trimEnd('/')

    // Ilova konteksti — Activity niki EMAS: konteyner ilova umriga
    // bog'langan, Activity esa qayta yaratiladi va sizib qolardi.
    private val appContext = context.applicationContext
    private val authStore = AuthStore(appContext)

    private val api = DeklarantApi(
        client = httpClient,
        baseUrl = url,
        token = ClientToken(httpClient, url, BuildConfig.API_KEY, authStore),
    )

    val authRepository: AuthRepository = AuthRepositoryImpl(api, authStore)
    val chatRepository: ChatRepository = ChatRepositoryImpl(api)
    val codeRepository: CodeRepository = CodeRepositoryImpl(api)
    val lawRepository: LawRepository = LawRepositoryImpl(api)

    val imageRepository: ImageRepository = AndroidImageRepository(appContext)

    /** Suhbat diskda — ilova yopilganda yo'qolmasin. */
    val chatHistory = ChatHistoryStore(appContext)

    /** Ilova yopilganda ulanishlarni bo'shatadi. */
    fun close() {
        httpClient.close()
    }
}
