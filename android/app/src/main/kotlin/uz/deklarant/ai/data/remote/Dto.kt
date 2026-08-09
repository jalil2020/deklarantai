package uz.deklarant.ai.data.remote

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

// Tarmoq shakllari (DTO). Domen modellaridan ATAYLAB ajratilgan:
// backend javobi o'zgarsa, o'zgarish shu faylda to'xtaydi.

@Serializable
internal data class ImageDto(
    @SerialName("media_type") val mediaType: String,
    val data: String,
)

@Serializable
internal data class MessageDto(
    val role: String,
    val content: String,
    val images: List<ImageDto>? = null,
)

@Serializable
internal data class ChatRequest(
    val messages: List<MessageDto>,
    val mode: String,
)

/** SSE oqimidagi bitta hodisa. */
@Serializable
internal data class StreamEvent(
    val text: String? = null,
    val error: String? = null,
    val done: Boolean = false,
)

@Serializable
internal data class NodeDto(
    val id: String,
    val title: String = "",
    val count: Int = 0,
    val leaf: Boolean = false,
    @SerialName("import_duty") val importDuty: Double = 0.0,
    val vat: Double = 0.0,
    val unit: String = "",
)

@Serializable
internal data class CrumbDto(
    val level: String,
    val id: String,
    val title: String = "",
)

@Serializable
internal data class BrowseDto(
    val level: String,
    val parent: String? = null,
    val items: List<NodeDto> = emptyList(),
    val path: List<CrumbDto> = emptyList(),
)

@Serializable
internal data class SearchRequest(
    val query: String,
    @SerialName("use_ai") val useAi: Boolean = false,
    /** Taklif ro'yxati uchun. Sukut 5, backend 20 gacha beradi. */
    val limit: Int? = null,
)

@Serializable
internal data class MatchDto(val code: CodeDto, val score: Double = 0.0)

@Serializable
internal data class CodeDto(
    val code: String,
    @SerialName("name_uz") val nameUz: String = "",
    @SerialName("name_ru") val nameRu: String = "",
    @SerialName("import_duty") val importDuty: Double = 0.0,
    val vat: Double = 0.0,
    val unit: String = "",
    /** Kombinatsiyalangan stavkaning qat'iy qismi (dollarda, birlik uchun). */
    @SerialName("import_duty_specific") val specific: Double? = null,
    @SerialName("import_duty_specific_unit") val specificUnit: String? = null,
)

/**
 * Qidiruv javobi.
 *
 * `matches` NULL BO'LISHI MUMKIN: hech narsa topilmasa backend bo'sh
 * ro'yxat emas, `null` qaytaradi. Nullable qilinmasa kotlinx
 * serializatsiyasi xato tashlardi (webda aynan shu nuqson chiqqan).
 */
@Serializable
internal data class SearchResponse(val matches: List<MatchDto>? = null)

@Serializable
internal data class LawDocDto(
    val id: Int,
    val name: String,
    val date: String? = null,
    val lex: String? = null,
    val chunks: Int = 0,
)

@Serializable
internal data class LawArticleDto(
    val doc: Int,
    val index: Int,
    val title: String = "",
    val preview: String = "",
)

@Serializable
internal data class LawChunkDto(
    val doc: Int,
    val name: String = "",
    val date: String? = null,
    val lex: String? = null,
    val title: String = "",
    val text: String = "",
)

@Serializable
internal data class LawsBrowseDto(
    val level: String,
    val docs: List<LawDocDto> = emptyList(),
    val articles: List<LawArticleDto> = emptyList(),
    val article: LawChunkDto? = null,
    @SerialName("doc_name") val docName: String = "",
)

/** Backend xato javobi: {"error": "..."} */
@Serializable
internal data class ErrorDto(val error: String = "")

// ---------------------------------------------------------------- auth

@Serializable
internal data class AuthRequest(
    val login: String,
    val password: String,
    val name: String? = null,
    val role: String? = null,
)

@Serializable
internal data class UserDto(
    val id: String,
    val login: String,
    val name: String = "",
    val role: String,
    @SerialName("daily_quota") val dailyQuota: Int = 0,
    @SerialName("chat_mode") val chatMode: String = "deklarant",
)

@Serializable
internal data class SessionDtoFull(
    val token: String,
    val user: UserDto,
)

@Serializable
internal data class RoleDto(
    val role: String,
    @SerialName("chat_mode") val chatMode: String = "deklarant",
    @SerialName("daily_quota") val dailyQuota: Int = 0,
    @SerialName("self_signup") val selfSignup: Boolean = true,
)

@Serializable
internal data class RolesDto(val roles: List<RoleDto> = emptyList())
