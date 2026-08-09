package uz.deklarant.ai.domain.model

/**
 * Foydalanuvchi roli.
 *
 * Rol UCH narsani belgilaydi: chat javobining sukut uslubi, kunlik
 * so'rov kvotasi va admin panelga kirish. Ro'yxat ataylab qisqa —
 * ishlamaydigan "ruxsatlar" xavfsizlik tuyg'usini beradi-yu, hech
 * narsani himoya qilmaydi.
 */
enum class Role(val wire: String, val label: String) {
    Declarant("DECLARANT", "Deklarant"),
    Business("BUSINESS", "Tadbirkor"),
    Inspector("INSPECTOR", "Inspektor"),
    Admin("ADMIN", "Administrator");

    companion object {
        /** Noma'lum rol kelsa deklarantga tushamiz — ilova ishlashdan to'xtamasin. */
        fun of(wire: String): Role = entries.firstOrNull { it.wire == wire } ?: Declarant
    }
}

data class User(
    val id: String,
    val login: String,
    val name: String,
    val role: Role,
    val dailyQuota: Int,
    val chatMode: ChatMode,
)

/**
 * Ro'yxatdan o'tishda ko'rsatiladigan rol ma'lumoti.
 *
 * Kvota va uslub SERVERDAN keladi: ular ikki joyda yozilsa, vaqt o'tib
 * ajralib ketardi va foydalanuvchi noto'g'ri raqamni ko'rardi.
 */
data class RoleInfo(
    val role: Role,
    val chatMode: ChatMode,
    val dailyQuota: Int,
    /** ADMIN ni ro'yxatdan o'tishda tanlab bo'lmaydi. */
    val selfSignup: Boolean,
)
