import { useEffect, useState } from 'react'
import './App.css'
import { api } from './api'
import Chat from './components/Chat'

function App() {
  const [aiAvailable, setAiAvailable] = useState(false)
  const [codeCount, setCodeCount] = useState(0)

  useEffect(() => {
    api.health()
      .then((h) => { setAiAvailable(h.ai_available); setCodeCount(h.codes) })
      .catch(() => { /* backend hali tayyor emas */ })
  }, [])

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

      <main className="content">
        <Chat aiAvailable={aiAvailable} />
      </main>

      <footer className="footer">
        Deklarant AI — demo versiya. Stavkalar va hisob-kitoblar taxminiy.
      </footer>
    </div>
  )
}

export default App
