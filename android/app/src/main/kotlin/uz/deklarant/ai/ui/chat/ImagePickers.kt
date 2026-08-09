package uz.deklarant.ai.ui.chat

import android.content.Context
import android.net.Uri
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.runtime.Composable
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.core.content.FileProvider
import java.io.File

/**
 * Rasm tanlash — ikkita yo'l, ikkalasi ham RUXSATSIZ.
 *
 *  Galereya — Photo Picker (`PickMultipleVisualMedia`): tizim tanlagichi,
 *             faqat tanlangan faylga kirish beradi. Android 13+ da
 *             tizimga o'rnatilgan, eskilarida esa kutubxona uni taqlid
 *             qiladi. READ_MEDIA_IMAGES kerak emas.
 *
 *  Kamera   — `TakePicture` + FileProvider: surat tizimning kamera
 *             ilovasi orqali olinadi. CAMERA ruxsati e'lon qilinmagani
 *             uchun undan so'ralmaydi ham.
 *
 * Shu tufayli ilovada birorta ham ish vaqti ruxsat dialogi yo'q.
 */

/** Bir marta tanlanadigan rasmlar soni. */
private const val MAX_IMAGES = 5

/** Galereyadan tanlash. */
@Composable
internal fun rememberGalleryPicker(onPicked: (List<String>) -> Unit): () -> Unit {
    val launcher = rememberLauncherForActivityResult(
        ActivityResultContracts.PickMultipleVisualMedia(MAX_IMAGES),
    ) { uris -> onPicked(uris.map { it.toString() }) }

    return {
        launcher.launch(
            androidx.activity.result.PickVisualMediaRequest(
                ActivityResultContracts.PickVisualMedia.ImageOnly,
            ),
        )
    }
}

/**
 * Kameradan surat olish.
 *
 * `TakePicture` faqat "muvaffaqiyatli/yo'q" qaytaradi, surat esa biz
 * bergan manzilga yoziladi — shuning uchun manzilni eslab turishimiz
 * kerak.
 */
@Composable
internal fun rememberCameraCapture(
    context: Context,
    onCaptured: (String) -> Unit,
): () -> Unit {
    val pending = remember { mutableStateOf<Uri?>(null) }

    val launcher = rememberLauncherForActivityResult(
        ActivityResultContracts.TakePicture(),
    ) { success ->
        val uri = pending.value
        pending.value = null
        // Bekor qilinsa `success` false — bu xato emas, jim o'tamiz.
        if (success && uri != null) onCaptured(uri.toString())
    }

    return {
        val uri = createCameraUri(context)
        pending.value = uri
        launcher.launch(uri)
    }
}

/**
 * Kesh papkasida yangi surat uchun manzil yasaydi.
 *
 * Nom vaqt bilan — bir seansda bir necha surat olinsa, ular
 * bir-birini yozib yubormasin.
 */
private fun createCameraUri(context: Context): Uri {
    val dir = File(context.cacheDir, "camera").apply { mkdirs() }
    val file = File(dir, "surat_${System.currentTimeMillis()}.jpg")
    return FileProvider.getUriForFile(context, "${context.packageName}.fileprovider", file)
}
