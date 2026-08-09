package uz.deklarant.ai.ui.chat

import android.graphics.BitmapFactory
import android.util.Base64
import androidx.compose.foundation.Image
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.imePadding
import androidx.compose.foundation.layout.navigationBarsPadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.widthIn
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.LazyRow
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.lazy.rememberLazyListState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.Send
import androidx.compose.material.icons.filled.Check
import androidx.compose.material.icons.filled.Close
import androidx.compose.material.icons.filled.ContentCopy
import androidx.compose.material.icons.filled.DeleteOutline
import androidx.compose.material.icons.filled.PhotoCamera
import androidx.compose.material.icons.filled.PhotoLibrary
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.FilterChip
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.derivedStateOf
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.asImageBitmap
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.platform.LocalClipboardManager
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.text.AnnotatedString
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import cafe.adriel.voyager.core.model.rememberScreenModel
import cafe.adriel.voyager.core.screen.Screen
import coil.compose.AsyncImage
import kotlinx.coroutines.delay
import uz.deklarant.ai.R
import uz.deklarant.ai.domain.model.ChatImage
import uz.deklarant.ai.domain.model.ChatMessage
import uz.deklarant.ai.domain.model.ChatMode
import uz.deklarant.ai.ui.auth.LoginDialog
import uz.deklarant.ai.ui.auth.LoginScreenModel
import uz.deklarant.ai.ui.common.LocalChatHandoff
import uz.deklarant.ai.ui.common.Markdown
import uz.deklarant.ai.ui.common.LocalContainer
import uz.deklarant.ai.ui.common.LocalLoginPrompt

/**
 * Chat — ilovaning asosiy ekrani.
 *
 * Rasm biriktirish shu yerda: tovar yoki invoys surati yuborilsa, AI
 * undagi tovar, miqdor va narxni o'qib kodni taklif qiladi.
 */
class ChatScreen : Screen {

    // Kalit turg'un: tab almashganda ekran holati saqlanib qolsin.
    override val key: String = "chat"

    @Composable
    override fun Content() {
        val container = LocalContainer.current
        val context = LocalContext.current
        val model = rememberScreenModel {
            ChatScreenModel(
                container.chatRepository,
                container.imageRepository,
                container.chatHistory,
            )
        }
        val state by model.state.collectAsState()

        var input by remember { mutableStateOf("") }

        // Boshqa bo'limdan kelgan savolni maydonga qo'yamiz.
        // consume() uni bir marta beradi — foydalanuvchi tergan matn
        // qayta chizishda o'chib ketmasin.
        val handoff = LocalChatHandoff.current
        LaunchedEffect(handoff.hasPending) {
            handoff.consume()?.let { input = it }
        }

        val openGallery = rememberGalleryPicker { uris -> model.attach(uris) }
        val openCamera = rememberCameraCapture(context) { uri -> model.attach(listOf(uri)) }

        val listState = rememberLazyListState()

        // Avtomatik surish FAQAT foydalanuvchi pastda turganda.
        //
        // NEGA SHART: ilgari har bo'lakda `animateScrollToItem` chaqirilar
        // edi. Javob 30–50 soniya yozilgani uchun foydalanuvchi ko'pincha
        // shu paytda yuqoriga qarab jadvalni yoki oldingi javobni o'qiydi
        // — va ro'yxat uni har safar pastga tortib tashlardi.
        //
        // "Pastda" — oxirgi element ko'rinib turibdi degani. 2 element
        // zaxira: bir necha piksel farq bilan "pastda emas" bo'lib
        // qolmasin.
        val atBottom by remember {
            derivedStateOf {
                val last = listState.layoutInfo.visibleItemsInfo.lastOrNull()
                    ?: return@derivedStateOf true
                last.index >= listState.layoutInfo.totalItemsCount - 2
            }
        }
        LaunchedEffect(state.messages.size, state.streaming) {
            // Foydalanuvchi O'ZI yuborgan xabarga esa DOIM tushamiz —
            // u yuqorida turgan bo'lsa ham, o'z savolini ko'rishni
            // kutadi.
            val mine = state.messages.lastOrNull()?.role == ChatMessage.Role.User
            if (!atBottom && !mine) return@LaunchedEffect
            val count = state.messages.size + if (state.streaming.isNotEmpty()) 1 else 0
            if (count > 0) listState.animateScrollToItem(count - 1)
        }

        val user by container.authRepository.user.collectAsState()
        var showLogin by remember { mutableStateOf(false) }

        // Tag kerak: shu ekranda allaqachon ChatScreenModel bor.
        val loginModel = rememberScreenModel(tag = "login") {
            LoginScreenModel(container.authRepository)
        }
        if (showLogin) LoginDialog(loginModel, onDismiss = { showLogin = false })

        // Yon menyudagi "Kirish" oynani SHU YERDA ochadi — sabab
        // LoginPrompt izohida (menyu oddiy composable, u yerda
        // ScreenModel yasab bo'lmaydi).
        val loginPrompt = LocalLoginPrompt.current
        LaunchedEffect(loginPrompt.requested) {
            if (loginPrompt.requested) {
                showLogin = true
                loginPrompt.clear()
            }
        }

        Column(Modifier.fillMaxSize().imePadding()) {
            // Rejim chatga xos sozlama, shuning uchun shu yerda qoladi.
            // Foydalanuvchi ma'lumoti esa yon menyuga ko'chdi: bu qator
            // telefon enida ism, rol, kvota va ikki tugmani sig'dira
            // olmasdi va matn qirqilardi.
            Row(
                Modifier.fillMaxWidth().padding(start = 12.dp, end = 4.dp, top = 6.dp, bottom = 6.dp),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                ModeRow(mode = state.mode, onSelect = model::setMode)
                Spacer(Modifier.weight(1f))

                // Tozalash faqat GAP BO'LSA ko'rinadi — bo'sh ekranda
                // u faqat chalg'itardi.
                if (state.messages.isNotEmpty()) {
                    var confirm by remember { mutableStateOf(false) }
                    IconButton(onClick = { confirm = true }) {
                        Icon(
                            imageVector = Icons.Default.DeleteOutline,
                            contentDescription = "Suhbatni tozalash",
                            tint = MaterialTheme.colorScheme.onSurfaceVariant,
                        )
                    }
                    // Tasdiq SO'RALADI: suhbat endi diskda saqlanadi,
                    // ya'ni tasodifiy bosish bir necha kunlik yozishmani
                    // o'chirib yuborishi mumkin.
                    if (confirm) {
                        AlertDialog(
                            onDismissRequest = { confirm = false },
                            title = { Text("Suhbat tozalansinmi?") },
                            text = { Text("Barcha xabarlar qurilmadan o'chiriladi. Buni ortga qaytarib bo'lmaydi.") },
                            confirmButton = {
                                TextButton(onClick = {
                                    model.clearHistory()
                                    confirm = false
                                }) { Text("Tozalash") }
                            },
                            dismissButton = {
                                TextButton(onClick = { confirm = false }) { Text("Bekor qilish") }
                            },
                        )
                    }
                }
            }

            state.error?.let { message ->
                ErrorBanner(
                    message = message,
                    // Qayta urinish faqat javob kelmagan holatda ma'noli:
                    // rasm o'qilmagani haqidagi xatoni qayta yuborish
                    // hech narsani o'zgartirmaydi.
                    onRetry = if (state.messages.lastOrNull()?.role == ChatMessage.Role.User) {
                        model::retry
                    } else {
                        null
                    },
                    onDismiss = model::dismissError,
                )
            }

            LazyColumn(
                state = listState,
                modifier = Modifier.weight(1f).fillMaxWidth(),
                contentPadding = PaddingValues(12.dp),
                verticalArrangement = Arrangement.spacedBy(8.dp),
            ) {
                items(state.messages) { Bubble(it) }
                if (state.streaming.isNotEmpty()) {
                    item { Bubble(ChatMessage(ChatMessage.Role.Assistant, state.streaming)) }
                }
                if (state.messages.isEmpty() && state.streaming.isEmpty()) {
                    item { Hint() }
                }
            }

            if (state.pending.isNotEmpty() || state.isAttaching) {
                PendingRow(
                    pending = state.pending,
                    isLoading = state.isAttaching,
                    onRemove = model::removeImage,
                )
            }

            InputRow(
                value = input,
                onChange = { input = it },
                isSending = state.isSending,
                onSend = {
                    model.send(input)
                    input = ""
                },
                onStop = model::stop,
                onGallery = openGallery,
                onCamera = openCamera,
                locked = user == null,
                onNeedLogin = { showLogin = true },
                modifier = Modifier.navigationBarsPadding(),
            )
        }
    }
}

@Composable
private fun ModeRow(mode: ChatMode, onSelect: (ChatMode) -> Unit, modifier: Modifier = Modifier) {
    Row(modifier, horizontalArrangement = Arrangement.spacedBy(8.dp)) {
        FilterChip(
            selected = mode == ChatMode.Declarant,
            onClick = { onSelect(ChatMode.Declarant) },
            label = { Text("Deklarant") },
        )
        FilterChip(
            selected = mode == ChatMode.Business,
            onClick = { onSelect(ChatMode.Business) },
            label = { Text("Tadbirkor") },
        )
    }
}

@Composable
private fun Bubble(message: ChatMessage) {
    val isUser = message.role == ChatMessage.Role.User
    val bg = if (isUser) MaterialTheme.colorScheme.primary else MaterialTheme.colorScheme.surfaceVariant
    val fg = if (isUser) MaterialTheme.colorScheme.onPrimary else MaterialTheme.colorScheme.onSurfaceVariant

    Box(
        Modifier.fillMaxWidth(),
        contentAlignment = if (isUser) Alignment.CenterEnd else Alignment.CenterStart,
    ) {
        // AI javobiga KENGROQ joy: unda jadval bo'ladi (boj, QQS, yig'im
        // GTD kodlari bo'yicha) va 320dp da ustunlar har qatorda ikkiga
        // bo'linib ketardi. Foydalanuvchi xabari qisqa, unga keng joy
        // kerak emas — u o'ngga tekislangani ko'rinib tursin.
        val width =
            if (isUser) Modifier.widthIn(max = 320.dp)
            else Modifier.fillMaxWidth(0.97f)

        Surface(color = bg, shape = RoundedCornerShape(16.dp), modifier = width) {
            Column(
                Modifier.padding(horizontal = 6.dp, vertical = 6.dp),
                verticalArrangement = Arrangement.spacedBy(6.dp),
            ) {
                // Yuborilgan rasm KO'RINIB TURISHI kerak: foydalanuvchi
                // nima jo'natganini eslab qolsin va noto'g'ri surat
                // ketgan bo'lsa darrov bilsin.
                message.images.forEach { SentImage(it) }

                if (message.content.isNotEmpty()) {
                    val padding = Modifier.padding(horizontal = 8.dp, vertical = 4.dp)
                    // Foydalanuvchi xabari — o'zi yozgani, o'sha holicha.
                    // Markdown faqat AI javobiga qo'llanadi: aks holda
                    // savolda yozilgan yulduzcha yoki tik chiziq matnni
                    // "bezab" yuborardi.
                    if (isUser) {
                        Text(
                            text = message.content,
                            color = fg,
                            style = MaterialTheme.typography.bodyMedium,
                            modifier = padding,
                        )
                    } else {
                        Markdown(text = message.content, color = fg, modifier = padding)
                        CopyRow(message.content)
                    }
                }
            }
        }
    }
}

/**
 * Javobni nusxalash.
 *
 * NEGA KERAK: deklarant javobdagi kodni, summani yoki modda raqamini
 * boshqa joyga (GTD, xat, hisob) ko'chiradi. Markdown chizilgandan
 * keyin matnni uzoq bosib tanlab ham bo'lmaydi — `Text` tanlanmaydi.
 * Nusxa olinadigan matn — XOM Markdown: jadval belgilarini yo'qotib
 * yuborsak, boshqa joyga qo'yilganda ustunlar aralashib ketardi.
 */
@Composable
private fun CopyRow(text: String) {
    val clipboard = LocalClipboardManager.current
    var copied by remember { mutableStateOf(false) }

    // Tasdiq ikki soniya ko'rinadi. Busiz bosilgani bilinmasdi —
    // buferga yozish hech qanday belgi bermaydi.
    LaunchedEffect(copied) {
        if (copied) {
            delay(2000)
            copied = false
        }
    }

    Row(
        Modifier.fillMaxWidth().padding(horizontal = 4.dp),
        horizontalArrangement = Arrangement.End,
        verticalAlignment = Alignment.CenterVertically,
    ) {
        TextButton(
            onClick = {
                clipboard.setText(AnnotatedString(text))
                copied = true
            },
            contentPadding = PaddingValues(horizontal = 10.dp, vertical = 2.dp),
        ) {
            Icon(
                imageVector = if (copied) Icons.Default.Check else Icons.Default.ContentCopy,
                contentDescription = null,
                modifier = Modifier.size(15.dp),
            )
            Text(
                text = if (copied) " Nusxa olindi" else " Nusxa olish",
                style = MaterialTheme.typography.labelSmall,
            )
        }
    }
}

/**
 * Yuborilgan rasm base64 dan chiziladi.
 *
 * `remember(data)` MUHIM: dekodlash qimmat, uni har qayta chizishda
 * takrorlash oqim kelayotganda ro'yxatni sekinlashtirardi.
 */
@Composable
private fun SentImage(image: ChatImage) {
    val bitmap = remember(image.data) {
        runCatching {
            val bytes = Base64.decode(image.data, Base64.NO_WRAP)
            BitmapFactory.decodeByteArray(bytes, 0, bytes.size)?.asImageBitmap()
        }.getOrNull()
    }
    if (bitmap != null) {
        Image(
            bitmap = bitmap,
            contentDescription = "Biriktirilgan rasm",
            contentScale = ContentScale.Fit,
            modifier = Modifier
                .widthIn(max = 260.dp)
                .clip(RoundedCornerShape(12.dp)),
        )
    }
}

/** Yuborishdan oldingi rasmlar — o'chirish tugmasi bilan. */
@Composable
private fun PendingRow(
    pending: List<PendingImage>,
    isLoading: Boolean,
    onRemove: (String) -> Unit,
) {
    LazyRow(
        Modifier.fillMaxWidth().padding(horizontal = 8.dp, vertical = 4.dp),
        horizontalArrangement = Arrangement.spacedBy(8.dp),
    ) {
        items(pending) { item ->
            Box {
                AsyncImage(
                    model = item.uri,
                    contentDescription = "Biriktirilgan rasm",
                    contentScale = ContentScale.Crop,
                    modifier = Modifier.size(64.dp).clip(RoundedCornerShape(10.dp)),
                )
                // O'chirish — noto'g'ri surat tanlansa qayta boshlash
                // kerak bo'lmasin.
                IconButton(
                    onClick = { onRemove(item.uri) },
                    modifier = Modifier.align(Alignment.TopEnd).size(24.dp),
                ) {
                    Icon(
                        imageVector = Icons.Default.Close,
                        contentDescription = "Olib tashlash",
                        tint = MaterialTheme.colorScheme.error,
                    )
                }
            }
        }
        if (isLoading) {
            item {
                Box(Modifier.size(64.dp), contentAlignment = Alignment.Center) {
                    CircularProgressIndicator(Modifier.size(24.dp))
                }
            }
        }
    }
}

@Composable
private fun Hint() {
    Column(
        Modifier.fillMaxWidth().padding(24.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.spacedBy(16.dp),
    ) {
        Image(
            painter = painterResource(R.drawable.logo_mark),
            contentDescription = null,
            modifier = Modifier.size(112.dp),
        )
        Text(
            text = "Tovar nomini yozing yoki surat biriktiring.\n" +
                "TIF TN va Qonunlar — yuqoridagi ☰ menyuda.",
            style = MaterialTheme.typography.bodyMedium,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
            textAlign = TextAlign.Center,
        )
    }
}

@Composable
private fun ErrorBanner(message: String, onRetry: (() -> Unit)?, onDismiss: () -> Unit) {
    Row(
        Modifier
            .fillMaxWidth()
            .background(MaterialTheme.colorScheme.errorContainer)
            .padding(start = 12.dp, top = 4.dp, bottom = 4.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Text(
            text = message,
            color = MaterialTheme.colorScheme.onErrorContainer,
            style = MaterialTheme.typography.bodySmall,
            modifier = Modifier.weight(1f),
        )
        onRetry?.let {
            TextButton(onClick = it) {
                Text(
                    text = "Qayta urinish",
                    style = MaterialTheme.typography.labelMedium,
                    color = MaterialTheme.colorScheme.onErrorContainer,
                )
            }
        }
        IconButton(onClick = onDismiss) {
            Icon(
                imageVector = Icons.Default.Close,
                contentDescription = "Yopish",
                tint = MaterialTheme.colorScheme.onErrorContainer,
            )
        }
    }
}

@Composable
private fun InputRow(
    value: String,
    onChange: (String) -> Unit,
    isSending: Boolean,
    onSend: () -> Unit,
    onStop: () -> Unit,
    onGallery: () -> Unit,
    onCamera: () -> Unit,
    /** Kirilmagan bo'lsa maydon yopiq va bosilganda kirish oynasi ochiladi. */
    locked: Boolean = false,
    onNeedLogin: () -> Unit = {},
    modifier: Modifier = Modifier,
) {
    // Yopiq holatda har qanday urinish kirish oynasiga olib boradi —
    // bitta joyda ushlanadi (DRY).
    fun guard(action: () -> Unit): () -> Unit = { if (locked) onNeedLogin() else action() }

    Row(
        modifier.fillMaxWidth().padding(horizontal = 4.dp, vertical = 4.dp),
        verticalAlignment = Alignment.Bottom,
    ) {
        IconButton(onClick = guard(onCamera)) {
            Icon(Icons.Default.PhotoCamera, contentDescription = "Surat olish")
        }
        IconButton(onClick = guard(onGallery)) {
            Icon(Icons.Default.PhotoLibrary, contentDescription = "Galereyadan tanlash")
        }

        Box(Modifier.weight(1f)) {
            OutlinedTextField(
                value = value,
                onValueChange = onChange,
                modifier = Modifier.fillMaxWidth(),
                placeholder = {
                    Text(
                        if (locked) "Savol berish uchun kiring — bosing"
                        else "So'rov yozing...",
                    )
                },
                readOnly = locked,
                maxLines = 5,
            )

            // Yopiq holatda maydon USTIGA shaffof qatlam qo'yiladi.
            //
            // NEGA: `Modifier.clickable` ni maydonning o'ziga qo'yish
            // ISHLAMAYDI — TextField bosishni o'zi yutib, kursorni
            // qo'yadi va tashqi bosish hech qachon chaqirilmaydi
            // (emulyatorda shunday chiqdi). `enabled = false` esa
            // bosishni umuman o'tkazmaydi va maydon o'chgan ko'rinadi.
            if (locked) {
                Spacer(
                    Modifier
                        .matchParentSize()
                        .clickable(onClick = onNeedLogin),
                )
            }
        }

        // Yuborish va TO'XTATISH bitta tugmada: javob 20-50 soniya
        // kelayotganda foydalanuvchida uni to'xtatish imkoni bo'lishi kerak.
        IconButton(onClick = guard { if (isSending) onStop() else onSend() }) {
            Icon(
                imageVector = if (isSending) Icons.Default.Close else Icons.AutoMirrored.Filled.Send,
                contentDescription = if (isSending) "To'xtatish" else "Yuborish",
                tint = if (isSending) MaterialTheme.colorScheme.error else MaterialTheme.colorScheme.primary,
            )
        }
    }
}

