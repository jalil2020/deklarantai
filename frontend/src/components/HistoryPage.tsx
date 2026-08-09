import type { ChatMessage } from '../api'
import { clearChats, deleteChat, formatWhen, useChats } from '../store'

interface Props {
  /** Suhbatni ochish — chat bo'limiga o'tadi. */
  onOpen: (id: string, messages: ChatMessage[]) => void
}

export default function HistoryPage({ onOpen }: Props) {
  const chats = useChats()

  return (
    <div className="page">
      <div className="page-head">
        <h1>Tarixcha</h1>
        {chats.length > 0 && (
          <button className="ghost" onClick={() => {
            if (confirm('Barcha suhbatlar o\'chirilsinmi?')) clearChats()
          }}>Hammasini o'chirish</button>
        )}
      </div>

      <LocalNote />

      {chats.length === 0 ? (
        <div className="empty">Hozircha suhbat yo'q. Chat bo'limida savol bering.</div>
      ) : (
        <div className="results">
          {chats.map((c) => (
            <article key={c.id} className="card row-card">
              <button className="row-main" onClick={() => onOpen(c.id, c.messages)}>
                <span className="row-title">{c.title}</span>
                <span className="row-meta">
                  {formatWhen(c.at)} · {c.messages.length} xabar
                </span>
              </button>
              <button
                className="row-x"
                onClick={() => deleteChat(c.id)}
                aria-label="O'chirish"
                title="O'chirish"
              >✕</button>
            </article>
          ))}
        </div>
      )}
    </div>
  )
}

/**
 * Saqlash joyi haqidagi ogohlantirish.
 *
 * Foydalanuvchi tarixcha SERVERDA turadi deb o'ylab qolmasligi kerak:
 * brauzer tozalansa yozuvlar yo'qoladi va bu kutilmagan yo'qotish
 * bo'lardi.
 */
export function LocalNote() {
  return (
    <p className="local-note">
      Yozuvlar <b>shu brauzerda</b> saqlanadi — boshqa qurilmada
      ko'rinmaydi va brauzer ma'lumoti tozalansa yo'qoladi. Akkauntlar
      qo'shilganda ular serverga ko'chiriladi.
    </p>
  )
}
