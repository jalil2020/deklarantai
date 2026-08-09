package uz.deklarant.ai.data.repository

import uz.deklarant.ai.data.remote.BrowseDto
import uz.deklarant.ai.data.remote.CrumbDto
import uz.deklarant.ai.data.remote.ImageDto
import uz.deklarant.ai.data.remote.LawArticleDto
import uz.deklarant.ai.data.remote.LawChunkDto
import uz.deklarant.ai.data.remote.LawDocDto
import uz.deklarant.ai.data.remote.MatchDto
import uz.deklarant.ai.data.remote.MessageDto
import uz.deklarant.ai.data.remote.NodeDto
import uz.deklarant.ai.data.remote.RoleDto
import uz.deklarant.ai.data.remote.UserDto
import uz.deklarant.ai.domain.model.BrowseLevel
import uz.deklarant.ai.domain.model.BrowseNode
import uz.deklarant.ai.domain.model.BrowsePage
import uz.deklarant.ai.domain.model.ChatMessage
import uz.deklarant.ai.domain.model.ChatMode
import uz.deklarant.ai.domain.model.Crumb
import uz.deklarant.ai.domain.model.LawArticle
import uz.deklarant.ai.domain.model.LawDoc
import uz.deklarant.ai.domain.model.LawText
import uz.deklarant.ai.domain.model.Role
import uz.deklarant.ai.domain.model.RoleInfo
import uz.deklarant.ai.domain.model.User

// DTO ↔ domen o'girish. Bir joyda (DRY) va bir yo'nalishli:
// tarmoq shakli domenga kiradi, teskarisi faqat so'rov yasashda.

internal fun ChatMessage.toDto() = MessageDto(
    role = when (role) {
        ChatMessage.Role.User -> "user"
        ChatMessage.Role.Assistant -> "assistant"
    },
    content = content,
    images = images.takeIf { it.isNotEmpty() }?.map { ImageDto(it.mediaType, it.data) },
)


internal fun NodeDto.toDomain() = BrowseNode(
    id = id,
    title = title,
    count = count,
    isLeaf = leaf,
    importDuty = importDuty,
    vat = vat,
    unit = unit,
)

internal fun MatchDto.toDomain() = BrowseNode(
    id = code.code,
    // Qidiruv natijasida o'zbekcha nom afzal; bo'lmasa ruschasi.
    title = code.nameUz.ifBlank { code.nameRu },
    isLeaf = true,
    importDuty = code.importDuty,
    vat = code.vat,
    unit = code.unit,
    specific = code.specific,
    specificUnit = code.specificUnit,
)

internal fun CrumbDto.toDomain() = Crumb(
    level = level.toBrowseLevel(),
    id = id,
    title = title,
)

internal fun BrowseDto.toDomain() = BrowsePage(
    level = level.toBrowseLevel(),
    parent = parent,
    items = items.map { it.toDomain() },
    path = path.map { it.toDomain() },
)

/**
 * Backend darajasini sanab o'tilgan turga o'giradi.
 *
 * Noma'lum qiymat kelsa — bo'limlarga qaytamiz. Bu jim degradatsiya,
 * lekin bu yerda to'g'ri: yangi backend noma'lum daraja qo'shsa, eski
 * ilova ishlashdan to'xtamasligi kerak.
 */
private fun String.toBrowseLevel(): BrowseLevel = when (this) {
    "sections", "section" -> BrowseLevel.Sections
    "groups", "group" -> BrowseLevel.Groups
    "headings", "heading" -> BrowseLevel.Headings
    "codes", "code" -> BrowseLevel.Codes
    else -> BrowseLevel.Sections
}

internal fun LawDocDto.toDomain() = LawDoc(
    id = id,
    name = name,
    date = date,
    lex = lex,
    chunks = chunks,
)

internal fun LawArticleDto.toDomain() = LawArticle(
    doc = doc,
    index = index,
    title = title,
    preview = preview,
)

internal fun LawChunkDto.toDomain() = LawText(
    docName = name,
    date = date,
    lex = lex,
    title = title,
    text = text,
)

// ---------------------------------------------------------------- auth

internal fun UserDto.toDomain() = User(
    id = id,
    login = login,
    name = name,
    role = Role.of(role),
    dailyQuota = dailyQuota,
    chatMode = chatMode.toChatMode(),
)

internal fun RoleDto.toDomain() = RoleInfo(
    role = Role.of(role),
    chatMode = chatMode.toChatMode(),
    dailyQuota = dailyQuota,
    selfSignup = selfSignup,
)

/** Noma'lum uslub kelsa deklarantga tushamiz — ilova to'xtamasin. */
private fun String.toChatMode(): ChatMode =
    ChatMode.entries.firstOrNull { it.wire == this } ?: ChatMode.Declarant
