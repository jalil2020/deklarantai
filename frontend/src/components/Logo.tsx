/**
 * Deklarant AI belgisi.
 *
 * Rasm — dizayner bergan asl fayl, faqat matni kesilgan (`tools/logo`).
 * ATAYLAB vektorga ko'chirilmagan: qayta chizish belgini o'zgartirib
 * yuboradi.
 *
 * Fayl `public/` da — Vite uni o'zgartirmasdan nashr qiladi, shu tufayli
 * bir xil manzil favicon uchun ham, sahifa ichida ham ishlaydi.
 */
const SRC = '/logo.png'
const ALT = 'Deklarant AI'

/** Hero uchun katta belgi (suzuvchi animatsiya CSS da). */
export function LogoMark() {
  return <img className="logo-mark" src={SRC} alt={ALT} width={512} height={512} />
}

/** Header uchun kichik belgi + wordmark. */
export function MiniLogo() {
  return (
    <div className="mini-logo">
      <img className="mini-robot" src={SRC} alt="" width={512} height={512} />
      <span className="wordmark sm">
        Deklarant <span className="wm-a">AI</span>
      </span>
    </div>
  )
}
