import type { Role, User } from '../api'
import { MiniLogo } from './Logo'

/**
 * Ilova bo'limlari.
 *
 * Ikki guruh: ISH (savol berish, topish, hisoblash, tekshirish) va
 * SAQLANGAN (o'tgan suhbatlar, belgilangan kodlar). Ajratish kerak,
 * chunki ikkinchi guruh boshqacha ishlaydi — u brauzerda saqlanadi va
 * qurilmaga bog'langan.
 */
export const VIEWS = [
  'chat', 'search', 'calc', 'risk', 'laws', 'history', 'favorites',
] as const
type NavView = (typeof VIEWS)[number]

export type View = NavView

const LABELS: Record<NavView, { icon: string; label: string; hint: string }> = {
  chat: { icon: '💬', label: 'Chat', hint: 'AI bilan suhbat' },
  search: { icon: '🔎', label: 'Qidiruv', hint: 'TIF TN kodini topish' },
  calc: { icon: '🧮', label: 'Kalkulyator', hint: 'Boj va yig\'imlar' },
  risk: { icon: '🛡️', label: 'Risk baholash', hint: 'Ruxsatnoma va xavflar' },
  laws: { icon: '📖', label: 'Qonunlar', hint: 'Kodekslar va qarorlar' },
  history: { icon: '🕘', label: 'Tarixcha', hint: 'O\'tgan suhbatlar' },
  favorites: { icon: '☆', label: 'Sevimlilar', hint: 'Saqlangan kod va moddalar' },
}

/** Ikkinchi guruh shu banddan boshlanadi. */
const SAVED: NavView[] = ['history', 'favorites']

/** Rol nomlari — menyudagi chipda ko'rsatiladi. */
const ROLE_SHORT: Record<Role, string> = {
  DECLARANT: 'Deklarant',
  BUSINESS: 'Tadbirkor',
  INSPECTOR: 'Inspektor',
  ADMIN: 'Administrator',
}

interface Props {
  view: View
  onGo: (v: View) => void
  /** Mobil qurilmada menyu ustma-ust ochiladi. */
  open: boolean
  onClose: () => void
  user: User | null
  onLogin: () => void
  onLogout: () => void
}

export default function Nav({ view, onGo, open, onClose, user, onLogin, onLogout }: Props) {
  const work = VIEWS.filter((v) => !SAVED.includes(v))

  return (
    <>
      {open && <div className="nav-scrim" onClick={onClose} />}

      <nav className={'nav' + (open ? ' open' : '')} aria-label="Bo'limlar">
        <div className="nav-brand">
          <MiniLogo />
        </div>

        <ul className="nav-list">
          {work.map((v) => <Item key={v} v={v} view={view} onGo={onGo} />)}

          <li className="nav-sep" aria-hidden="true">Saqlangan</li>
          {SAVED.map((v) => <Item key={v} v={v} view={view} onGo={onGo} />)}
        </ul>

        <div className="nav-foot">
          {user ? (
            <div className="who">
              <div className="who-main">
                <span className="who-name">{user.name || user.login}</span>
                <span className="who-role">
                  {ROLE_SHORT[user.role]} · kuniga {user.daily_quota}
                </span>
              </div>
              <button className="who-x" onClick={onLogout} title="Chiqish" aria-label="Chiqish">⏻</button>
            </div>
          ) : (
            <button className="who-in" onClick={onLogin}>Kirish</button>
          )}

          {/* Baza holati — foydalanuvchi qaysi ma'lumot bilan
              ishlayotganini bilishi kerak. Stavkalar sanasi javob
              to'g'riligini belgilaydi. */}
          <span className="nav-base">TIF TN 2025 · ПКМ 181</span>
        </div>
      </nav>
    </>
  )
}

function Item({ v, view, onGo }: { v: NavView; view: View; onGo: (v: View) => void }) {
  return (
    <li>
      <button
        className={'nav-item' + (view === v ? ' active' : '')}
        onClick={() => onGo(v)}
        aria-current={view === v ? 'page' : undefined}
        title={LABELS[v].hint}
      >
        <span className="nav-icon" aria-hidden="true">{LABELS[v].icon}</span>
        <span className="nav-label">{LABELS[v].label}</span>
      </button>
    </li>
  )
}
