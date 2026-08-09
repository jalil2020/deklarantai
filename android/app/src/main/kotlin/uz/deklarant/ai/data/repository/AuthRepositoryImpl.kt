package uz.deklarant.ai.data.repository

import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import uz.deklarant.ai.data.auth.AuthStore
import uz.deklarant.ai.data.remote.AuthRequest
import uz.deklarant.ai.data.remote.DeklarantApi
import uz.deklarant.ai.data.remote.SessionDtoFull
import uz.deklarant.ai.domain.model.Role
import uz.deklarant.ai.domain.model.RoleInfo
import uz.deklarant.ai.domain.model.User
import uz.deklarant.ai.domain.repository.AuthRepository

internal class AuthRepositoryImpl(
    private val api: DeklarantApi,
    private val store: AuthStore,
) : AuthRepository {

    private val _user = MutableStateFlow<User?>(null)
    override val user: StateFlow<User?> = _user.asStateFlow()

    /**
     * Saqlangan tokenni tekshiradi.
     *
     * Token muddati tugagan yoki chiqish qilingan bo'lsa server 401
     * qaytaradi — o'shanda tokenni O'CHIRAMIZ. Aks holda u har so'rovda
     * yuborilib, foydalanuvchi nima uchun ishlamayotganini tushunmasdi.
     */
    override suspend fun restore() {
        val token = store.token
        if (token.isEmpty()) return
        runCatching { api.me(token) }
            .onSuccess { _user.value = it.toDomain() }
            .onFailure {
                store.clear()
                _user.value = null
            }
    }

    override suspend fun login(login: String, password: String): User =
        apply(api.login(AuthRequest(login = login, password = password)))

    override suspend fun register(
        login: String,
        password: String,
        name: String,
        role: Role,
    ): User = apply(
        api.register(
            AuthRequest(
                login = login,
                password = password,
                name = name.ifBlank { null },
                role = role.wire,
            ),
        ),
    )

    /**
     * Chiqish.
     *
     * Serverga ham xabar beriladi: token IMZOLANGAN va serverda
     * saqlanmaydi, ya'ni faqat qurilmadan o'chirish uni bekor
     * qilmaydi — o'g'irlangan nusxa muddati tugagunicha ishlayverardi.
     * Server esa token versiyasini oshiradi va hammasi darrov o'ladi.
     *
     * Tarmoq ishlamasa ham mahalliy tokenni o'chiramiz: foydalanuvchi
     * "chiqdim" deb o'ylab, aslida kirgan holda qolmasin.
     */
    override suspend fun logout() {
        val token = store.token
        if (token.isNotEmpty()) {
            runCatching { api.logout(token) }
        }
        store.clear()
        _user.value = null
    }

    override suspend fun roles(): List<RoleInfo> =
        api.roles().roles.map { it.toDomain() }

    private fun apply(session: SessionDtoFull): User {
        store.token = session.token
        val u = session.user.toDomain()
        _user.value = u
        return u
    }
}
