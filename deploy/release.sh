#!/usr/bin/env bash
# Deklarant AI — yangi versiyani serverga chiqarish.
#
# MAHALLIY mashinada yurgiziladi (loyiha ildizidan):
#
#   bash deploy/release.sh
#
# Muhit o'zgaruvchilari bilan boshqariladi:
#   DEPLOY_HOST  — server (sukut: deklarantpro.uz, DNS tayyor bo'lmasa IP bering)
#   DEPLOY_USER  — sukut: root
#   SSH_KEY      — sukut: ~/.ssh/deklarant_deploy

set -euo pipefail

# ⚠️ Server manzili KODGA YOZILMAYDI. Ochiq repozitoriyda u
# foydalanuvchi nomi va kalit yo'li bilan birga tayyor hujum manzili
# bo'lardi. DNS tayyor bo'lmasa IP ni muhit orqali bering:
#	DEPLOY_HOST=1.2.3.4 bash deploy/release.sh
HOST=${DEPLOY_HOST:-deklarantpro.uz}
USER=${DEPLOY_USER:-root}
KEY=${SSH_KEY:-$HOME/.ssh/deklarant_deploy}
SSH=(ssh -i "$KEY" -o StrictHostKeyChecking=accept-new "$USER@$HOST")
SCP=(scp -i "$KEY" -o StrictHostKeyChecking=accept-new)

root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
cd "$root"

log() { printf '\n\033[1m==> %s\033[0m\n' "$1"; }

# ---------------------------------------------------------------- qurish

log "Backend (linux/amd64)"
# KROSS-KOMPILYATSIYA: serverga Go o'rnatish shart emas — Go buni
# o'zi qiladi va natija bitta statik fayl. CGO o'chirilgan, ya'ni
# serverdagi glibc versiyasiga bog'liqlik ham yo'q.
( cd backend && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
	go build -trimpath -ldflags="-s -w" -o /tmp/deklarant-linux . )
ls -lh /tmp/deklarant-linux | awk '{print "  hajmi:", $5}'

log "Frontend"
# ⚠️ `npm ci` ATAYLAB ishlatilmaydi. U node_modules ni butunlay
# o'chirib qayta yozadi va Windows'da bu EPERM bilan yiqiladi:
# rolldown/esbuild ning .node fayli tahrirlovchi yoki antivirus
# tomonidan band bo'lib qoladi.
#
# Bog'liqliklar faqat YO'Q bo'lsa o'rnatiladi; qurish esa har safar
# toza bajariladi.
( cd frontend
  # Papka BORLIGI yetarli emas: yiqilgan `npm ci` uni yarim o'chirib
  # ketishi mumkin (aynan shunday bo'ldi — node_modules qoldi, lekin
  # ichida bitta paket va bo'sh .bin). Shuning uchun ASBOBNING O'ZI
  # tekshiriladi.
  [ -x node_modules/.bin/vite ] || npm install --no-audit --no-fund
  npm run build
)

# ---------------------------------------------------------------- yuklash

log "Fayllarni yuborish"
"${SSH[@]}" 'mkdir -p /opt/deklarant/data /var/www/deklarant'

# Dastur avval VAQTINCHALIK nom bilan yuklanadi: ishlayotgan faylning
# ustiga to'g'ridan-to'g'ri yozib bo'lmaydi ("text file busy").
"${SCP[@]}" /tmp/deklarant-linux "$USER@$HOST:/opt/deklarant/deklarant.new"

# Ma'lumot fayllari — ~24 MB. Har chiqarishda yuboriladi, chunki
# stavkalar va qonun korpusi yangilanib turadi.
"${SCP[@]}" backend/data/hscodes.json backend/data/laws.json \
	backend/data/docs.json backend/data/countries.json \
	backend/data/taxonomy.json "$USER@$HOST:/opt/deklarant/data/"

# ⚠️ users.json YUBORILMAYDI: u serverda yasaladi va ro'yxatdan
# o'tgan odamlarning parol xeshlari bor. Mahalliy nusxa bilan
# almashtirilsa, hamma foydalanuvchi yo'qolardi.

log "Frontend fayllari"
"${SCP[@]}" -r frontend/dist/. "$USER@$HOST:/var/www/deklarant/"

log "Sozlamalar"
"${SCP[@]}" deploy/Caddyfile "$USER@$HOST:/etc/caddy/Caddyfile"
"${SCP[@]}" deploy/deklarant.service "$USER@$HOST:/etc/systemd/system/deklarant.service"

# ---------------------------------------------------------------- ishga tushirish

log "Xizmatni almashtirish"
"${SSH[@]}" bash -s <<'REMOTE'
set -euo pipefail

# Kalitsiz ishga tushirmaymiz — chat o'chiq holda "ishlayapti" deb
# ko'rinishi eng yomon holat.
if ! grep -q '^ANTHROPIC_API_KEY=.\+' /etc/deklarant.env; then
	echo "⚠️ TO'XTATILDI: /etc/deklarant.env da ANTHROPIC_API_KEY bo'sh"
	exit 1
fi

mv /opt/deklarant/deklarant.new /opt/deklarant/deklarant
chmod 755 /opt/deklarant/deklarant
chown -R deklarant:deklarant /opt/deklarant
chown -R caddy:caddy /var/www/deklarant

systemctl daemon-reload
systemctl enable --now deklarant >/dev/null
systemctl restart deklarant

caddy validate --config /etc/caddy/Caddyfile --adapter caddyfile >/dev/null
systemctl reload caddy || systemctl restart caddy

sleep 2
systemctl is-active --quiet deklarant || { journalctl -u deklarant -n 30 --no-pager; exit 1; }

echo "--- salomatlik ---"
curl -fsS --max-time 10 http://127.0.0.1:8080/api/health | head -c 300; echo
echo "--- modellar ---"
journalctl -u deklarant -n 40 --no-pager | grep -E 'Modellar|AI yoqilgan' | tail -2
REMOTE

log "Tayyor"
echo "  https://deklarantpro.uz  (DNS tayyor bo'lgach)"
echo "  Jurnal: ssh -i $KEY $USER@$HOST journalctl -u deklarant -f"
