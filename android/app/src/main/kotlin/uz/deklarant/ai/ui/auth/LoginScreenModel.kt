package uz.deklarant.ai.ui.auth

import cafe.adriel.voyager.core.model.ScreenModel
import cafe.adriel.voyager.core.model.screenModelScope
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import uz.deklarant.ai.domain.model.Role
import uz.deklarant.ai.domain.model.RoleInfo
import uz.deklarant.ai.domain.repository.AuthRepository

/** Ekranning ikki holati: kirish yoki ro'yxatdan o'tish. */
enum class AuthTab { Login, Register }

data class LoginState(
    val tab: AuthTab = AuthTab.Login,
    val login: String = "",
    val password: String = "",
    val name: String = "",
    val role: Role = Role.Declarant,
    val roles: List<RoleInfo> = emptyList(),
    val busy: Boolean = false,
    val error: String? = null,
) {
    /** Parol chegarasi backenddagi bilan bir xil (8 belgi). */
    val canSubmit: Boolean get() = !busy && login.isNotBlank() && password.length >= MIN_PASSWORD

    companion object {
        const val MIN_PASSWORD = 8
    }
}

class LoginScreenModel(private val auth: AuthRepository) : ScreenModel {

    private val _state = MutableStateFlow(LoginState())
    val state: StateFlow<LoginState> = _state.asStateFlow()

    init {
        // Rollar SERVERDAN: kvota va uslub u yerda belgilanadi. Ikki
        // joyda yozilsa, vaqt o'tib ajralib ketardi.
        screenModelScope.launch {
            runCatching { auth.roles() }
                .onSuccess { list -> _state.update { it.copy(roles = list.filter(RoleInfo::selfSignup)) } }
        }
    }

    fun setTab(tab: AuthTab) = _state.update { it.copy(tab = tab, error = null) }
    fun setLogin(v: String) = _state.update { it.copy(login = v) }
    fun setPassword(v: String) = _state.update { it.copy(password = v) }
    fun setName(v: String) = _state.update { it.copy(name = v) }
    fun setRole(v: Role) = _state.update { it.copy(role = v) }
    fun dismissError() = _state.update { it.copy(error = null) }

    /**
     * Yuborish.
     *
     * Muvaffaqiyat holatida hech narsa qaytarilmaydi: kirgan
     * foydalanuvchi `AuthRepository.user` oqimida paydo bo'ladi va
     * ekran O'ZI almashadi. Aks holda ikkita haqiqat manbai bo'lardi.
     */
    fun submit() {
        val s = _state.value
        if (!s.canSubmit) return
        _state.update { it.copy(busy = true, error = null) }

        screenModelScope.launch {
            val result = runCatching {
                when (s.tab) {
                    AuthTab.Login -> auth.login(s.login, s.password)
                    AuthTab.Register -> auth.register(s.login, s.password, s.name, s.role)
                }
            }
            _state.update {
                it.copy(busy = false, error = result.exceptionOrNull()?.message)
            }
        }
    }
}
