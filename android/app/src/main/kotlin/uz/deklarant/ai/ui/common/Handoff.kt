package uz.deklarant.ai.ui.common

import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.compose.runtime.staticCompositionLocalOf

/**
 * Bo'limlar orasida savol uzatish.
 *
 * NEGA KERAK: TIF TN yoki Qonunlar bo'limida element tanlanganda,
 * chatga tayyor savol qo'yilishi va chat bo'limi ochilishi kerak.
 * Voyager tab'lari bir-birini ko'rmaydi, shuning uchun uzatish
 * YUQORIDA turadi va ikkala tomon shu obyektni ko'radi.
 *
 * NEGA `consume()`: savol BIR MARTA qo'yilishi kerak. Oddiy o'zgaruvchi
 * bo'lsa, chat ekrani har qayta chizilganda uni yana maydonga
 * yozib, foydalanuvchi terganini o'chirib yuborardi.
 */
class ChatHandoff {
    private var pending by mutableStateOf<String?>(null)

    fun offer(question: String) {
        pending = question
    }

    /** Kutayotgan savolni oladi va tozalaydi. Yo'q bo'lsa — null. */
    fun consume(): String? {
        val value = pending
        pending = null
        return value
    }

    /** Savol kutyaptimi — chat bo'limiga o'tish kerakligini bildiradi. */
    val hasPending: Boolean get() = pending != null
}

val LocalChatHandoff = staticCompositionLocalOf<ChatHandoff> {
    error("ChatHandoff berilmagan")
}

/**
 * Kirish oynasini SO'RASH.
 *
 * NEGA KERAK: kirish tugmasi endi yon menyuda (drawer), oyna esa chat
 * ekranida ochiladi. Ularni to'g'ridan-to'g'ri bog'lab bo'lmaydi —
 * `LoginDialog` ga `LoginScreenModel` kerak, u esa `rememberScreenModel`
 * bilan olinadi va bu Voyager `Screen` ning kengaytmasi. Menyu oddiy
 * composable ichida, ya'ni u yerda ScreenModel yasab bo'lmaydi
 * (yasalsa ham umri boshqarilmay, oqib ketardi).
 *
 * Shuning uchun menyu faqat SO'RAYDI, oynani chat ekrani ochadi —
 * [ChatHandoff] bilan bir xil naqsh.
 */
class LoginPrompt {
    var requested by mutableStateOf(false)
        private set

    fun request() {
        requested = true
    }

    fun clear() {
        requested = false
    }
}

val LocalLoginPrompt = staticCompositionLocalOf<LoginPrompt> {
    error("LoginPrompt berilmagan")
}
