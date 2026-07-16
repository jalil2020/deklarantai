import { useEffect, useState } from 'react'
import './App.css'
import { api, type HSMatch } from './api'
import HSCodeSearch from './components/HSCodeSearch'
import DutyCalculator, { type Prefill } from './components/DutyCalculator'
import Chat from './components/Chat'

type Tab = 'hscode' | 'duty' | 'chat'

function App() {
  const [tab, setTab] = useState<Tab>('hscode')
  const [aiAvailable, setAiAvailable] = useState(false)
  const [codeCount, setCodeCount] = useState(0)
  const [prefill, setPrefill] = useState<Prefill | null>(null)

  useEffect(() => {
    api.health()
      .then((h) => { setAiAvailable(h.ai_available); setCodeCount(h.codes) })
      .catch(() => { /* backend hali tayyor emas */ })
  }, [])

  function useCodeInCalculator(m: HSMatch) {
    setPrefill({
      import_duty: m.code.import_duty,
      excise: m.code.excise,
      vat: m.code.vat,
      name: m.code.name,
    })
    setTab('duty')
  }

  return (
    <div className="app">
      <header className="header">
        <div className="brand">
          <span className="logo">🛃</span>
          <div>
            <h1>Deklarant AI</h1>
            <p>O'zbekiston bojxona yordamchisi</p>
          </div>
        </div>
        <div className="status">
          <span className="badge">{codeCount} TIF TN kod</span>
          <span className={`badge ${aiAvailable ? 'on' : 'off'}`}>
            AI {aiAvailable ? 'yoqilgan' : "o'chirilgan"}
          </span>
        </div>
      </header>

      <nav className="tabs">
        <button className={tab === 'hscode' ? 'active' : ''} onClick={() => setTab('hscode')}>
          🔎 HS kod
        </button>
        <button className={tab === 'duty' ? 'active' : ''} onClick={() => setTab('duty')}>
          🧮 Kalkulyator
        </button>
        <button className={tab === 'chat' ? 'active' : ''} onClick={() => setTab('chat')}>
          💬 Chat
        </button>
      </nav>

      <main className="content">
        {tab === 'hscode' && (
          <HSCodeSearch aiAvailable={aiAvailable} onUseCode={useCodeInCalculator} />
        )}
        {tab === 'duty' && <DutyCalculator prefill={prefill} />}
        {tab === 'chat' && <Chat aiAvailable={aiAvailable} />}
      </main>

      <footer className="footer">
        Deklarant AI — demo versiya. Stavkalar va hisob-kitoblar taxminiy.
      </footer>
    </div>
  )
}

export default App
