package main

/*
	domainperms.go

	API pro správu oprávnění na konkrétní proxy domény. Admin může u každé
	domény nastavit, kteří uživatelé ji vidí / můžou spravovat. Uživatel pak
	vidí jen domény, které mu admin přiřadil.
*/

import (
	"encoding/json"
	"net/http"
	"strings"

	"imuslab.com/zoraxy/mod/auth"
	"imuslab.com/zoraxy/mod/utils"
)

func RegisterDomainPermissionAPIs(authRouter *auth.RouterDef) {
	authRouter.HandleFunc("/api/proxy/domainPermissions", handleDomainPermissions)
}

// requireDomainAccess ověří, že aktuální uživatel smí přistupovat k dané doméně.
// Pokud ne, pošle chybu a vrátí false.
func requireDomainAccess(w http.ResponseWriter, r *http.Request, host string) bool {
	username, err := authAgent.GetUsernameFromRequest(r)
	if err != nil {
		utils.SendErrorResponse(w, "user not logged in")
		return false
	}
	if authAgent.UserCanAccessDomain(username, host) {
		return true
	}
	utils.SendErrorResponse(w, "insufficient permission for this domain")
	return false
}

// handleDomainPermissions - GET vrací seznam uživatelů a kdo má k doméně přístup,
// POST uloží, kteří uživatelé k doméně přístup mají.
func handleDomainPermissions(w http.ResponseWriter, r *http.Request) {
	if !authAgent.IsAdmin(r) {
		utils.SendErrorResponse(w, "insufficient permission")
		return
	}

	if r.Method == http.MethodGet {
		host, err := utils.GetPara(r, "host")
		if err != nil {
			utils.SendErrorResponse(w, "host not defined")
			return
		}

		users := authAgent.ListUserObjects()
		result := []map[string]interface{}{}
		for _, u := range users {
			result = append(result, map[string]interface{}{
				"username":  u.Username,
				"hasAccess": authAgent.UserCanAccessDomain(u.Username, host),
			})
		}
		js, _ := json.Marshal(result)
		utils.SendJSONResponse(w, string(js))
		return
	}

	if r.Method == http.MethodPost {
		host, err := utils.PostPara(r, "host")
		if err != nil {
			utils.SendErrorResponse(w, "host not defined")
			return
		}
		host = strings.TrimSpace(host)

		// comma separated seznam uživatelů, kteří k doméně mají mít přístup
		allowedUsers := map[string]bool{}
		for _, u := range strings.Split(r.FormValue("users"), ",") {
			u = strings.TrimSpace(u)
			if u != "" {
				allowedUsers[u] = true
			}
		}

		// Admin má vždy přístup, takže ho přeskakujeme.
		for _, username := range authAgent.ListUsers() {
			if authAgent.GetUserRole(username) == auth.DefaultRoleName {
				continue
			}
			domains := authAgent.GetUserDomains(username)
			newDomains := addOrRemoveDomain(domains, host, allowedUsers[username])
			_ = authAgent.SetUserDomains(username, newDomains)
		}

		utils.SendOK(w)
		return
	}

	http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
}

// HandleUnderAttack zapne / vypne režim "under attack" pro dané domény.
// Když je zapnutý, návštěvníci uvidí interstitial místo cílového webu.
func HandleUnderAttack(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	enable := r.FormValue("enable") == "true" || r.FormValue("enable") == "1"

	domains := []string{}
	for _, d := range strings.Split(r.FormValue("domains"), ",") {
		d = strings.TrimSpace(d)
		if d != "" {
			domains = append(domains, d)
		}
	}
	if len(domains) == 0 {
		utils.SendErrorResponse(w, "no domains given")
		return
	}

	for _, domain := range domains {
		if !requireDomainAccess(w, r, domain) {
			return
		}
		ep, err := dynamicProxyRouter.LoadProxy(domain)
		if err != nil {
			utils.SendErrorResponse(w, "domain not found: "+domain)
			return
		}
		ep.UnderAttack = enable
		if err := SaveReverseProxyConfig(ep); err != nil {
			utils.SendErrorResponse(w, "failed to save: "+domain)
			return
		}
	}

	utils.SendOK(w)
}

// addOrRemoveDomain přidá / odebere host ze seznamu domén (bez duplicit).
func addOrRemoveDomain(domains []string, host string, add bool) []string {
	out := []string{}
	found := false
	for _, d := range domains {
		if d == host {
			found = true
			if add {
				out = append(out, d)
			}
			continue
		}
		out = append(out, d)
	}
	if add && !found {
		out = append(out, host)
	}
	return out
}
