package uz.deklarant.ai

import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.activity.enableEdgeToEdge
import androidx.compose.runtime.CompositionLocalProvider
import androidx.compose.runtime.remember
import uz.deklarant.ai.ui.RootScreen
import uz.deklarant.ai.ui.common.ChatHandoff
import uz.deklarant.ai.ui.common.LocalChatHandoff
import uz.deklarant.ai.ui.common.LocalContainer
import uz.deklarant.ai.ui.common.LocalLoginPrompt
import uz.deklarant.ai.ui.common.LoginPrompt
import uz.deklarant.ai.ui.theme.DeklarantTheme

class MainActivity : ComponentActivity() {

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        enableEdgeToEdge()

        val container = (application as DeklarantApp).container

        setContent {
            // Bo'limlar orasidagi savol uzatgichi Activity umriga bog'langan.
            val handoff = remember { ChatHandoff() }
            // Yon menyudan kirish oynasini so'rash.
            val loginPrompt = remember { LoginPrompt() }

            DeklarantTheme {
                CompositionLocalProvider(
                    LocalContainer provides container,
                    LocalChatHandoff provides handoff,
                    LocalLoginPrompt provides loginPrompt,
                ) {
                    RootScreen()
                }
            }
        }
    }
}
