import { useEffect, useRef, useState } from 'react'
import { auth, type Role, type RoleInfo, type User } from '../api'

interface Props {
  onDone: (u: User) => void
  onClose: () => void
}

type Tab = 'login' | 'register'

/**
 * Kirish va ro'yxatdan o'tish — MODAL oyna.
 *
 * NEGA SAHIFA EMAS: kirish alohida sahifa bo'lganda chat butunlay
 * yo'qolardi va foydalanuvchi "ilova nima qila oladi" degan savolga
 * javob ololmasdi. Modal esa suhbat oynasini joyida qoldiradi —
 * kirish shunchaki oldida turadi va yopilsa hech narsa yo'qolmaydi.
 */
export default function LoginDialog({ onDone, onClose }: Props) {
  const [tab, setTab] = useState<Tab>('login')
  const [login, setLogin] = useState('')
  const [password, setPassword] = useState('')
  const [name, setName] = useState('')
  const [role, setRole] = useState<Role>('DECLARANT')
  const [roles, setRoles] = useState<RoleInfo[]>([])
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')

  const firstField = useRef<HTMLInputElement>(null)

  // Rollar ro'yxati serverdan: kvota va uslub u yerda belgilanadi,
  // ikki joyda takrorlansa ular ajralib ketardi.
  useEffect(() => {
    auth.roles().then((r) => setRoles(r.roles)).catch(() => setRoles([]))
  }, [])

  // Modal ochilganda fokus ichkariga; Esc yopadi.
  useEffect(() => {
    firstField.current?.focus()
    const onKey = (e: KeyboardEvent) => { if (e.key === 'Escape') onClose() }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [onClose])

  async function submit(e: React.FormEvent) {
    e.preventDefault()
    setBusy(true)
    setError('')
    try {
      const s = tab === 'login'
        ? await auth.login({ login, password })
        : await auth.register({ login, password, name, role })
      onDone(s.user)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Xatolik')
    } finally {
      setBusy(false)
    }
  }

  const selectable = roles.filter((r) => r.self_signup)

  return (
    <div className="modal-scrim" onClick={onClose}>
      <div
        className="modal"
        role="dialog"
        aria-modal="true"
        aria-labelledby="login-title"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="modal-head">
          <h2 id="login-title">{tab === 'login' ? 'Kirish' : 'Ro\'yxatdan o\'tish'}</h2>
          <button className="modal-x" onClick={onClose} aria-label="Yopish">✕</button>
        </div>

        <p className="modal-lead">
          Chat uchun kirish kerak. Qidiruv, kalkulyator, risk va qonunlar
          kirishsiz ham ochiq.
        </p>

        <div className="seg" role="tablist">
          <button role="tab" aria-selected={tab === 'login'} onClick={() => setTab('login')}>Kirish</button>
          <button role="tab" aria-selected={tab === 'register'} onClick={() => setTab('register')}>Ro'yxatdan o'tish</button>
        </div>

        <form className="form" onSubmit={submit}>
          <label className="field">
            <span className="field-label">Telefon yoki e-pochta<b aria-hidden="true"> *</b></span>
            <input
              ref={firstField}
              value={login}
              onChange={(e) => setLogin(e.target.value)}
              placeholder="+998 90 123-45-67"
              autoComplete="username"
              required
            />
            <span className="field-hint">
              Bo'shliq va chiziqchalar hisobga olinmaydi
            </span>
          </label>

          <label className="field">
            <span className="field-label">Parol<b aria-hidden="true"> *</b></span>
            <input
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              autoComplete={tab === 'login' ? 'current-password' : 'new-password'}
              minLength={8}
              required
            />
            {tab === 'register' && <span className="field-hint">Kamida 8 belgi</span>}
          </label>

          {tab === 'register' && (
            <>
              <label className="field">
                <span className="field-label">Ism</span>
                <input value={name} onChange={(e) => setName(e.target.value)} autoComplete="name" />
              </label>

              <fieldset className="roles">
                <legend className="field-label">Rol</legend>
                {selectable.map((r) => (
                  <label key={r.role} className={'role-opt' + (role === r.role ? ' on' : '')}>
                    <input
                      type="radio"
                      name="role"
                      value={r.role}
                      checked={role === r.role}
                      onChange={() => setRole(r.role)}
                    />
                    <span className="role-name">{ROLE_LABEL[r.role]}</span>
                    <span className="role-desc">{ROLE_DESC[r.role]}</span>
                    <span className="role-meta">
                      javob: {r.chat_mode} · kuniga {r.daily_quota} savol
                    </span>
                  </label>
                ))}
                <p className="field-hint">
                  ADMIN roli bu yerda berilmaydi — uni mavjud admin tayinlaydi.
                </p>
              </fieldset>
            </>
          )}

          {error && <div className="composer-error">{error}</div>}

          <button type="submit" disabled={busy || !login.trim() || password.length < 8}>
            {busy ? '…' : tab === 'login' ? 'Kirish' : 'Ro\'yxatdan o\'tish'}
          </button>
        </form>
      </div>
    </div>
  )
}

const ROLE_LABEL: Record<Role, string> = {
  DECLARANT: 'Deklarant',
  BUSINESS: 'Tadbirkor',
  INSPECTOR: 'Inspektor',
  ADMIN: 'Administrator',
}

const ROLE_DESC: Record<Role, string> = {
  DECLARANT: 'TIF TN va GTD grafalarini bilaman — qisqa, kod va modda raqami bilan',
  BUSINESS: 'Atamalarni bilmayman — "qancha turadi va nima qilishim kerak"',
  INSPECTOR: 'Bojxona xodimi — deklarant bilan bir xil javob, kattaroq chegara',
  ADMIN: 'Statistika va sozlamalar paneli',
}
