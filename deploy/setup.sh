#!/usr/bin/env bash
# Deklarant AI — serverni BIR MARTA sozlash (Ubuntu/Debian).
#
# Serverda root sifatida bir marta yurgiziladi. Qayta yurgizish
# xavfsiz: hamma qadam mavjud holatni tekshiradi.
#
#   bash setup.sh
#
# Dastur va ma'lumot fayllari `release.sh` bilan alohida yuklanadi —
# bu skript faqat MUHITNI tayyorlaydi.

set -euo pipefail

APP_USER=deklarant
APP_DIR=/opt/deklarant
WEB_DIR=/var/www/deklarant
ENV_FILE=/etc/deklarant.env

log() { printf '\n\033[1m==> %s\033[0m\n' "$1"; }

[ "$(id -u)" -eq 0 ] || { echo "root sifatida yurgizing"; exit 1; }

log "Paketlar"
export DEBIAN_FRONTEND=noninteractive
apt-get update -qq
apt-get install -y -qq ca-certificates curl gnupg ufw

log "Swap"
# ⚠️ 512 MB li serverda swap SHART. O'lchangan: backend bazalarni
# yuklagach ~133 MB oladi, Caddy ~30 MB, Ubuntu ~110 MB. Zaxira
# ozgina qoladi va 1,6 MB invoys rasmi kelganda (base64 dekodlash)
# tizim OOM-killer'ga borishi mumkin. Swap bilan bu sekinlashishga
# aylanadi — xizmat o'lmaydi.
#
# DigitalOcean droplet'larida swap sukut bo'yicha YO'Q.
if ! swapon --show=NAME --noheadings | grep -q .; then
	fallocate -l 1G /swapfile
	chmod 600 /swapfile
	mkswap /swapfile >/dev/null
	swapon /swapfile
	grep -q '^/swapfile' /etc/fstab || echo '/swapfile none swap sw 0 0' >> /etc/fstab
	# Swap'ga faqat CHINDAN kerak bo'lganda tushsin — disk sekin.
	sysctl -qw vm.swappiness=10
	grep -q '^vm.swappiness' /etc/sysctl.conf || echo 'vm.swappiness=10' >> /etc/sysctl.conf
	echo "1 GB swap yoqildi"
else
	echo "swap allaqachon bor"
fi

log "Caddy"
if ! command -v caddy >/dev/null; then
	# Rasmiy Caddy omboridan (Cloudsmith) — imzo kaliti bilan.
	curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/gpg.key' \
		| gpg --dearmor -o /usr/share/keyrings/caddy-stable-archive-keyring.gpg
	curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/debian.deb.txt' \
		| tee /etc/apt/sources.list.d/caddy-stable.list >/dev/null
	apt-get update -qq
	apt-get install -y -qq caddy
else
	echo "Caddy allaqachon o'rnatilgan"
fi

log "Foydalanuvchi va papkalar"
id -u "$APP_USER" >/dev/null 2>&1 || useradd --system --home "$APP_DIR" --shell /usr/sbin/nologin "$APP_USER"
mkdir -p "$APP_DIR/data" "$WEB_DIR" /var/log/caddy
chown -R "$APP_USER:$APP_USER" "$APP_DIR"
chown -R caddy:caddy /var/log/caddy

log "Muhit fayli"
if [ ! -f "$ENV_FILE" ]; then
	# Sirlar SHU YERDA yasaladi — hech qachon repozitoriyda emas.
	# CLIENT_TOKEN_SECRET tasodifiy: berilmasa har qayta ishga
	# tushishda o'zgarib, hamma tokenni kuchsizlantirardi.
	cat > "$ENV_FILE" <<EOF
# Deklarant AI — ishlab chiqarish sozlamalari
# ⚠️ BU FAYLNI HECH QAYERGA NUSXA QILMANG.

# --- MAJBURIY: kalitni QO'LDA yozing ---
ANTHROPIC_API_KEY=

# --- Avtomatik yasalgan sirlar ---
CLIENT_TOKEN_SECRET=$(head -c 32 /dev/urandom | base64 -w0)
ADMIN_PASSWORD=$(head -c 18 /dev/urandom | base64 -w0)

# --- Sozlamalar ---
PORT=8080
# ⚠️ Go uyum chegarasi. Busiz GC "xotira ko'p" deb o'ylab kechikadi va
# 458 MB li mashinada OOM-killer xizmatni o'ldirishi mumkin. Chegara
# bilan GC qattiqroq ishlaydi — sekinlashish bo'ladi, o'lim emas.
# O'lchangan: yuklangandan keyin ~133 MB.
GOMEMLIMIT=250MiB
ALLOWED_ORIGINS=https://deklarantpro.uz,https://www.deklarantpro.uz
ANTHROPIC_CACHE_TTL=1h
TRUST_PROXY=1
HSCODE_DATA=/opt/deklarant/data/hscodes.json
LAWS_DATA=/opt/deklarant/data/laws.json
DOCS_DATA=/opt/deklarant/data/docs.json
COUNTRIES_DATA=/opt/deklarant/data/countries.json
TAXONOMY_DATA=/opt/deklarant/data/taxonomy.json
USERS_DATA=/opt/deklarant/data/users.json
EOF
	chmod 600 "$ENV_FILE"
	echo "yasaldi: $ENV_FILE"
	echo "⚠️ ANTHROPIC_API_KEY bo'sh — uni qo'lda yozing"
else
	echo "$ENV_FILE allaqachon bor — tegilmadi"
fi

log "Xavfsizlik devori"
ufw allow OpenSSH >/dev/null
ufw allow 80/tcp >/dev/null
ufw allow 443/tcp >/dev/null
# ⚠️ 8080 TASHQARIGA OCHILMAYDI: backend faqat Caddy orqali
# ko'rinishi kerak, aks holda TLS va CORS chetlab o'tilardi.
yes | ufw enable >/dev/null 2>&1 || true

log "Tayyor"
cat <<'EOF'
Keyingi qadamlar:
  1. /etc/deklarant.env ichiga ANTHROPIC_API_KEY ni yozing
  2. Mahalliy mashinada: bash deploy/release.sh
  3. DNS: deklarantpro.uz A -> shu server IP si

Admin paroli:
  grep ADMIN_PASSWORD /etc/deklarant.env
EOF
