package uz.deklarant.ai.ui.chat

import cafe.adriel.voyager.core.model.StateScreenModel
import cafe.adriel.voyager.core.model.screenModelScope
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.Job
import kotlinx.coroutines.flow.catch
import kotlinx.coroutines.flow.onCompletion
import kotlinx.coroutines.launch
import uz.deklarant.ai.data.chat.ChatHistoryStore
import uz.deklarant.ai.domain.model.ChatImage
import uz.deklarant.ai.domain.model.ChatMessage
import uz.deklarant.ai.domain.model.ChatMode
import uz.deklarant.ai.domain.repository.ChatRepository
import uz.deklarant.ai.domain.repository.ImageRepository

/**
 * Yuborishga tayyorlangan rasm.
 *
 * `uri` saqlanadi, chunki ko'rinishda base64 ni qayta dekodlashdan
 * ko'ra asl manzildan chizish arzonroq.
 */
data class PendingImage(
    val uri: String,
    val image: ChatImage,
)

/**
 * Chat ekranining holati.
 *
 * `streaming` alohida maydon: javob YOZILAYOTGANDA u xabarlar ro'yxatiga
 * qo'shilmaydi. Aks holda har bo'lakda butun ro'yxat qayta chizilardi.
 */
data class ChatState(
    val messages: List<ChatMessage> = emptyList(),
    val pending: List<PendingImage> = emptyList(),
    val streaming: String = "",
    val isSending: Boolean = false,
    val isAttaching: Boolean = false,
    val mode: ChatMode = ChatMode.Declarant,
    val error: String? = null,
)

class ChatScreenModel(
    private val repository: ChatRepository,
    private val images: ImageRepository,
    private val history: ChatHistoryStore,
) : StateScreenModel<ChatState>(ChatState()) {

    private var job: Job? = null

    init {
        // Saqlangan suhbatni tiklaymiz.
        //
        // ⚠️ Faqat ro'yxat hali BO'SH bo'lsa qo'llanadi: o'qish diskdan
        // keladi va foydalanuvchi shu orada savol yuborib ulgurishi
        // mumkin — u holda tiklangan eski tarix uni bosib ketardi.
        screenModelScope.launch {
            val saved = history.load()
            if (saved.isNotEmpty() && mutableState.value.messages.isEmpty()) {
                mutableState.value = mutableState.value.copy(messages = saved)
            }
        }
    }

    /** Suhbatni diskka yozadi. */
    private fun persist() {
        val snapshot = mutableState.value.messages
        screenModelScope.launch { history.save(snapshot) }
    }

    /**
     * Suhbatni tozalaydi — ekranda ham, diskda ham.
     *
     * Ketayotgan javob ham to'xtatiladi: aks holda u tugagach o'zini
     * bo'shatilgan ro'yxatga qo'shib qo'yardi.
     */
    fun clearHistory() {
        job?.cancel()
        job = null
        mutableState.value = ChatState(mode = mutableState.value.mode)
        screenModelScope.launch { history.clear() }
    }

    fun setMode(mode: ChatMode) {
        mutableState.value = mutableState.value.copy(mode = mode)
    }

    fun dismissError() {
        mutableState.value = mutableState.value.copy(error = null)
    }

    // ------------------------------------------------------------ rasmlar

    /**
     * Tanlangan rasmlarni tayyorlaydi (o'qish, burish, siqish).
     *
     * Ish IO da bajariladi va uzoq davom etishi mumkin, shuning uchun
     * `isAttaching` bilan ko'rsatiladi — aks holda foydalanuvchi
     * tugma ishlamadi deb o'ylardi.
     */
    fun attach(uris: List<String>) {
        if (uris.isEmpty()) return
        mutableState.value = mutableState.value.copy(isAttaching = true, error = null)

        screenModelScope.launch {
            val prepared = mutableListOf<PendingImage>()
            var failure: String? = null

            for (uri in uris) {
                runCatching { images.load(uri) }
                    .onSuccess { prepared += PendingImage(uri, it) }
                    // Bitta rasm o'qilmasa, QOLGANLARI tashlab yuborilmaydi.
                    .onFailure { failure = it.message ?: "Rasmni o'qib bo'lmadi" }
            }

            mutableState.value = mutableState.value.copy(
                pending = mutableState.value.pending + prepared,
                isAttaching = false,
                error = failure,
            )
        }
    }

    fun removeImage(uri: String) {
        mutableState.value = mutableState.value.copy(
            pending = mutableState.value.pending.filterNot { it.uri == uri },
        )
    }

    // ------------------------------------------------------------ yuborish

    /** Savolni biriktirilgan rasmlar bilan yuboradi. */
    fun send(text: String) {
        val content = text.trim()
        val attached = mutableState.value.pending.map { it.image }

        // Rasm YOLG'IZ ham yuborilishi mumkin: "bu nima?" degan savol
        // matnsiz ham ma'noli.
        if ((content.isEmpty() && attached.isEmpty()) || mutableState.value.isSending) return

        val history = mutableState.value.messages +
            ChatMessage(ChatMessage.Role.User, content, attached)

        mutableState.value = mutableState.value.copy(
            messages = history,
            pending = emptyList(),
            streaming = "",
            isSending = true,
            error = null,
        )
        // Savolni DARROV saqlaymiz: javob 30–50 soniya keladi va shu
        // orada ilova o'ldirilsa, uzun savol yo'qolib ketardi.
        persist()
        stream(history)
    }

    /**
     * Xatodan keyin QAYTA urinish.
     *
     * NEGA KERAK: tarmoq uzilsa yoki server javob bermasa, savol
     * tarixda qolardi, lekin uni qayta yuborishning yo'li yo'q edi —
     * foydalanuvchi butun savolni (ko'pincha uzun: kod, miqdor, narx,
     * davlat) qaytadan terishga majbur bo'lardi. Rasm biriktirilgan
     * bo'lsa esa uni qaytadan tanlash kerak edi.
     *
     * Tarix o'zgarmaydi: oxirgi xabar allaqachon foydalanuvchiniki,
     * shuning uchun o'shani qayta yuboramiz.
     */
    fun retry() {
        val history = mutableState.value.messages
        if (history.lastOrNull()?.role != ChatMessage.Role.User) return
        if (mutableState.value.isSending) return

        mutableState.value = mutableState.value.copy(
            streaming = "",
            isSending = true,
            error = null,
        )
        stream(history)
    }

    /** Javob oqimini boshlaydi. `send` va `retry` uchun umumiy yo'l. */
    private fun stream(history: List<ChatMessage>) {
        job = screenModelScope.launch {
            val buffer = StringBuilder()
            repository.stream(history, mutableState.value.mode)
                .catch { e ->
                    // Bekor qilish XATO EMAS — foydalanuvchi o'zi to'xtatdi.
                    if (e is CancellationException) throw e
                    mutableState.value = mutableState.value.copy(
                        error = e.message ?: "Javob olinmadi",
                        isSending = false,
                        streaming = "",
                    )
                }
                .onCompletion { cause ->
                    // Yig'ilgan matn YO'QOLMASLIGI kerak: oqim yarmida
                    // uzilsa ham, kelgan qismi javob sifatida saqlanadi.
                    if (buffer.isNotEmpty()) {
                        mutableState.value = mutableState.value.copy(
                            messages = mutableState.value.messages +
                                ChatMessage(ChatMessage.Role.Assistant, buffer.toString()),
                        )
                        // Javob to'xtatilgan yoki uzilgan bo'lsa ham
                        // saqlanadi — kelgan qismi baribir qimmatli.
                        persist()
                    }
                    if (cause == null || cause is CancellationException) {
                        mutableState.value = mutableState.value.copy(
                            streaming = "",
                            isSending = false,
                        )
                    }
                }
                .collect { chunk ->
                    buffer.append(chunk)
                    mutableState.value = mutableState.value.copy(streaming = buffer.toString())
                }
        }
    }

    /** Ketayotgan javobni to'xtatadi. */
    fun stop() {
        job?.cancel()
        job = null
    }

    override fun onDispose() {
        job?.cancel()
    }
}
