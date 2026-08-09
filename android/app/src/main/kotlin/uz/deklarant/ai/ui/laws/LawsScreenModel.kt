package uz.deklarant.ai.ui.laws

import cafe.adriel.voyager.core.model.StateScreenModel
import cafe.adriel.voyager.core.model.screenModelScope
import kotlinx.coroutines.launch
import uz.deklarant.ai.domain.model.LawArticle
import uz.deklarant.ai.domain.model.LawDoc
import uz.deklarant.ai.domain.model.LawText
import uz.deklarant.ai.domain.repository.LawRepository

/**
 * Qonun bo'limining ko'rinishi.
 *
 * Uch daraja bitta MUHRLANGAN turda: shu tufayli ekran `when` bilan
 * to'liq qamrab oladi va yangi daraja qo'shilsa kompilyator unutilgan
 * shoxni ko'rsatadi.
 */
sealed interface LawsView {
    data class Docs(val items: List<LawDoc>) : LawsView
    data class Articles(val doc: LawDoc, val items: List<LawArticle>) : LawsView
    data class Article(val doc: LawDoc, val text: LawText) : LawsView
}

data class LawsState(
    val view: LawsView? = null,
    val isLoading: Boolean = false,
    val error: String? = null,
)

class LawsScreenModel(
    private val repository: LawRepository,
) : StateScreenModel<LawsState>(LawsState()) {

    // Ochilgan hujjatni eslab qolamiz: moddadan ro'yxatga qaytishda uni
    // qayta so'ramaslik uchun.
    private var currentDoc: LawDoc? = null

    init {
        loadDocs()
    }

    fun loadDocs() = load {
        currentDoc = null
        LawsView.Docs(repository.docs())
    }

    fun openDoc(doc: LawDoc) = load {
        currentDoc = doc
        LawsView.Articles(doc, repository.articles(doc.id))
    }

    fun openArticle(article: LawArticle) = load {
        val doc = currentDoc ?: error("hujjat tanlanmagan")
        LawsView.Article(doc, repository.article(article.doc, article.index))
    }

    fun backToArticles() {
        val doc = currentDoc ?: return loadDocs()
        openDoc(doc)
    }

    private fun load(block: suspend () -> LawsView) {
        mutableState.value = mutableState.value.copy(isLoading = true, error = null)
        screenModelScope.launch {
            runCatching { block() }
                .onSuccess { view -> mutableState.value = LawsState(view = view, isLoading = false) }
                .onFailure { e ->
                    mutableState.value = mutableState.value.copy(
                        isLoading = false,
                        error = e.message ?: "Yuklab bo'lmadi",
                    )
                }
        }
    }
}
