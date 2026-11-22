# Build složka - postavení Docker image

Tahle složka slouží k tomu, abys ze svého upraveného Zoraxy (s podporou více
uživatelů a rolí) postavil Docker image a dostal ho na svůj běžící server.

Build běží přes **WSL** (protože tam máš Docker), výsledek se uloží jako tar do
`build/output/`.

## Soubory

| Soubor      | K čemu slouží                                             |
|-------------|-----------------------------------------------------------|
| `build.bat` | Postaví image a uloží ho do `build/output/zoraxy-multi.tar` |
| `Dockerfile`| Dockerfile **bez ZeroTieru** (používá ho `build.bat`)      |
| `load.bat`  | Na cíli načte tar zpět do Dockeru (`docker load`)         |
| `push.bat`  | (volitelné) pushne image do Docker registru               |
| `deploy.sh` | Na serveru: `docker load` + přepíše image + `compose up -d` |

> **Proč bez ZeroTieru?** Oficiální Dockerfile kompiluje ZeroTier (funkce
> "Global Area Network"), což je zdaleka nejpomalejší část buildu (stahuje
> zdroják a kompiluje celý C++/Rust projekt). Tenhle `build/Dockerfile` ho
> vynechává, takže build je o mnoho minut rychlejší. **Pokud ZeroTier
> nepoužíváš (env `ZEROTIER=false`), je to přesně to, co chceš.**

## Postup (nejjednodušší - přes tar)

### 1. Na svém PC postav image

Dvakrát klikni na **`build.bat`** (nebo ho spusť z příkazové řádky). Co se stane:

1. Zkopíruje `src/`, `docker/Dockerfile` a `docker/entrypoint.go` do dočasného
   build contextu `build/.context`.
2. Přes WSL spustí `docker build -t zoraxy-multi:latest`.
3. `docker save` uloží image do `build/output/zoraxy-multi.tar`.

> První build trvá déle (stahuje base image a kompiluje), další jsou rychlejší.

### 2. Přenos na server

Celou složku `build/output/` zkopíruj na server (třeba přes `scp`):

```powershell
scp -r build\output\ uzivatel@server:/cesta/k/zoraxy/
```

### 3. Na serveru nahraj image

Na serveru (v té složce) spusť **`load.bat`** (na Windows) nebo ručně:

```bash
docker load -i zoraxy-multi.tar
```

### 4. Přepni compose na nový image

V `docker-compose.yml` změň image:

```yaml
# image: zoraxydocker/zoraxy:latest
image: zoraxy-multi:latest
```

A restartuj:

```bash
docker compose up -d
```

> **Před nasazením si zálohuj config volume** (je v něm `sys.db` a `conf/`).
> Migrace na nové uživatele/role proběhne při startu automaticky.

## Varianta přes registr (místo kopírování taru)

Pokud máš přístup k Docker Hubu / GHCR / vlastnímu registru, můžeš místo
kopírování taru pushnout image do registru a na serveru ho jen pullnout:

1. V `push.bat` si nastav `REGISTRY` (např. `tvujlogin/zoraxy-multi`).
2. `docker login` (pro privátní registry).
3. Spusť `push.bat`.
4. Na serveru: `docker pull tvujlogin/zoraxy-multi:latest`.

## Rollback

Pokud by nová verze nešla, vrať v compose `image: zoraxydocker/zoraxy:latest`
a `docker compose up -d`. Stará verze nové klíče v databázi ignoruje, takže se
nic nerozbije (pro jistotu máš ale zálohu configu).
