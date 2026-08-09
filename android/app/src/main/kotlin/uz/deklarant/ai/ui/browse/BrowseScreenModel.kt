package uz.deklarant.ai.ui.browse

import cafe.adriel.voyager.core.model.StateScreenModel
import cafe.adriel.voyager.core.model.screenModelScope
import kotlinx.coroutines.Job
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch
import uz.deklarant.ai.domain.model.BrowseLevel
import uz.deklarant.ai.domain.model.BrowseNode
import uz.deklarant.ai.domain.model.BrowsePage
import uz.deklarant.ai.domain.model.Crumb
import uz.deklarant.ai.domain.repository.CodeRepository

data class BrowseState(
    val page: BrowsePage? = null,
    val isLoading: Boolean = false,
    val error: String? = null,

    // ---- qidiruv ----
    val query: String = "",
    /** null — qidirilmagan (ierarxiya ko'rinadi); bo'sh ro'yxat — topilmadi. */
    val results: List<BrowseNode>? = null,
    val isSearching: Boolean = false,
) {
    /** Qidiruv natijasi ko'rsatilayaptimi (ierarxiya o'rniga). */
    val inSearch: Boolean get() = results != null
}

/**
 * TIF TN ierarxiyasi bo'yicha ko'rish.
 *
 * NEGA BITTA MODEL TO'RT DARAJAGA: har daraja bir xil shakldagi ro'yxat
 * — nom, sanoq, pastga tushish. Ularni alohida ekranga bo'lish to'rtta
 * deyarli bir xil sinf yasardi (DRY buzilardi). Daraja backend javobida
 * keladi va faqat qaysi parametr yuborilishini belgilaydi.
 */
class BrowseScreenModel(
    private val repository: CodeRepository,
) : StateScreenModel<BrowseState>(BrowseState()) {

    /** Qidiruvni kechiktiruvchi ish — har harfda so'rov ketmasin. */
    private var searchJob: Job? = null

    init {
        loadSections()
    }

    fun loadSections() = load { repository.browse() }

    /**
     * Qidiruv matni o'zgardi.
     *
     * So'rov DARROV ketmaydi: 250 ms kutiladi va oldingi ish bekor
     * qilinadi. Aks holda "noutbuk" so'zi yetti so'rov yuborardi va
     * sekin kelgan eski javob yangisini bosib ketishi mumkin edi.
     */
    fun onQuery(text: String) {
        mutableState.value = mutableState.value.copy(query = text)
        searchJob?.cancel()

        if (text.trim().length < MIN_QUERY) {
            // Qidiruvdan chiqamiz — ierarxiya qaytadi.
            mutableState.value = mutableState.value.copy(results = null, isSearching = false)
            return
        }

        searchJob = screenModelScope.launch {
            delay(DEBOUNCE_MS)
            mutableState.value = mutableState.value.copy(isSearching = true, error = null)
            runCatching { repository.search(text.trim(), SUGGEST_LIMIT) }
                .onSuccess { list ->
                    mutableState.value = mutableState.value.copy(results = list, isSearching = false)
                }
                .onFailure { e ->
                    mutableState.value = mutableState.value.copy(
                        isSearching = false,
                        results = emptyList(),
                        error = e.message ?: "Qidirib bo'lmadi",
                    )
                }
        }
    }

    /** Qidiruvni tozalash — ierarxiyaga qaytadi. */
    fun clearQuery() {
        searchJob?.cancel()
        mutableState.value = mutableState.value.copy(
            query = "", results = null, isSearching = false, error = null,
        )
    }

    fun open(node: BrowseNode) {
        val level = mutableState.value.page?.level ?: return
        when (level) {
            BrowseLevel.Sections -> load { repository.browse(section = node.id) }
            BrowseLevel.Groups -> load { repository.browse(group = node.id) }
            BrowseLevel.Headings -> load { repository.browse(heading = node.id) }
            // Barg — bu yerda ochilmaydi; ekran uni chatga uzatadi.
            BrowseLevel.Codes -> Unit
        }
    }

    fun openCrumb(crumb: Crumb) {
        when (crumb.level) {
            BrowseLevel.Sections -> load { repository.browse(section = crumb.id) }
            BrowseLevel.Groups -> load { repository.browse(group = crumb.id) }
            BrowseLevel.Headings -> load { repository.browse(heading = crumb.id) }
            BrowseLevel.Codes -> Unit
        }
    }

    /**
     * Yuklashning yagona yo'li.
     *
     * Har chaqiruvda `isLoading`, `error` va natijani qo'lda boshqarish
     * takrorlanardi va biror joyda unutilishi mumkin edi — masalan xato
     * tozalanmay qolib, eski xabar yangi sahifada osilib turardi.
     */
    private fun load(block: suspend () -> BrowsePage) {
        mutableState.value = mutableState.value.copy(isLoading = true, error = null)
        screenModelScope.launch {
            runCatching { block() }
                .onSuccess { page ->
                    mutableState.value = BrowseState(page = page, isLoading = false)
                }
                .onFailure { e ->
                    mutableState.value = mutableState.value.copy(
                        isLoading = false,
                        error = e.message ?: "Yuklab bo'lmadi",
                    )
                }
        }
    }

    private companion object {
        /** Qidiruv boshlanadigan eng qisqa so'rov. */
        const val MIN_QUERY = 2
        /** Teruvchi to'xtagach shuncha kutiladi (ms). */
        const val DEBOUNCE_MS = 250L
        /** Taklif ro'yxatida nechta variant. */
        const val SUGGEST_LIMIT = 15
    }
}
