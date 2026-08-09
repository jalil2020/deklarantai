package uz.deklarant.ai.ui.browse

import androidx.compose.foundation.clickable
import androidx.compose.foundation.horizontalScroll
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.rememberScrollState
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Close
import androidx.compose.material.icons.filled.Search
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import cafe.adriel.voyager.core.model.rememberScreenModel
import cafe.adriel.voyager.core.screen.Screen
import uz.deklarant.ai.domain.model.BrowseLevel
import uz.deklarant.ai.domain.model.BrowseNode
import uz.deklarant.ai.domain.model.Crumb
import uz.deklarant.ai.ui.common.EmptyBox
import uz.deklarant.ai.ui.common.ErrorBox
import uz.deklarant.ai.ui.common.LoadingBox
import uz.deklarant.ai.ui.common.LocalChatHandoff
import uz.deklarant.ai.ui.common.LocalContainer

/**
 * TIF TN kodini topish — IKKI YO'L bitta ekranda.
 *
 *	Qidiruv  — kod raqami («8703 23») yoki tovar nomi («noutbuk»)
 *	Ierarxiya — Bo'lim (21) → Guruh (96) → Pozitsiya → Kod
 *
 * NEGA IKKALASI: qidiruv tovarni NOMENKLATURA TILIDA atashni talab
 * qiladi — "musor tashuvchi mashina" u yerda "maxsus avtotransport".
 * Ierarxiya esa hech qanday atama bilishni talab qilmaydi. Deklarant
 * kodni ba'zan biladi, ba'zan bilmaydi.
 *
 * Qidiruv maydoni bo'shatilsa ierarxiya qaytadi.
 */
class BrowseScreen : Screen {

    override val key: String = "browse"

    @Composable
    override fun Content() {
        val container = LocalContainer.current
        val handoff = LocalChatHandoff.current
        val model = rememberScreenModel { BrowseScreenModel(container.codeRepository) }
        val state by model.state.collectAsState()

        Column(Modifier.fillMaxSize()) {
            SearchField(
                query = state.query,
                isSearching = state.isSearching,
                onQuery = model::onQuery,
                onClear = model::clearQuery,
            )

            // Qidiruv paytida yo'l zanjiri KERAK EMAS: natijalar
            // ierarxiyaning bir joyiga tegishli emas, butun baza bo'ylab.
            if (!state.inSearch) {
                Crumbs(
                    path = state.page?.path.orEmpty(),
                    onRoot = model::loadSections,
                    onCrumb = model::openCrumb,
                )
            }
            HorizontalDivider()

            when {
                // ---- qidiruv natijasi ----
                state.inSearch -> {
                    val found = state.results.orEmpty()
                    when {
                        state.error != null -> ErrorBox(state.error!!, onRetry = { model.onQuery(state.query) })
                        found.isEmpty() && state.isSearching -> LoadingBox()
                        found.isEmpty() -> EmptyBox("Topilmadi — boshqacha atashga urinib ko'ring")
                        else -> LazyColumn(Modifier.fillMaxSize()) {
                            items(found) { node ->
                                // Qidiruv natijasi HAR DOIM barg (kod).
                                NodeRow(node = node, isLeaf = true, onClick = { handoff.offer(questionFor(node)) })
                                HorizontalDivider()
                            }
                        }
                    }
                }

                // ---- ierarxiya ----
                state.isLoading && state.page == null -> LoadingBox()
                state.error != null -> ErrorBox(state.error!!, onRetry = model::loadSections)
                state.page == null -> LoadingBox()
                state.page!!.items.isEmpty() -> EmptyBox("Bo'sh")
                else -> {
                    val page = state.page!!
                    LazyColumn(Modifier.fillMaxSize()) {
                        items(page.items) { node ->
                            NodeRow(
                                node = node,
                                isLeaf = page.level == BrowseLevel.Codes,
                                onClick = {
                                    if (page.level == BrowseLevel.Codes) handoff.offer(questionFor(node))
                                    else model.open(node)
                                },
                            )
                            HorizontalDivider()
                        }
                    }
                }
            }
        }
    }
}

/**
 * Qidiruv maydoni.
 *
 * Kod raqami ham, tovar nomi ham yoziladi: «8703 23» yoki «noutbuk».
 * Bo'sh qolsa ierarxiya qaytadi — ikkalasi bitta ekranda, chunki
 * deklarant kodni ba'zan biladi, ba'zan bilmaydi.
 */
@Composable
private fun SearchField(
    query: String,
    isSearching: Boolean,
    onQuery: (String) -> Unit,
    onClear: () -> Unit,
) {
    OutlinedTextField(
        value = query,
        onValueChange = onQuery,
        modifier = Modifier.fillMaxWidth().padding(horizontal = 10.dp, vertical = 6.dp),
        placeholder = { Text("Kod yoki tovar nomi — «8703» yoki «noutbuk»") },
        leadingIcon = { Icon(Icons.Default.Search, contentDescription = null) },
        trailingIcon = {
            when {
                isSearching -> CircularProgressIndicator(Modifier.size(18.dp), strokeWidth = 2.dp)
                query.isNotEmpty() -> IconButton(onClick = onClear) {
                    Icon(Icons.Default.Close, contentDescription = "Tozalash")
                }
            }
        },
        singleLine = true,
    )
}

@Composable
private fun Crumbs(
    path: List<Crumb>,
    onRoot: () -> Unit,
    onCrumb: (Crumb) -> Unit,
) {
    Row(
        Modifier
            .fillMaxWidth()
            .horizontalScroll(rememberScrollState())
            .padding(horizontal = 4.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        TextButton(onClick = onRoot) { Text("Bo'limlar") }
        path.forEach { crumb ->
            Text("›", color = MaterialTheme.colorScheme.onSurfaceVariant)
            TextButton(onClick = { onCrumb(crumb) }) { Text(crumb.id) }
        }
    }
}

@Composable
private fun NodeRow(node: BrowseNode, isLeaf: Boolean, onClick: () -> Unit) {
    Row(
        Modifier
            .fillMaxWidth()
            .clickable(onClick = onClick)
            .padding(horizontal = 14.dp, vertical = 12.dp),
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(12.dp),
    ) {
        Text(
            text = if (isLeaf) formatCode(node.id) else node.id,
            style = MaterialTheme.typography.labelLarge,
            fontFamily = FontFamily.Monospace,
            color = MaterialTheme.colorScheme.primary,
        )
        Text(
            text = node.title.ifBlank { "—" },
            style = MaterialTheme.typography.bodyMedium,
            maxLines = 2,
            modifier = Modifier.weight(1f),
        )
        Column(horizontalAlignment = Alignment.End) {
            Text(
                text = if (isLeaf) "${trimZero(node.importDuty)}%" else node.count.toString(),
                style = MaterialTheme.typography.labelMedium,
                fontWeight = if (isLeaf) FontWeight.SemiBold else FontWeight.Normal,
                color = if (isLeaf) MaterialTheme.colorScheme.primary
                else MaterialTheme.colorScheme.onSurfaceVariant,
            )
            // Kombinatsiyalangan stavka — 1 555 kodda. Ko'rsatilmasa,
            // faqat foizni ko'rgan deklarant bojni kam hisoblardi.
            node.specific?.let { s ->
                Text(
                    text = "$${trimZero(s)}/${node.specificUnit ?: "kg"}",
                    style = MaterialTheme.typography.labelSmall,
                    color = MaterialTheme.colorScheme.tertiary,
                )
            }
        }
    }
}

/**
 * TIF TN kodi rasmiy ravishda 4-2-3-1 guruhlab yoziladi: 8418 10 200 1.
 * Bir bo'lak 10 raqam o'qishga qiyin va nusxa olishda xatoga olib keladi.
 */
private fun formatCode(code: String): String =
    if (code.length != 10) code
    else "${code.take(4)} ${code.substring(4, 6)} ${code.substring(6, 9)} ${code.last()}"

/** 20.0 → "20", 12.5 → "12.5" — ortiqcha nol ko'rsatilmaydi. */
private fun trimZero(v: Double): String =
    if (v % 1.0 == 0.0) v.toInt().toString() else v.toString()

/**
 * Kod uchun tayyor savol.
 *
 * Savol TUGALLANMAGAN qoldiriladi ("Qiymati: ") — foydalanuvchi qiymat
 * va davlatni qo'shishi kerak. Avtomatik yuborilsa, u yarim savolga
 * javob olardi.
 */
private fun questionFor(node: BrowseNode): String =
    "${node.id} (${node.title}) bo'yicha bojni hisobla. Qiymati: "
