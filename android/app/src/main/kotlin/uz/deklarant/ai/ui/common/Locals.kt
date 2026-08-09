package uz.deklarant.ai.ui.common

import androidx.compose.runtime.staticCompositionLocalOf
import uz.deklarant.ai.di.AppContainer

/**
 * Bog'liqliklar konteyneri kompozitsiya bo'ylab uzatiladi.
 *
 * staticCompositionLocalOf — qiymat ilova ishlashi davomida O'ZGARMAYDI,
 * shuning uchun uni o'qiydigan joylarni qayta chizish shart emas.
 */
val LocalContainer = staticCompositionLocalOf<AppContainer> {
    error("AppContainer berilmagan — MainActivity da CompositionLocalProvider qo'yilganmi?")
}
