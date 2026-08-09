package uz.deklarant.ai.data.repository

import kotlinx.coroutines.flow.Flow
import uz.deklarant.ai.data.remote.ChatRequest
import uz.deklarant.ai.data.remote.DeklarantApi
import uz.deklarant.ai.domain.model.BrowseNode
import uz.deklarant.ai.domain.model.BrowsePage
import uz.deklarant.ai.domain.model.ChatMessage
import uz.deklarant.ai.domain.model.ChatMode
import uz.deklarant.ai.domain.model.LawArticle
import uz.deklarant.ai.domain.model.LawDoc
import uz.deklarant.ai.domain.model.LawText
import uz.deklarant.ai.domain.repository.ChatRepository
import uz.deklarant.ai.domain.repository.CodeRepository
import uz.deklarant.ai.domain.repository.LawRepository

// Repozitoriylar INGICHKA: ular API ni chaqiradi va domenga o'giradi.
// Boshqa mantiq yo'q — u ekran modellarida yoki domenda turadi.

internal class ChatRepositoryImpl(
    private val api: DeklarantApi,
) : ChatRepository {

    override fun stream(history: List<ChatMessage>, mode: ChatMode): Flow<String> =
        api.chatStream(
            ChatRequest(
                messages = history.map { it.toDto() },
                mode = mode.wire,
            ),
        )
}

internal class CodeRepositoryImpl(
    private val api: DeklarantApi,
) : CodeRepository {

    override suspend fun browse(section: String?, group: String?, heading: String?): BrowsePage =
        api.browse(section, group, heading).toDomain()

    // Hech narsa topilmasa backend `matches: null` qaytaradi — bo'sh
    // ro'yxat emas. `orEmpty()` shuni yutadi.
    override suspend fun search(query: String, limit: Int): List<BrowseNode> =
        api.search(query, limit).matches.orEmpty().map { it.toDomain() }
}

internal class LawRepositoryImpl(
    private val api: DeklarantApi,
) : LawRepository {

    override suspend fun docs(): List<LawDoc> =
        api.laws().docs.map { it.toDomain() }

    override suspend fun articles(doc: Int): List<LawArticle> =
        api.laws(doc = doc).articles.map { it.toDomain() }

    override suspend fun article(doc: Int, index: Int): LawText {
        val dto = api.laws(doc = doc, index = index).article
            ?: error("modda topilmadi")
        return dto.toDomain()
    }
}
