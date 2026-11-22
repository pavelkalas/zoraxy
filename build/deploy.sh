#!/bin/sh
# ============================================================
#  deploy.sh
#
#  Nahraje zoraxy-multi image z taru, prepise image v compose
#  a udela docker compose up -d (recreate).
#
#  Spusteni:
#      chmod +x deploy.sh
#      ./deploy.sh
#
#  Cesty si uprav podle sebe. Vychozi hodnoty sedi na rozlozeni:
#      /root/zoraxy-multi.tar          (tar s imagem)
#      /root/zoraxy/docker-compose.yml (compose + config volume)
# ============================================================

TAR="/root/zoraxy-multi.tar"
COMPOSE_DIR="/root/zoraxy"
COMPOSE_FILE="$COMPOSE_DIR/docker-compose.yml"
IMAGE="zoraxy-multi:latest"

set -e

echo "=== Zoraxy deploy ==="
echo "Tar     : $TAR"
echo "Compose : $COMPOSE_FILE"
echo "Image   : $IMAGE"
echo

# 1) Kontrola vstupu
if [ ! -f "$TAR" ]; then
    echo "CHYBA: tar soubor '$TAR' neexistuje."
    exit 1
fi
if [ ! -f "$COMPOSE_FILE" ]; then
    echo "CHYBA: compose soubor '$COMPOSE_FILE' neexistuje."
    exit 1
fi

# 2) Zaloha configu (pred jakoukoli zmenou)
if [ -d "$COMPOSE_DIR/config" ]; then
    BACKUP="$COMPOSE_DIR/config-zaloha-$(date +%Y%m%d-%H%M%S)"
    echo "[1/4] Zaloha config -> $BACKUP"
    cp -r "$COMPOSE_DIR/config" "$BACKUP"
else
    echo "[1/4] Slozka config neexistuje, zalohu preskakuju."
fi

# 3) Nahrani image
echo "[2/4] Nacitam image z taru..."
docker load -i "$TAR"

# 4) Prepsani image v compose (idempotentni)
echo "[3/4] Upravuji image v docker-compose.yml..."
sed -i "s|image:.*|image: $IMAGE|" "$COMPOSE_FILE"
grep -n "image:" "$COMPOSE_FILE"

# 5) Recreate
echo "[4/4] Spoustim docker compose up -d..."
cd "$COMPOSE_DIR"
if docker compose version >/dev/null 2>&1; then
    docker compose up -d
elif command -v docker-compose >/dev/null 2>&1; then
    docker-compose up -d
else
    echo "CHYBA: docker compose neni dostupny."
    exit 1
fi

echo
echo "=== Hotovo ==="
docker ps --format "table {{.Names}}\t{{.Image}}\t{{.Status}}" | head -5
