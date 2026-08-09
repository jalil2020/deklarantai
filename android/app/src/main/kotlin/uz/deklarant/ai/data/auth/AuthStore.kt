package uz.deklarant.ai.data.auth

import android.content.Context
import androidx.core.content.edit

/**
 * Kirish tokenini saqlaydi.
 *
 * NEGA DISKKA: token 30 kun amal qiladi. Faqat xotirada saqlansa,
 * foydalanuvchi ilovani har ochganda qayta kirishga majbur bo'lardi —
 * bu hech qanday xavfsizlik bermaydi, faqat noqulaylik.
 *
 * ⚠️ SharedPreferences shifrlanmagan. Root qilingan qurilmada uni o'qish
 * mumkin. Bu qabul qilingan murosa: token muddatli va serverdan bekor
 * qilinadi (chiqish token versiyasini oshiradi), parolning o'zi esa bu
 * yerda umuman saqlanmaydi.
 */
internal class AuthStore(context: Context) {

    private val prefs = context.getSharedPreferences("deklarant.auth", Context.MODE_PRIVATE)

    var token: String
        get() = prefs.getString(KEY_TOKEN, "").orEmpty()
        set(value) = prefs.edit {
            if (value.isEmpty()) remove(KEY_TOKEN) else putString(KEY_TOKEN, value)
        }

    fun clear() {
        token = ""
    }

    private companion object {
        const val KEY_TOKEN = "token"
    }
}
