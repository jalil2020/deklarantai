import { useEffect, useState } from 'react'
import './App.css'
import { api } from './api'
import Chat from './components/Chat'

function App() {
  const [aiAvailable, setAiAvailable] = useState(false)

  useEffect(() => {
    api.health()
      .then((h) => setAiAvailable(h.ai_available))
      .catch(() => { /* backend hali tayyor emas */ })
  }, [])

  return (
    <div className="app">
      <Chat aiAvailable={aiAvailable} />
    </div>
  )
}

export default App
