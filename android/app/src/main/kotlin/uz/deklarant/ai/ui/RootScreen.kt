package uz.deklarant.ai.ui

import androidx.activity.compose.BackHandler
import androidx.compose.foundation.Image
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.Login
import androidx.compose.material.icons.automirrored.filled.Logout
import androidx.compose.material.icons.filled.Menu
import androidx.compose.material3.DrawerValue
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.ModalDrawerSheet
import androidx.compose.material3.ModalNavigationDrawer
import androidx.compose.material3.NavigationDrawerItem
import androidx.compose.material3.NavigationDrawerItemDefaults
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.TopAppBar
import androidx.compose.material3.rememberDrawerState
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.unit.dp
import cafe.adriel.voyager.navigator.tab.CurrentTab
import cafe.adriel.voyager.navigator.tab.LocalTabNavigator
import cafe.adriel.voyager.navigator.tab.Tab
import cafe.adriel.voyager.navigator.tab.TabNavigator
import kotlinx.coroutines.launch
import uz.deklarant.ai.R
import uz.deklarant.ai.domain.model.User
import uz.deklarant.ai.ui.common.LocalChatHandoff
import uz.deklarant.ai.ui.common.LocalContainer
import uz.deklarant.ai.ui.common.LocalLoginPrompt

/**
 * Ilova ildizi — bo'limlar YON MENYUDA (drawer).
 *
 * NEGA PASTKI PANEL EMAS: chat — asosiy ekran va u ekranning butun
 * balandligini talab qiladi. Pastda doimiy panel turganda kiritish
 * maydoni, klaviatura va panel bir-biri bilan joy talashardi; telefonda
 * suhbat uchun qoladigan joy sezilarli qisqarardi. Pastki panel
 * bo'limlar TENG va tez-tez almashtiriladigan bo'lsa oqlanadi, bu yerda
 * esa nisbat boshqacha: chat — 90% vaqt, TIF TN va Qonunlar — qidirib
 * kelinadigan yordamchi bo'limlar.
 *
 * Shuning uchun ekranning pasti FAQAT chatga tegishli, bo'limlar esa
 * yuqoridagi menyu tugmasi ortida.
 */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun RootScreen() {
    val container = LocalContainer.current

    // Saqlangan token hali amaldami — ilova ochilishida bir marta.
    //
    // Busiz foydalanuvchi har safar qayta kirishga majbur bo'lardi:
    // token diskda turadi, lekin uning kim ekanini faqat server aytadi.
    LaunchedEffect(Unit) { container.authRepository.restore() }

    TabNavigator(ChatTab) { tabNavigator ->
        val handoff = LocalChatHandoff.current
        val loginPrompt = LocalLoginPrompt.current
        val drawerState = rememberDrawerState(DrawerValue.Closed)
        val scope = rememberCoroutineScope()
        val user by container.authRepository.user.collectAsState()

        // Savol kutayotgan bo'lsa — chat bo'limiga o'tamiz.
        //
        // Bu yerda, tab'lar ichida emas: aks holda har bo'lim boshqa
        // bo'limga o'tishni bilishi kerak bo'lardi va ular bir-biriga
        // bog'lanib qolardi.
        LaunchedEffect(handoff.hasPending) {
            if (handoff.hasPending) {
                tabNavigator.current = ChatTab
                drawerState.close()
            }
        }

        // ORQAGA tugmasi ochiq menyuni yopadi.
        //
        // ⚠️ `ModalNavigationDrawer` buni O'ZI qilmaydi. Emulyatorda
        // sinaldi: menyu ochiq turganda "orqaga" bosilsa ilova butunlay
        // yopilib, bosh ekranga chiqib ketdi. Foydalanuvchi esa menyu
        // yopilishini kutadi.
        BackHandler(enabled = drawerState.isOpen) {
            scope.launch { drawerState.close() }
        }

        ModalNavigationDrawer(
            drawerState = drawerState,
            // Ochiq menyuda surish yopadi; yopiq holatda esa surish
            // O'CHIRILADI — aks holda chatdagi gorizontal ishoralar va
            // TIF TN dagi yo'l zanjiri surilishi menyuni tortib ochardi.
            gesturesEnabled = drawerState.isOpen,
            drawerContent = {
                ModalDrawerSheet {
                    DrawerHeader()
                    HorizontalDivider()

                    sections.forEach { tab ->
                        DrawerItem(
                            tab = tab,
                            selected = tabNavigator.current.key == tab.key,
                            onClick = {
                                tabNavigator.current = tab
                                scope.launch { drawerState.close() }
                            },
                        )
                    }

                    HorizontalDivider(Modifier.padding(vertical = 8.dp))
                    AccountBlock(
                        user = user,
                        onLogin = {
                            // Oyna chat ekranida ochiladi (izohi
                            // LoginPrompt da), shuning uchun avval o'sha
                            // bo'limga o'tamiz.
                            tabNavigator.current = ChatTab
                            loginPrompt.request()
                            scope.launch { drawerState.close() }
                        },
                        onLogout = {
                            scope.launch {
                                container.authRepository.logout()
                                // Suhbat diskda qoladi, shuning uchun uni
                                // CHIQISHDA o'chiramiz: aks holda bir
                                // qurilmadan foydalanadigan keyingi odam
                                // oldingisining yozishmasini ko'rardi.
                                container.chatHistory.clear()
                                drawerState.close()
                            }
                        },
                    )
                }
            },
        ) {
            Scaffold(
                topBar = {
                    TopAppBar(
                        title = { Text(tabNavigator.current.options.title) },
                        navigationIcon = {
                            IconButton(onClick = { scope.launch { drawerState.open() } }) {
                                Icon(Icons.Default.Menu, contentDescription = "Bo'limlar")
                            }
                        },
                    )
                },
            ) { padding ->
                Box(Modifier.fillMaxSize().padding(padding)) {
                    CurrentTab()
                }
            }
        }
    }
}

/**
 * Menyudagi bo'limlar — YAGONA ro'yxat.
 *
 * Ilgari tartib `RootScreen` da qo'lda yozilar, `TabOptions.index` esa
 * alohida turardi. Ikki manba vaqt o'tib ajralib ketardi.
 */
private val sections: List<Tab> = listOf(ChatTab, CodesTab, LawsTab)

@Composable
private fun DrawerHeader() {
    Row(
        Modifier.fillMaxWidth().padding(horizontal = 20.dp, vertical = 18.dp),
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(12.dp),
    ) {
        Image(
            painter = painterResource(R.drawable.logo_mark),
            contentDescription = null,
            modifier = Modifier.size(40.dp),
        )
        Column {
            Text("Deklarant AI", style = MaterialTheme.typography.titleMedium)
            Text(
                text = "Bojxona yordamchisi",
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
        }
    }
}

@Composable
private fun DrawerItem(tab: Tab, selected: Boolean, onClick: () -> Unit) {
    val options = tab.options
    NavigationDrawerItem(
        label = { Text(options.title) },
        selected = selected,
        onClick = onClick,
        icon = {
            options.icon?.let { Icon(painter = it, contentDescription = null) }
        },
        modifier = Modifier.padding(NavigationDrawerItemDefaults.ItemPadding),
    )
}

/**
 * Kim kirgani va kunlik chegara.
 *
 * NEGA MENYUDA: ilgari bu blok chat ekranining yuqorisida, rejim
 * tugmalari bilan BIR QATORDA turardi — telefon enida ism, rol, kvota
 * va ikkita tugma sig'masdi va matn qirqilardi. Menyuda esa eni
 * yetarli va u har doim ko'rinib turishi shart emas.
 *
 * Rol va kvota KO'RSATILADI: javob uslubi rolga qarab tanlanadi va
 * kvota ham rolga bog'liq. Foydalanuvchi buni bilmasa, uslub
 * "o'z-o'zidan" o'zgargandek tuyulardi.
 */
@Composable
private fun AccountBlock(user: User?, onLogin: () -> Unit, onLogout: () -> Unit) {
    if (user == null) {
        NavigationDrawerItem(
            label = { Text("Kirish") },
            selected = false,
            onClick = onLogin,
            icon = { Icon(Icons.AutoMirrored.Filled.Login, contentDescription = null) },
            modifier = Modifier.padding(NavigationDrawerItemDefaults.ItemPadding),
        )
        return
    }

    Row(
        Modifier.fillMaxWidth().padding(start = 28.dp, end = 12.dp, top = 4.dp, bottom = 12.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Column(Modifier.weight(1f)) {
            Text(
                text = user.name.ifBlank { user.login },
                style = MaterialTheme.typography.bodyMedium,
                maxLines = 1,
            )
            Text(
                text = "${user.role.label} · kuniga ${user.dailyQuota}",
                style = MaterialTheme.typography.labelSmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
                maxLines = 1,
            )
        }
        IconButton(onClick = onLogout) {
            Icon(
                imageVector = Icons.AutoMirrored.Filled.Logout,
                contentDescription = "Chiqish",
                tint = MaterialTheme.colorScheme.onSurfaceVariant,
            )
        }
    }
}
