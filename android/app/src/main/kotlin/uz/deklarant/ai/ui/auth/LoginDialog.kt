package uz.deklarant.ai.ui.auth

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.imePadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.selection.selectable
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Close
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.RadioButton
import androidx.compose.material3.SegmentedButton
import androidx.compose.material3.SegmentedButtonDefaults
import androidx.compose.material3.SingleChoiceSegmentedButtonRow
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.semantics.Role as SemanticsRole
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.input.PasswordVisualTransformation
import androidx.compose.ui.unit.dp
import androidx.compose.ui.window.Dialog
import uz.deklarant.ai.domain.model.Role
import uz.deklarant.ai.domain.model.RoleInfo
import uz.deklarant.ai.ui.common.LocalContainer

/**
 * Kirish va ro'yxatdan o'tish — MODAL oyna.
 *
 * NEGA EKRAN EMAS: kirish alohida ekran bo'lganda chat butunlay
 * yo'qolardi va foydalanuvchi "ilova nima qila oladi" degan savolga
 * javob ololmasdi. Modal suhbat oynasini joyida qoldiradi — kirish
 * shunchaki oldida turadi va yopilsa hech narsa yo'qolmaydi.
 *
 * Chat PUL TURADI, shuning uchun kirish talab qilinadi. TIF TN va
 * Qonunlar AI ga bormaydi va kirishsiz ishlayveradi.
 */
// Model TASHQARIDAN uzatiladi: `rememberScreenModel` — Voyager'ning
// `Screen` kengaytmasi va oddiy composable ichida chaqirib bo'lmaydi.
// Uni chaqiruvchi ekran yasaydi, shunda model hayotiy sikli ham
// Voyager qo'lida qoladi.
@Composable
fun LoginDialog(model: LoginScreenModel, onDismiss: () -> Unit) {
    val container = LocalContainer.current
    val state by model.state.collectAsState()
    val user by container.authRepository.user.collectAsState()

    // Kirish muvaffaqiyatli bo'lsa oyna O'ZI yopiladi: natijani
    // kuzatuvchi oqim beradi, ya'ni ikkita haqiqat manbai bo'lmaydi.
    LaunchedEffect(user) {
        if (user != null) onDismiss()
    }

    Dialog(onDismissRequest = onDismiss) {
        Surface(shape = RoundedCornerShape(20.dp), tonalElevation = 6.dp) {
            Column(
                Modifier
                    .imePadding()
                    .verticalScroll(rememberScrollState())
                    .padding(horizontal = 18.dp, vertical = 16.dp),
                verticalArrangement = Arrangement.spacedBy(12.dp),
            ) {
                Row(verticalAlignment = Alignment.CenterVertically) {
                    Text(
                        text = if (state.tab == AuthTab.Login) "Kirish" else "Ro'yxatdan o'tish",
                        style = MaterialTheme.typography.titleLarge,
                        modifier = Modifier.weight(1f),
                    )
                    IconButton(onClick = onDismiss) {
                        Icon(Icons.Default.Close, contentDescription = "Yopish")
                    }
                }

                Text(
                    text = "Chat uchun kirish kerak. TIF TN va Qonunlar " +
                        "bo'limlari kirishsiz ham ishlaydi.",
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )

                SingleChoiceSegmentedButtonRow(Modifier.fillMaxWidth()) {
                    AuthTab.entries.forEachIndexed { i, tab ->
                        SegmentedButton(
                            selected = state.tab == tab,
                            onClick = { model.setTab(tab) },
                            shape = SegmentedButtonDefaults.itemShape(i, AuthTab.entries.size),
                        ) {
                            Text(if (tab == AuthTab.Login) "Kirish" else "Ro'yxatdan o'tish")
                        }
                    }
                }

                OutlinedTextField(
                    value = state.login,
                    onValueChange = model::setLogin,
                    label = { Text("Telefon yoki e-pochta") },
                    supportingText = { Text("Bo'shliq va chiziqchalar hisobga olinmaydi") },
                    singleLine = true,
                    keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Phone),
                    modifier = Modifier.fillMaxWidth(),
                )

                OutlinedTextField(
                    value = state.password,
                    onValueChange = model::setPassword,
                    label = { Text("Parol") },
                    supportingText = {
                        if (state.tab == AuthTab.Register) {
                            Text("Kamida ${LoginState.MIN_PASSWORD} belgi")
                        }
                    },
                    singleLine = true,
                    visualTransformation = PasswordVisualTransformation(),
                    keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Password),
                    modifier = Modifier.fillMaxWidth(),
                )

                if (state.tab == AuthTab.Register) {
                    OutlinedTextField(
                        value = state.name,
                        onValueChange = model::setName,
                        label = { Text("Ism") },
                        singleLine = true,
                        modifier = Modifier.fillMaxWidth(),
                    )

                    Text("Rol", style = MaterialTheme.typography.labelLarge)
                    state.roles.forEach { info ->
                        RoleOption(
                            info = info,
                            selected = state.role == info.role,
                            onSelect = { model.setRole(info.role) },
                        )
                    }
                    Text(
                        // ADMIN ro'yxatdan o'tishda berilmaydi — buni aytib
                        // qo'yish kerak, aks holda "rol yo'q" deb o'ylanardi.
                        text = "ADMIN roli bu yerda berilmaydi — uni mavjud admin tayinlaydi.",
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                    )
                }

                state.error?.let { message ->
                    Text(
                        text = message,
                        color = MaterialTheme.colorScheme.error,
                        style = MaterialTheme.typography.bodyMedium,
                    )
                }

                Button(
                    onClick = model::submit,
                    enabled = state.canSubmit,
                    modifier = Modifier.fillMaxWidth(),
                ) {
                    if (state.busy) {
                        CircularProgressIndicator(Modifier.padding(2.dp), strokeWidth = 2.dp)
                    } else {
                        Text(if (state.tab == AuthTab.Login) "Kirish" else "Ro'yxatdan o'tish")
                    }
                }
            }
        }
    }
}

/**
 * Rol tanlash bandi.
 *
 * Kvota va uslub KO'RSATILADI: rol nomining o'zi "nima o'zgaradi"
 * degan savolga javob bermaydi.
 */
@Composable
private fun RoleOption(info: RoleInfo, selected: Boolean, onSelect: () -> Unit) {
    val border = if (selected) {
        MaterialTheme.colorScheme.primary
    } else {
        MaterialTheme.colorScheme.outlineVariant
    }
    Row(
        Modifier
            .fillMaxWidth()
            .border(1.dp, border, RoundedCornerShape(12.dp))
            .background(
                if (selected) MaterialTheme.colorScheme.primaryContainer.copy(alpha = 0.35f)
                else MaterialTheme.colorScheme.surface,
                RoundedCornerShape(12.dp),
            )
            .selectable(selected = selected, role = SemanticsRole.RadioButton, onClick = onSelect)
            .padding(horizontal = 10.dp, vertical = 10.dp),
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(6.dp),
    ) {
        RadioButton(selected = selected, onClick = null)
        Column(Modifier.fillMaxWidth()) {
            Text(info.role.label, style = MaterialTheme.typography.bodyLarge)
            Text(
                text = describe(info.role),
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
            Text(
                text = "javob: ${info.chatMode.wire} · kuniga ${info.dailyQuota} savol",
                style = MaterialTheme.typography.labelSmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
        }
    }
}

private fun describe(role: Role): String = when (role) {
    Role.Declarant -> "TIF TN va GTD grafalarini bilaman — qisqa, kod va modda raqami bilan"
    Role.Business -> "Atamalarni bilmayman — \"qancha turadi va nima qilishim kerak\""
    Role.Inspector -> "Bojxona xodimi — deklarant bilan bir xil javob, kattaroq chegara"
    Role.Admin -> "Statistika va sozlamalar paneli"
}
