package uz.deklarant.ai.ui

import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.Chat
import androidx.compose.material.icons.automirrored.filled.MenuBook
import androidx.compose.material.icons.filled.AccountTree
import androidx.compose.runtime.Composable
import androidx.compose.ui.graphics.painter.Painter
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.graphics.vector.rememberVectorPainter
import cafe.adriel.voyager.navigator.Navigator
import cafe.adriel.voyager.navigator.tab.Tab
import cafe.adriel.voyager.navigator.tab.TabOptions
import uz.deklarant.ai.ui.browse.BrowseScreen
import uz.deklarant.ai.ui.chat.ChatScreen
import uz.deklarant.ai.ui.laws.LawsScreen

// Har tab ICHIDA o'z Navigator'i bor.
//
// NEGA: Navigator ekran modellarining umrini boshqaradi va ichki
// o'tishlarni (kelajakda: kod tafsiloti, hisob-kitob natijasi)
// imkonli qiladi. Navigatorsiz `rememberScreenModel` ishlamaydi —
// unga ekran hayotiy sikli kerak.

object ChatTab : Tab {
    override val options: TabOptions
        @Composable get() = TabOptions(
            index = 0u,
            title = "Chat",
            icon = rememberIcon(Icons.AutoMirrored.Filled.Chat),
        )

    @Composable
    override fun Content() {
        Navigator(ChatScreen())
    }
}

object CodesTab : Tab {
    override val options: TabOptions
        @Composable get() = TabOptions(
            index = 1u,
            title = "TIF TN",
            icon = rememberIcon(Icons.Default.AccountTree),
        )

    @Composable
    override fun Content() {
        Navigator(BrowseScreen())
    }
}

object LawsTab : Tab {
    override val options: TabOptions
        @Composable get() = TabOptions(
            index = 2u,
            title = "Qonunlar",
            icon = rememberIcon(Icons.AutoMirrored.Filled.MenuBook),
        )

    @Composable
    override fun Content() {
        Navigator(LawsScreen())
    }
}

/** TabOptions Painter kutadi, bizda esa ImageVector — bir joyda o'giramiz. */
@Composable
private fun rememberIcon(vector: ImageVector): Painter =
    rememberVectorPainter(vector)
