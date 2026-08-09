plugins {
    alias(libs.plugins.android.application)
    alias(libs.plugins.kotlin.compose)
    alias(libs.plugins.kotlin.serialization)
}

android {
    namespace = "uz.deklarant.ai"
    compileSdk = 36

    defaultConfig {
        applicationId = "uz.deklarant.ai"
        minSdk = 26
        targetSdk = 36
        versionCode = 1
        versionName = "0.1.0"

        // Backend manzili KOD ICHIDA YOZILMAYDI: emulyator, LAN va
        // ishlab chiqarish uchun uchta boshqa manzil kerak bo'ladi.
        // Qiymat gradle.properties dan yoki -PapiBaseUrl bilan keladi.
        val apiBaseUrl = (project.findProperty("apiBaseUrl") as String?)
            ?: (project.findProperty("deklarant.apiBaseUrl") as String?)
            ?: "http://10.0.2.2:8080"
        buildConfigField("String", "API_BASE_URL", "\"$apiBaseUrl\"")

        // Mijoz kaliti. Bo'sh bo'lsa ilova /api/session dan anonim
        // token oladi — ya'ni kalitsiz ham ishlaydi.
        //
        // DIQQAT: APK ichidagi kalit SIR EMAS, uni ajratib olish mumkin.
        // U shunchaki ilovani anonim mijozdan ajratadi (statistika va
        // kelajakdagi kvota uchun).
        val apiKey = (project.findProperty("apiKey") as String?)
            ?: (project.findProperty("deklarant.apiKey") as String?)
            ?: ""
        buildConfigField("String", "API_KEY", "\"$apiKey\"")
    }

    buildTypes {
        debug {
            // Shifrlanmagan HTTP faqat DEBUG da — mahalliy backend uchun.
            // Release da network_security_config buni taqiqlaydi.
            isMinifyEnabled = false
        }
        release {
            isMinifyEnabled = true
            isShrinkResources = true
            proguardFiles(
                getDefaultProguardFile("proguard-android-optimize.txt"),
                "proguard-rules.pro",
            )
        }
    }

    buildFeatures {
        compose = true
        buildConfig = true
    }

    compileOptions {
        sourceCompatibility = JavaVersion.VERSION_17
        targetCompatibility = JavaVersion.VERSION_17
    }

    kotlin {
        compilerOptions {
            jvmTarget.set(org.jetbrains.kotlin.gradle.dsl.JvmTarget.JVM_17)
        }
    }

    sourceSets["main"].java.srcDirs("src/main/kotlin")

    packaging {
        resources.excludes += "/META-INF/{AL2.0,LGPL2.1}"
    }
}

dependencies {
    implementation(platform(libs.compose.bom))
    implementation(libs.compose.ui)
    implementation(libs.compose.ui.graphics)
    implementation(libs.compose.ui.tooling.preview)
    implementation(libs.compose.material3)
    implementation(libs.compose.material.icons)
    debugImplementation(libs.compose.ui.tooling)

    implementation(libs.androidx.activity.compose)
    implementation(libs.androidx.lifecycle.runtime)

    implementation(libs.kotlinx.coroutines)
    implementation(libs.kotlinx.serialization.json)

    implementation(libs.ktor.client.core)
    implementation(libs.ktor.client.okhttp)
    implementation(libs.ktor.client.content.negotiation)
    implementation(libs.ktor.serialization.json)
    implementation(libs.ktor.client.logging)

    implementation(libs.voyager.navigator)
    implementation(libs.voyager.screenmodel)
    implementation(libs.voyager.tab.navigator)
    implementation(libs.voyager.transitions)

    implementation(libs.coil.compose)
    implementation(libs.androidx.exifinterface)
}
