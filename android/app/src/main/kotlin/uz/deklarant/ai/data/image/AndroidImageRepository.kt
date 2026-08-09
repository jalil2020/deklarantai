package uz.deklarant.ai.data.image

import android.content.Context
import android.graphics.Bitmap
import android.graphics.BitmapFactory
import android.graphics.Matrix
import android.net.Uri
import android.util.Base64
import androidx.exifinterface.media.ExifInterface
import java.io.ByteArrayOutputStream
import java.io.InputStream
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import uz.deklarant.ai.domain.model.ChatImage
import uz.deklarant.ai.domain.repository.ImageRepository

/**
 * Rasmni o'qiydi, kichraytiradi va base64 ga o'giradi.
 *
 * UCHTA MAJBURIY QADAM, har biri sababi bilan:
 *
 *  1. KICHRAYTIRISH — telefon kamerasi 4000px atrofida surat beradi.
 *     Base64 hajmni yana ~33% oshiradi va natija backend'ning
 *     MAX_BODY_BYTES (8 MB) chegarasidan oshib ketardi. 1600px invoys
 *     matnini o'qishga yetarli.
 *
 *  2. EXIF BURISH — kamera suratni ko'pincha burilgan holda saqlaydi va
 *     to'g'ri yo'nalishni EXIF ga yozadi. Uni hisobga olmasak, model
 *     yonboshlagan invoysni o'qishga urinardi.
 *
 *  3. JPEG SIQISH — PNG matnli suratda bir necha barobar kattaroq.
 */
internal class AndroidImageRepository(
    private val context: Context,
) : ImageRepository {

    override suspend fun load(uri: String): ChatImage = withContext(Dispatchers.IO) {
        val parsed = Uri.parse(uri)

        // Avval FAQAT o'lchamni o'qiymiz — to'liq rasmni xotiraga
        // yuklamasdan. Katta surat OutOfMemory berishi mumkin.
        val bounds = BitmapFactory.Options().apply { inJustDecodeBounds = true }
        openStream(parsed).use { BitmapFactory.decodeStream(it, null, bounds) }

        val options = BitmapFactory.Options().apply {
            inSampleSize = sampleSizeFor(bounds.outWidth, bounds.outHeight)
        }
        val decoded = openStream(parsed).use { BitmapFactory.decodeStream(it, null, options) }
            ?: error("Rasmni o'qib bo'lmadi")

        val rotated = applyExifRotation(parsed, decoded)
        val scaled = scaleDown(rotated)

        val bytes = ByteArrayOutputStream().use { out ->
            scaled.compress(Bitmap.CompressFormat.JPEG, QUALITY, out)
            out.toByteArray()
        }
        // Bitmap'larni darhol bo'shatamiz — ular katta xotira egallaydi.
        if (scaled !== rotated) rotated.recycle()
        if (rotated !== decoded) decoded.recycle()
        scaled.recycle()

        ChatImage(
            mediaType = "image/jpeg",
            // NO_WRAP: satr ko'chirish belgilari JSON ni buzadi.
            data = Base64.encodeToString(bytes, Base64.NO_WRAP),
        )
    }

    private fun openStream(uri: Uri): InputStream =
        context.contentResolver.openInputStream(uri) ?: error("Rasm ochilmadi")

    /** Ikki darajali kichraytirish — dekodlashda tez va xotirani tejaydi. */
    private fun sampleSizeFor(width: Int, height: Int): Int {
        var sample = 1
        var w = width
        var h = height
        while (w / 2 >= MAX_SIDE && h / 2 >= MAX_SIDE) {
            w /= 2
            h /= 2
            sample *= 2
        }
        return sample
    }

    private fun applyExifRotation(uri: Uri, bitmap: Bitmap): Bitmap {
        val degrees = runCatching {
            openStream(uri).use { stream ->
                when (ExifInterface(stream).getAttributeInt(
                    ExifInterface.TAG_ORIENTATION,
                    ExifInterface.ORIENTATION_NORMAL,
                )) {
                    ExifInterface.ORIENTATION_ROTATE_90 -> 90f
                    ExifInterface.ORIENTATION_ROTATE_180 -> 180f
                    ExifInterface.ORIENTATION_ROTATE_270 -> 270f
                    else -> 0f
                }
            }
        }.getOrDefault(0f)

        if (degrees == 0f) return bitmap
        val matrix = Matrix().apply { postRotate(degrees) }
        return Bitmap.createBitmap(bitmap, 0, 0, bitmap.width, bitmap.height, matrix, true)
    }

    /** Uzun tomonni MAX_SIDE ga keltiradi, nisbatni saqlab. */
    private fun scaleDown(bitmap: Bitmap): Bitmap {
        val longest = maxOf(bitmap.width, bitmap.height)
        if (longest <= MAX_SIDE) return bitmap
        val ratio = MAX_SIDE.toFloat() / longest
        return Bitmap.createScaledBitmap(
            bitmap,
            (bitmap.width * ratio).toInt().coerceAtLeast(1),
            (bitmap.height * ratio).toInt().coerceAtLeast(1),
            true,
        )
    }

    private companion object {
        const val MAX_SIDE = 1600
        const val QUALITY = 85
    }
}
