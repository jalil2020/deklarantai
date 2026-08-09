import { useEffect, useState } from 'react'
import './App.css'
import { api, auth, type ChatMessage, type HSMatch, type User } from './api'
import LoginDialog from './components/LoginDialog'
import Chat from './components/Chat'
import CalcPage, { type CalcSeed } from './components/CalcPage'
import FavoritesPage from './components/FavoritesPage'
import HistoryPage from './components/HistoryPage'
import LawsPanel from './components/LawsPanel'
import Nav, { VIEWS, type View } from './components/Nav'
import RiskPage from './components/RiskPage'
import SearchPage from './components/SearchPage'

/**
 * Manzildan bo'limni o'qiydi.
 *
 * NEGA HASH: router kutubxonasi qo'shilmadi — bir necha bo'lim uchun u
 * ortiqcha. Lekin holat faqat `useState` da tursa, sahifa yangilanganda
 * bo'lim yo'qolardi va "orqaga" tugmasi ilovadan chiqarib yuborardi.
 */
function viewFromHash(): View {
  const v = location.hash.replace('#/', '') as View
  return VIEWS.includes(v) ? v : 'chat'
}

function App() {
  const [aiAvailable, setAiAvailable] = useState(false)
  const [view, setView] = useState<View>(viewFromHash)
  const [navOpen, setNavOpen] = useState(false)
  const [user, setUser] = useState<User | null>(null)
  // Kirish oynasi — MODAL. Bo'lim emas: chat joyida qoladi va
  // foydalanuvchi ilova nima qila olishini ko'rib turadi.
  const [loginOpen, setLoginOpen] = useState(false)

  // Chatga uzatiladigan matn. `at` — hodisa vaqti: bir xil kod ikki
  // marta tanlansa ham effekt qayta ishlashi uchun.
  const [inject, setInject] = useState<{ text: string; at: number }>()
  const [calcSeed, setCalcSeed] = useState<CalcSeed>()
  const [riskSeed, setRiskSeed] = useState<{ code: string; at: number }>()
  const [restore, setRestore] = useState<{ id: string; messages: ChatMessage[]; at: number }>()

  useEffect(() => {
    api.health()
      .then((h) => setAiAvailable(h.ai_available))
      .catch(() => { /* backend hali tayyor emas */ })
    // Saqlangan token amaldami — sahifa yangilanganda qayta kirmasin.
    auth.me().then(setUser).catch(() => setUser(null))
  }, [])

  useEffect(() => {
    const onHash = () => setView(viewFromHash())
    window.addEventListener('hashchange', onHash)
    return () => window.removeEventListener('hashchange', onHash)
  }, [])

  const go = (v: View) => {
    location.hash = '#/' + v
    setView(v)
    setNavOpen(false)
  }

  /** Kod bo'yicha tayyor savol — foydalanuvchi qiymat va davlatni qo'shsa bo'ldi. */
  const ask = (code: string, title: string) => {
    setInject({ text: `${code} (${title}) bo'yicha bojni hisobla. Qiymati: `, at: Date.now() })
    go('chat')
  }

  const askText = (text: string) => {
    setInject({ text, at: Date.now() })
    go('chat')
  }

  /** Stavkalar qo'lda ko'chirilmasin — ko'chirishda xato qilinadi. */
  const calc = (m: HSMatch) => {
    setCalcSeed({
      code: m.code.code,
      title: m.code.name_uz || m.code.name_ru,
      importDuty: m.code.import_duty,
      exportDuty: m.code.export_duty,
      vat: m.code.vat,
      specific: m.code.import_duty_specific,
      specificUnit: m.code.import_duty_specific_unit,
      unit: m.code.unit,
      at: Date.now(),
    })
    go('calc')
  }

  const risk = (code: string) => {
    setRiskSeed({ code, at: Date.now() })
    go('risk')
  }

  return (
    <div className="app">
      <Nav
        view={view}
        onGo={go}
        open={navOpen}
        onClose={() => setNavOpen(false)}
        user={user}
        onLogin={() => setLoginOpen(true)}
        onLogout={() => { void auth.logout().then(() => setUser(null)) }}
      />

      {loginOpen && (
        <LoginDialog
          onDone={(u) => { setUser(u); setLoginOpen(false) }}
          onClose={() => setLoginOpen(false)}
        />
      )}

      <button
        className="nav-open"
        onClick={() => setNavOpen(true)}
        aria-label="Menyu"
      >☰</button>

      <main className="main">
        {/* Chat HAR DOIM chiziladi. Kirish talab qilinsa, maydon
            yopiladi va kirish oynasi ustidan ochiladi — ilova nima
            qila olishini ko'rmasdan kirishga majburlash noto'g'ri. */}
        {view === 'chat' && (
          <Chat
            aiAvailable={aiAvailable}
            inject={inject}
            restore={restore}
            signedIn={user !== null}
            onNeedLogin={() => setLoginOpen(true)}
          />
        )}
        {view === 'search' && <SearchPage onCalc={calc} onAsk={ask} onRisk={risk} />}
        {view === 'calc' && <CalcPage seed={calcSeed} />}
        {view === 'risk' && <RiskPage seed={riskSeed} onAsk={askText} />}
        {view === 'laws' && (
          <div className="page">
            <div className="page-head"><h1>Qonunchilik</h1></div>
            <LawsPanel onAsk={askText} />
          </div>
        )}
        {view === 'history' && (
          <HistoryPage onOpen={(id, messages) => {
            setRestore({ id, messages, at: Date.now() })
            go('chat')
          }} />
        )}
        {view === 'favorites' && (
          <FavoritesPage onCode={(code) => risk(code)} onAsk={askText} />
        )}
      </main>
    </div>
  )
}

export default App
