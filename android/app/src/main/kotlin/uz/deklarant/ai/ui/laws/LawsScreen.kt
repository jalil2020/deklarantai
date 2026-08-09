package uz.deklarant.ai.ui.laws

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalUriHandler
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import cafe.adriel.voyager.core.model.rememberScreenModel
import cafe.adriel.voyager.core.screen.Screen
import uz.deklarant.ai.domain.model.LawArticle
import uz.deklarant.ai.domain.model.LawDoc
import uz.deklarant.ai.domain.model.LawText
import uz.deklarant.ai.ui.common.EmptyBox
import uz.deklarant.ai.ui.common.ErrorBox
import uz.deklarant.ai.ui.common.LoadingBox
import uz.deklarant.ai.ui.common.LocalChatHandoff
import uz.deklarant.ai.ui.common.LocalContainer

/**
 * Qonun korpusi bo'ylab ko'rish.
 *
 * NEGA KERAK: qidiruv "nima izlayotganingizni bilasiz" deb faraz qiladi.
 * Deklarant ko'pincha aksincha ish tutadi — kodeksni ochib, moddalarni
 * ko'zdan kechiradi.
 *
 * Hujjat (89) → Modda (1405) → To'liq matn
 */
class LawsScreen : Screen {

    override val key: String = "laws"

    @Composable
    override fun Content() {
        val container = LocalContainer.current
        val handoff = LocalChatHandoff.current
        val model = rememberScreenModel { LawsScreenModel(container.lawRepository) }
        val state by model.state.collectAsState()

        Column(Modifier.fillMaxSize()) {
            Crumbs(state.view, model)
            HorizontalDivider()

            when {
                state.error != null -> ErrorBox(state.error!!, onRetry = model::loadDocs)
                state.isLoading && state.view == null -> LoadingBox()
                else -> when (val view = state.view) {
                    null -> LoadingBox()
                    is LawsView.Docs -> DocList(view.items, model::openDoc)
                    is LawsView.Articles -> ArticleList(view.items, model::openArticle)
                    is LawsView.Article -> ArticleText(view.text) { handoff.offer(it) }
                }
            }
        }
    }
}

@Composable
private fun Crumbs(view: LawsView?, model: LawsScreenModel) {
    Row(Modifier.fillMaxWidth().padding(horizontal = 4.dp)) {
        TextButton(onClick = model::loadDocs) { Text("Hujjatlar") }
        when (view) {
            is LawsView.Articles -> {
                Text("›", color = MaterialTheme.colorScheme.onSurfaceVariant)
                Text(
                    text = "${view.items.size} modda",
                    style = MaterialTheme.typography.labelLarge,
                    modifier = Modifier.padding(12.dp),
                )
            }
            is LawsView.Article -> {
                Text("›", color = MaterialTheme.colorScheme.onSurfaceVariant)
                TextButton(onClick = model::backToArticles) { Text("Moddalar") }
            }
            else -> Unit
        }
    }
}

@Composable
private fun DocList(docs: List<LawDoc>, onOpen: (LawDoc) -> Unit) {
    if (docs.isEmpty()) return EmptyBox("Hujjat yo'q")
    LazyColumn(Modifier.fillMaxSize()) {
        items(docs) { doc ->
            Column(
                Modifier
                    .fillMaxWidth()
                    .clickable { onOpen(doc) }
                    .padding(horizontal = 14.dp, vertical = 10.dp),
            ) {
                Text(doc.name, style = MaterialTheme.typography.bodyMedium, fontWeight = FontWeight.SemiBold, maxLines = 3)
                Text(
                    text = buildString {
                        append("${doc.chunks} parcha")
                        doc.date?.let { append(" · $it") }
                        if (doc.lex != null) append(" · lex.uz")
                    },
                    style = MaterialTheme.typography.labelSmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
            }
            HorizontalDivider()
        }
    }
}

@Composable
private fun ArticleList(articles: List<LawArticle>, onOpen: (LawArticle) -> Unit) {
    if (articles.isEmpty()) return EmptyBox("Modda yo'q")
    LazyColumn(Modifier.fillMaxSize()) {
        items(articles) { a ->
            Column(
                Modifier
                    .fillMaxWidth()
                    .clickable { onOpen(a) }
                    .padding(horizontal = 14.dp, vertical = 10.dp),
            ) {
                Text(
                    text = a.title.ifBlank { "(sarlavhasiz)" },
                    style = MaterialTheme.typography.bodyMedium,
                    fontWeight = FontWeight.SemiBold,
                    maxLines = 3,
                )
                Text(
                    text = a.preview,
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                    maxLines = 2,
                )
            }
            HorizontalDivider()
        }
    }
}

@Composable
private fun ArticleText(text: LawText, onAsk: (String) -> Unit) {
    val uriHandler = LocalUriHandler.current
    Column(
        Modifier
            .fillMaxSize()
            .verticalScroll(rememberScrollState())
            .padding(16.dp),
        verticalArrangement = Arrangement.spacedBy(10.dp),
    ) {
        Text(text.title, style = MaterialTheme.typography.titleMedium)
        Text(
            text = buildString {
                append(text.docName)
                text.date?.let { append(" · $it") }
            },
            style = MaterialTheme.typography.labelSmall,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
        )
        Text(text.text, style = MaterialTheme.typography.bodyMedium)

        Row(horizontalArrangement = Arrangement.spacedBy(10.dp)) {
            OutlinedButton(onClick = { onAsk("\"${text.title}\" moddasini oddiy tilda tushuntir.") }) {
                Text("Chatda tushuntir")
            }
            // Rasmiy manba — ilova matni bilan qonun matni orasida
            // farq bo'lsa, foydalanuvchi asl nusxani ko'ra olsin.
            text.lex?.let { link ->
                TextButton(onClick = { uriHandler.openUri(link) }) { Text("lex.uz") }
            }
        }
    }
}
