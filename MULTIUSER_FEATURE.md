# Podpora více uživatelů a rolí (RBAC)

Tahle změna přidává do management rozhraní Zoraxy podporu více uživatelů,
rolí a oprávnění. Původně šel Zoraxy ovládat jen přes jeden administrátorský
účet, teď můžeš vytvořit víc uživatelů a každému dát jen to, co smí ovládat.

## Co je nového

- **Více uživatelů** – můžeš si založit libovolný počet účtů pro management rozhraní.
- **Role s oprávněními** – každý uživatel má přiřazenou roli a role určuje,
  které sekce (Status, Ingress, Reverse Proxy, Security, ...) smí vidět a používat.
- **Uživatele vytváří vždy admin** – není žádná samoobslužná registrace. První
  admin se pořád zakládá přes přihlašovací stránku při prvním spuštění, další
  uživatele pak přidává admin v sekci **Users**.
- **"Uživatel si musí změnit heslo při prvním přihlášení"** – při zakládání
  (nebo editaci) uživatele můžeš zaškrtnout, že si uživatel musí po prvním
  přihlášení nastavit vlastní heslo.

## Oprávnění

Každé oprávnění odpovídá jedné sekci v menu. Výchozí role `admin` má všechna.

| Oprávnění     | Co povoluje                                              |
|---------------|----------------------------------------------------------|
| `status`      | Status / přehled systému                                 |
| `ingress`     | Ingress (globální nastavení) a Default Site              |
| `reverseproxy`| HTTP Proxy, Virtual Directory, Proxy Rules, Stream Proxy |
| `access`      | Redirection, Access Control, Trusted Proxy               |
| `security`    | TLS certifikáty, ACME, SSO / OAuth2                      |
| `services`    | Static Web Server, Uptime Monitor                        |
| `networking`  | Network Tools, Network Interfaces, mDNS, GeoIP           |
| `stats`       | Statistical Analysis                                     |
| `utils`       | Utilities (export konfigurace, logy)                     |
| `appstore`    | App Store                                                |
| `plugins`     | Plugin Manager                                           |
| `users`       | Správa uživatelů a rolí (tahle sekce)                    |

Oprávnění se vynucují na dvou místech:

1. **Backend** – každý API endpoint má přiřazené oprávnění. Když uživatel
   nemá oprávnění, dostane `403 Insufficient permission`.
2. **Frontend** – položky menu, na které uživatel nemá oprávnění, se mu skryjí.

## Jak to používat

1. Přihlas se jako admin.
2. V menu klikni na **Users**.
3. V záložce **Roles** si vytvoř roli (nebo použij výchozí `admin`).
   U role zaškrtni, která oprávnění má mít.
4. V záložce **Users** klikni na **Register New User**.
   - Zadej uživatelské jméno, heslo a vyber roli.
   - Volitelně zaškrtni **User must change password on first login**.
5. Hotovo – uživatel se přihlásí svým heslem. Pokud má zaškrtnutou povinnou
   změnu hesla, bude po přihlášení přesměrován na stránku pro změnu hesla
   a dokud si heslo nezmění, jinam se nedostane.

## Ukládání dat

- Uživatelé jsou pořád v tabulce `auth` (klíče `passhash/<username>`,
  `email/<username>`), nově přibyly `role/<username>` a
  `mustchangepw/<username>`.
- Definice rolí jsou v tabulce `roles` (klíč `role/<název>`).

Stávající instalace se nemusí bát – při startu se doplní role a flag pro
všechny uživatele, kteří vznikli před touhle změnou (dostanou roli `admin`).

## Omezení / pojistky

- Roli `admin` nelze smazat.
- Nelze smazat ani degradovat posledního admina.
- Uživatel nemůže smazat sám sebe.
- Roli, která je pořád přiřazená nějakému uživateli, nelze smazat.
