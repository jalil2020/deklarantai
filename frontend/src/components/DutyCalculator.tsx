import { useEffect, useState } from 'react'
import { api, formatSom, type DutyResult } from '../api'

export interface Prefill {
  import_duty: number
  excise: number
  vat: number
  name?: string
}

interface Props {
  prefill?: Prefill | null
}

export default function DutyCalculator({ prefill }: Props) {
  const [customsValue, setCustomsValue] = useState('')
  const [importDuty, setImportDuty] = useState('0')
  const [excise, setExcise] = useState('0')
  const [vat, setVat] = useState('12')
  const [quantity, setQuantity] = useState('1')
  const [result, setResult] = useState<DutyResult | null>(null)
  const [error, setError] = useState('')
  const [note, setNote] = useState('')

  // HS qidiruvdan stavkalar kelsa, avtomatik to'ldiramiz.
  useEffect(() => {
    if (prefill) {
      setImportDuty(String(prefill.import_duty))
      setExcise(String(prefill.excise))
      setVat(String(prefill.vat))
      setNote(prefill.name ? `Stavkalar "${prefill.name}" uchun yuklandi` : '')
    }
  }, [prefill])

  async function calculate(e: React.FormEvent) {
    e.preventDefault()
    setError('')
    const cv = parseFloat(customsValue)
    if (isNaN(cv) || cv <= 0) {
      setError('Bojxona qiymatini kiriting')
      return
    }
    try {
      const res = await api.calculateDuty({
        customs_value: cv,
        import_duty: parseFloat(importDuty) || 0,
        excise: parseFloat(excise) || 0,
        vat: parseFloat(vat) || 0,
        quantity: parseFloat(quantity) || 1,
      })
      setResult(res)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Xatolik')
    }
  }

  return (
    <div className="panel">
      <h2>Bojxona to'lovlari kalkulyatori</h2>
      <p className="hint">Bojxona qiymati va stavkalarni kiriting — jami to'lovni hisoblaymiz.</p>

      {note && <div className="ai-comment"><p>{note}</p></div>}

      <form onSubmit={calculate} className="calc-form">
        <div className="field">
          <label>Bojxona qiymati (so'm)</label>
          <input type="number" value={customsValue} min="0"
            onChange={(e) => setCustomsValue(e.target.value)} placeholder="10000000" />
        </div>
        <div className="field-row">
          <div className="field">
            <label>Import boji (%)</label>
            <input type="number" value={importDuty} min="0"
              onChange={(e) => setImportDuty(e.target.value)} />
          </div>
          <div className="field">
            <label>Aksiz (%)</label>
            <input type="number" value={excise} min="0"
              onChange={(e) => setExcise(e.target.value)} />
          </div>
          <div className="field">
            <label>QQS (%)</label>
            <input type="number" value={vat} min="0"
              onChange={(e) => setVat(e.target.value)} />
          </div>
          <div className="field">
            <label>Miqdor</label>
            <input type="number" value={quantity} min="0"
              onChange={(e) => setQuantity(e.target.value)} />
          </div>
        </div>
        <button type="submit">Hisoblash</button>
      </form>

      {error && <div className="error">{error}</div>}

      {result && (
        <div className="calc-result">
          <table>
            <thead>
              <tr><th>To'lov turi</th><th>Stavka</th><th>Baza</th><th>Summa</th></tr>
            </thead>
            <tbody>
              {result.items.map((it) => (
                <tr key={it.name}>
                  <td>{it.name}</td>
                  <td>{it.rate ? it.rate + '%' : '—'}</td>
                  <td>{formatSom(it.base)}</td>
                  <td>{formatSom(it.amount)}</td>
                </tr>
              ))}
            </tbody>
            <tfoot>
              <tr><td colSpan={3}><b>Jami to'lov</b></td><td><b>{formatSom(result.total)}</b></td></tr>
            </tfoot>
          </table>
        </div>
      )}

      <p className="disclaimer">
        ⚠️ Hisob-kitob taxminiy. Bojxona yig'imi shartli qat'iy summa sifatida olingan.
        Aniq to'lovlar uchun bojxona brokeriga murojaat qiling.
      </p>
    </div>
  )
}
