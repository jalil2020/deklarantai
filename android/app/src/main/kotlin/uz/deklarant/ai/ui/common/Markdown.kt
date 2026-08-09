package uz.deklarant.ai.ui.common

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.remember
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.AnnotatedString
import androidx.compose.ui.text.LinkAnnotation
import androidx.compose.ui.text.SpanStyle
import androidx.compose.ui.text.TextLinkStyles
import androidx.compose.ui.text.buildAnnotatedString
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontStyle
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.text.style.TextDecoration
import androidx.compose.ui.text.withLink
import androidx.compose.ui.unit.dp

/**
 * Kichik Markdown chizuvchi.
 *
 * NEGA O'ZIMIZNIKI: modelning javobi deyarli doim JADVAL bilan keladi
 * (boj, QQS, yig'im — GTD kodlari bo'yicha), ilova esa uni oddiy `Text`
 * bilan chizardi va ekranda xom `| Kod | To'lov |` hamda `**JAMI**`
 * ko'rinardi.
 *
 * Tayyor kutubxona olinmadi: mavjudlarining ko'pchiligi jadvalni umuman
 * chizmaydi yoki HTML/WebView orqali chizadi. Bizga esa aynan jadval
 * kerak, qolgani — bir nechta oddiy element. Webda ham shu sabab bilan
 * o'z chizuvchisi bor (`frontend/src/components/Markdown.tsx`) va bu
 * ikkalasi BIR XIL to'plamni qo'llab-quvvatlaydi.
 *
 * Qo'llab-quvvatlanadi: sarlavha, jadval, ro'yxat (belgili/raqamli),
 * **qalin**, *kursiv*, `kod`, havola, --- ajratgich, > iqtibos.
 */
@Composable
fun Markdown(
    text: String,
    color: Color,
    modifier: Modifier = Modifier,
) {
    // Tahlil qimmat emas, lekin oqim kelayotganda javob har bo'lakda
    // qayta chiziladi — o'zgarmagan matnni qayta tahlil qilmaymiz.
    val blocks = remember(text) { parseBlocks(text.replace("\r\n", "\n")) }

    Column(modifier, verticalArrangement = Arrangement.spacedBy(6.dp)) {
        blocks.forEach { block ->
            when (block) {
                is Block.Heading -> Text(
                    text = inline(block.text, color),
                    style = when (block.level) {
                        1 -> MaterialTheme.typography.titleMedium
                        2 -> MaterialTheme.typography.titleSmall
                        else -> MaterialTheme.typography.bodyLarge
                    },
                    fontWeight = FontWeight.SemiBold,
                    color = color,
                )

                is Block.Paragraph -> Text(
                    text = inline(block.text, color),
                    style = MaterialTheme.typography.bodyMedium,
                    color = color,
                )

                is Block.Bullet -> Column(verticalArrangement = Arrangement.spacedBy(2.dp)) {
                    block.items.forEachIndexed { i, item ->
                        Row {
                            Text(
                                text = if (block.ordered) "${block.start + i}. " else "• ",
                                style = MaterialTheme.typography.bodyMedium,
                                color = color,
                            )
                            Text(
                                text = inline(item, color),
                                style = MaterialTheme.typography.bodyMedium,
                                color = color,
                            )
                        }
                    }
                }

                is Block.Quote -> Row {
                    // Chap chiziq — iqtibosni matndan ajratadi.
                    Column(
                        Modifier
                            .padding(end = 8.dp)
                            .width(3.dp)
                            .background(color.copy(alpha = 0.35f)),
                    ) { Text("") }
                    Text(
                        text = inline(block.text, color),
                        style = MaterialTheme.typography.bodyMedium,
                        color = color.copy(alpha = 0.85f),
                    )
                }

                Block.Divider -> HorizontalDivider(color = color.copy(alpha = 0.25f))

                is Block.Table -> TableBlock(block, color)
            }
        }
    }
}

@Composable
private fun TableBlock(table: Block.Table, color: Color) {
    // USTUN ENLARI MAZMUNGA QARAB.
    //
    // Teng ulush berilsa, "GTD" yoki "10" kabi tor ustun keng
    // summalar bilan bir xil joy olardi va raqamlar har qatorda
    // ikkiga bo'linib ketardi. Uzunlik chegaralanadi: bitta juda
    // uzun izoh qolgan ustunlarni siqib qo'ymasin.
    val weights = remember(table) {
        val cols = table.header.size
        IntArray(cols) { c ->
            var longest = table.header.getOrElse(c) { "" }.length
            table.rows.forEach { r ->
                longest = maxOf(longest, r.getOrElse(c) { "" }.length)
            }
            longest.coerceIn(4, 18)
        }
    }

    Column(
        Modifier
            .fillMaxWidth()
            .background(color.copy(alpha = 0.05f)),
    ) {
        TableRow(table.header, table.aligns, weights, color, header = true)
        HorizontalDivider(color = color.copy(alpha = 0.3f))
        table.rows.forEachIndexed { i, row ->
            TableRow(row, table.aligns, weights, color, header = false)
            if (i < table.rows.lastIndex) {
                HorizontalDivider(color = color.copy(alpha = 0.12f))
            }
        }
    }
}

@Composable
private fun TableRow(
    cells: List<String>,
    aligns: List<TextAlign>,
    weights: IntArray,
    color: Color,
    header: Boolean,
) {
    Row(Modifier.fillMaxWidth().padding(vertical = 4.dp)) {
        weights.indices.forEach { c ->
            Text(
                text = inline(cells.getOrElse(c) { "" }, color),
                style = MaterialTheme.typography.labelMedium,
                fontWeight = if (header) FontWeight.SemiBold else FontWeight.Normal,
                color = color,
                textAlign = aligns.getOrElse(c) { TextAlign.Start },
                modifier = Modifier
                    .weight(weights[c].toFloat())
                    .padding(horizontal = 5.dp),
            )
        }
    }
}

// ---------------------------------------------------------- blok darajasi

private sealed interface Block {
    data class Heading(val level: Int, val text: String) : Block
    data class Paragraph(val text: String) : Block
    data class Bullet(val items: List<String>, val ordered: Boolean, val start: Int) : Block
    data class Quote(val text: String) : Block
    data class Table(
        val header: List<String>,
        val aligns: List<TextAlign>,
        val rows: List<List<String>>,
    ) : Block

    data object Divider : Block
}

private val headingRe = Regex("""^(#{1,6})\s+(.*)$""")
private val dividerRe = Regex("""^\s*([-*_])\s*\1\s*\1[\s\-*_]*$""")
private val bulletRe = Regex("""^\s*[-*•]\s+(.*)$""")
private val orderedRe = Regex("""^\s*(\d+)[.)]\s+(.*)$""")
private val quoteRe = Regex("""^\s*>\s?(.*)$""")
private val tableDividerRe = Regex("""^\s*\|?\s*:?-{2,}:?\s*(\|\s*:?-{2,}:?\s*)*\|?\s*$""")

private fun parseBlocks(src: String): List<Block> {
    val lines = src.split("\n")
    val out = mutableListOf<Block>()
    val para = mutableListOf<String>()

    fun flush() {
        if (para.isNotEmpty()) {
            out += Block.Paragraph(para.joinToString("\n"))
            para.clear()
        }
    }

    var i = 0
    while (i < lines.size) {
        val ln = lines[i]

        if (ln.isBlank()) {
            flush()
            i++
            continue
        }

        // Ajratgich TEKSHIRUVI jadvalnikidan OLDIN emas, keyin turishi
        // kerak edi — "---" jadval ajratgichi ham shu qolipga tushadi.
        // Shuning uchun jadval avval qaraladi.
        if (ln.contains('|') && i + 1 < lines.size && tableDividerRe.matches(lines[i + 1])) {
            flush()
            val header = cells(ln)
            val aligns = cells(lines[i + 1]).map(::alignOf)
            i += 2
            val rows = mutableListOf<List<String>>()
            while (i < lines.size && lines[i].contains('|') && lines[i].isNotBlank()) {
                rows += cells(lines[i])
                i++
            }
            out += Block.Table(header, aligns, rows)
            continue
        }

        if (dividerRe.matches(ln)) {
            flush()
            out += Block.Divider
            i++
            continue
        }

        val heading = headingRe.find(ln)
        if (heading != null) {
            flush()
            out += Block.Heading(heading.groupValues[1].length, heading.groupValues[2])
            i++
            continue
        }

        if (quoteRe.matches(ln)) {
            flush()
            val text = StringBuilder()
            while (i < lines.size && quoteRe.matches(lines[i])) {
                if (text.isNotEmpty()) text.append('\n')
                text.append(quoteRe.find(lines[i])!!.groupValues[1])
                i++
            }
            out += Block.Quote(text.toString())
            continue
        }

        if (bulletRe.matches(ln) || orderedRe.matches(ln)) {
            flush()
            val ordered = orderedRe.matches(ln)
            val start = if (ordered) orderedRe.find(ln)!!.groupValues[1].toIntOrNull() ?: 1 else 1
            val items = mutableListOf<String>()
            while (i < lines.size) {
                val cur = lines[i]
                val m = if (ordered) orderedRe.find(cur) else bulletRe.find(cur)
                if (m == null) {
                    // Davomi: qo'shimcha bo'shliq bilan boshlangan qator
                    // oldingi bandga qo'shiladi.
                    if (items.isNotEmpty() && cur.isNotBlank() && cur.startsWith("  ")) {
                        items[items.lastIndex] = items.last() + " " + cur.trim()
                        i++
                        continue
                    }
                    break
                }
                items += m.groupValues.last()
                i++
            }
            out += Block.Bullet(items, ordered, start)
            continue
        }

        para += ln
        i++
    }
    flush()
    return out
}

/** `| a | b |` → ["a", "b"]. Chetdagi bo'sh kataklar tashlanadi. */
private fun cells(line: String): List<String> {
    val parts = line.trim().split("|").toMutableList()
    if (parts.isNotEmpty() && parts.first().isBlank()) parts.removeAt(0)
    if (parts.isNotEmpty() && parts.last().isBlank()) parts.removeAt(parts.lastIndex)
    return parts.map { it.trim() }
}

/** `---:` → o'ngga, `:---:` → markazga, qolgani chapga. */
private fun alignOf(spec: String): TextAlign {
    val s = spec.trim()
    val left = s.startsWith(":")
    val right = s.endsWith(":")
    return when {
        left && right -> TextAlign.Center
        right -> TextAlign.End
        else -> TextAlign.Start
    }
}

// --------------------------------------------------------- ichki belgilar

private val inlineRe = Regex(
    """\*\*(.+?)\*\*|__(.+?)__|\*(.+?)\*|`([^`]+)`|\[([^\]]+)]\((https?://[^)\s]+)\)""",
)

/**
 * **qalin**, *kursiv*, `kod` va havolalarni bezaydi.
 *
 * Havola BOSILADIGAN: qonun parchalari lex.uz manzili bilan keladi va
 * foydalanuvchi rasmiy matnni ochib tekshira olishi kerak.
 */
private fun inline(text: String, color: Color): AnnotatedString = buildAnnotatedString {
    var last = 0
    for (m in inlineRe.findAll(text)) {
        append(text.substring(last, m.range.first))
        val g = m.groupValues
        when {
            g[1].isNotEmpty() || g[2].isNotEmpty() ->
                pushAndAppend(SpanStyle(fontWeight = FontWeight.Bold), g[1].ifEmpty { g[2] })

            g[3].isNotEmpty() -> pushAndAppend(SpanStyle(fontStyle = FontStyle.Italic), g[3])

            g[4].isNotEmpty() -> pushAndAppend(
                SpanStyle(fontFamily = FontFamily.Monospace, background = color.copy(alpha = 0.10f)),
                g[4],
            )

            g[5].isNotEmpty() -> withLink(
                LinkAnnotation.Url(
                    url = g[6],
                    styles = TextLinkStyles(
                        style = SpanStyle(
                            color = color,
                            textDecoration = TextDecoration.Underline,
                        ),
                    ),
                ),
            ) { append(g[5]) }
        }
        last = m.range.last + 1
    }
    append(text.substring(last))
}

private fun AnnotatedString.Builder.pushAndAppend(style: SpanStyle, text: String) {
    val at = pushStyle(style)
    append(text)
    pop(at)
}
