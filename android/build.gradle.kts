// Ildiz build fayli — plaginlar shu yerda e'lon qilinadi, qo'llanmaydi.
plugins {
    alias(libs.plugins.android.application) apply false
    alias(libs.plugins.kotlin.compose) apply false
    alias(libs.plugins.kotlin.serialization) apply false
}
