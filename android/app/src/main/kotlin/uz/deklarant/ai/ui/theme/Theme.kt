package uz.deklarant.ai.ui.theme

import android.os.Build
import androidx.compose.foundation.isSystemInDarkTheme
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.darkColorScheme
import androidx.compose.material3.dynamicDarkColorScheme
import androidx.compose.material3.dynamicLightColorScheme
import androidx.compose.material3.lightColorScheme
import androidx.compose.runtime.Composable
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.platform.LocalContext

private val Brand = Color(0xFF2563EB)
private val BrandDark = Color(0xFF3B82F6)

private val LightColors = lightColorScheme(
    primary = Brand,
    secondary = Color(0xFF475569),
)

private val DarkColors = darkColorScheme(
    primary = BrandDark,
    secondary = Color(0xFF94A3B8),
)

@Composable
fun DeklarantTheme(
    darkTheme: Boolean = isSystemInDarkTheme(),
    content: @Composable () -> Unit,
) {
    // Android 12+ da tizim ranglariga moslashamiz — ilova qurilma bilan
    // bir butun ko'rinadi. Eskilarida o'z palitramiz.
    val colors = when {
        Build.VERSION.SDK_INT >= Build.VERSION_CODES.S -> {
            val ctx = LocalContext.current
            if (darkTheme) dynamicDarkColorScheme(ctx) else dynamicLightColorScheme(ctx)
        }
        darkTheme -> DarkColors
        else -> LightColors
    }
    MaterialTheme(colorScheme = colors, content = content)
}
