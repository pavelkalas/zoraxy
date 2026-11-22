package main

/*
	usermanagement.go

	API endpointy pro správu uživatelů a rolí management rozhraní Zoraxy.
	Tyhle endpointy jsou registrované přes authRouter, takže se na ně
	automaticky aplikuje kontrola přihlášení a oprávnění "users".
*/

import (
	"encoding/json"
	"net/http"
	"net/mail"
	"strings"

	"imuslab.com/zoraxy/mod/auth"
	"imuslab.com/zoraxy/mod/utils"
)

func RegisterUserManagementAPIs(authRouter *auth.RouterDef) {
	// Uživatelé
	authRouter.HandleFunc("/api/users/list", handleUsersList)
	authRouter.HandleFunc("/api/users/create", handleUserCreate)
	authRouter.HandleFunc("/api/users/update", handleUserUpdate)
	authRouter.HandleFunc("/api/users/delete", handleUserDelete)

	// Role
	authRouter.HandleFunc("/api/users/roles/list", handleRolesList)
	authRouter.HandleFunc("/api/users/roles/create", handleRoleCreate)
	authRouter.HandleFunc("/api/users/roles/update", handleRoleUpdate)
	authRouter.HandleFunc("/api/users/roles/delete", handleRoleDelete)
}

// countAdmins spočítá, kolik uživatelů je momentálně v admin roli.
func countAdmins() int {
	count := 0
	for _, u := range authAgent.ListUserObjects() {
		if u.Role == auth.DefaultRoleName {
			count++
		}
	}
	return count
}

func handleUsersList(w http.ResponseWriter, r *http.Request) {
	users := authAgent.ListUserObjects()
	js, _ := json.Marshal(users)
	utils.SendJSONResponse(w, string(js))
}

func handleUserCreate(w http.ResponseWriter, r *http.Request) {
	username := strings.TrimSpace(r.FormValue("username"))
	password := r.FormValue("password")
	email := strings.TrimSpace(r.FormValue("email"))
	role := strings.TrimSpace(r.FormValue("role"))
	mustChangePassword := r.FormValue("mustChangePassword") == "true" || r.FormValue("mustChangePassword") == "1"

	if username == "" {
		utils.SendErrorResponse(w, "missing username")
		return
	}
	if password == "" {
		utils.SendErrorResponse(w, "missing password")
		return
	}
	if strings.ContainsAny(username, "/\\") {
		utils.SendErrorResponse(w, "username contains invalid characters")
		return
	}
	if email != "" {
		if _, err := mail.ParseAddress(email); err != nil {
			utils.SendErrorResponse(w, "invalid email format")
			return
		}
	}

	if err := authAgent.CreateUserAccountFull(username, password, email, role, mustChangePassword); err != nil {
		utils.SendErrorResponse(w, err.Error())
		return
	}

	//Přiřaď uživateli konkrétní proxy domény (pokud jsou poslané).
	if err := authAgent.SetUserDomains(username, parseDomains(r)); err != nil {
		utils.SendErrorResponse(w, err.Error())
		return
	}

	//Globální oprávnění vidět všechny domény.
	domainsAll := r.FormValue("domainsAll") == "true" || r.FormValue("domainsAll") == "1"
	_ = authAgent.SetUserDomainsAll(username, domainsAll)

	//Permission matrix po jednotlivých záložkách (allow + viewOnly).
	_ = authAgent.SetUserTabPermissions(username, parseTabPermissions(r))

	authAgent.Logger.PrintAndLog("auth", "Admin created user "+username, nil)
	utils.SendOK(w)
}

func handleUserUpdate(w http.ResponseWriter, r *http.Request) {
	username, err := utils.PostPara(r, "username")
	if err != nil {
		utils.SendErrorResponse(w, "missing username")
		return
	}
	if !authAgent.UserExists(username) {
		utils.SendErrorResponse(w, "user not found")
		return
	}

	// Email
	if _, ok := r.Form["email"]; ok {
		email := strings.TrimSpace(r.FormValue("email"))
		if email != "" {
			if _, err := mail.ParseAddress(email); err != nil {
				utils.SendErrorResponse(w, "invalid email format")
				return
			}
			_ = authAgent.Database.Write("auth", "email/"+username, email)
		} else {
			_ = authAgent.Database.Delete("auth", "email/"+username)
		}
	}

	// Heslo (volitelné)
	if password := r.FormValue("password"); password != "" {
		if err := authAgent.ChangePassword(username, password); err != nil {
			utils.SendErrorResponse(w, err.Error())
			return
		}
	}

	// Role
	if role := strings.TrimSpace(r.FormValue("role")); role != "" {
		currentRole := authAgent.GetUserRole(username)
		if currentRole == auth.DefaultRoleName && role != auth.DefaultRoleName && countAdmins() <= 1 {
			utils.SendErrorResponse(w, "cannot change the role of the last admin account")
			return
		}
		if err := authAgent.SetUserRole(username, role); err != nil {
			utils.SendErrorResponse(w, err.Error())
			return
		}
	}

	// Flag "musí si změnit heslo"
	if values, ok := r.Form["mustChangePassword"]; ok && len(values) > 0 {
		must := values[0] == "true" || values[0] == "1"
		_ = authAgent.SetUserMustChangePassword(username, must)
	}

	// Přiřazené proxy domény
	if _, ok := r.Form["domains"]; ok {
		if err := authAgent.SetUserDomains(username, parseDomains(r)); err != nil {
			utils.SendErrorResponse(w, err.Error())
			return
		}
	}

	// Globální oprávnění vidět všechny domény
	if values, ok := r.Form["domainsAll"]; ok && len(values) > 0 {
		all := values[0] == "true" || values[0] == "1"
		_ = authAgent.SetUserDomainsAll(username, all)
	}

	// Permission matrix po jednotlivých záložkách (allow + viewOnly).
	if _, ok := r.Form["tabPermissions"]; ok {
		if err := authAgent.SetUserTabPermissions(username, parseTabPermissions(r)); err != nil {
			utils.SendErrorResponse(w, err.Error())
			return
		}
	}

	utils.SendOK(w)
}

func handleUserDelete(w http.ResponseWriter, r *http.Request) {
	username, err := utils.PostPara(r, "username")
	if err != nil {
		utils.SendErrorResponse(w, "missing username")
		return
	}

	currentUser, _ := authAgent.GetUserName(w, r)
	if username == currentUser {
		utils.SendErrorResponse(w, "you cannot delete your own account")
		return
	}
	if !authAgent.UserExists(username) {
		utils.SendErrorResponse(w, "user not found")
		return
	}
	if authAgent.GetUserRole(username) == auth.DefaultRoleName && countAdmins() <= 1 {
		utils.SendErrorResponse(w, "cannot delete the last admin account")
		return
	}

	if err := authAgent.UnregisterUser(username); err != nil {
		utils.SendErrorResponse(w, err.Error())
		return
	}

	authAgent.Logger.PrintAndLog("auth", "Admin deleted user "+username, nil)
	utils.SendOK(w)
}

func handleRolesList(w http.ResponseWriter, r *http.Request) {
	roles, err := authAgent.ListRoles()
	if err != nil {
		utils.SendErrorResponse(w, err.Error())
		return
	}
	js, _ := json.Marshal(roles)
	utils.SendJSONResponse(w, string(js))
}

// parseDomains přečte seznam proxy domén z POST parametru "domains"
// (čárkou oddělený seznam). Prázdné hodnoty se vynechají.
func parseDomains(r *http.Request) []string {
	raw := r.FormValue("domains")
	if raw == "" {
		return []string{}
	}
	parts := strings.Split(raw, ",")
	out := []string{}
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// parseTabPermissions přečte permission matrix z POST parametru "tabPermissions",
// který očekáváme jako JSON mapu: {"cert":{"allow":true,"viewOnly":false}, ...}.
func parseTabPermissions(r *http.Request) map[string]auth.TabPermission {
	perms := map[string]auth.TabPermission{}
	raw := r.FormValue("tabPermissions")
	if raw == "" {
		return perms
	}
	_ = json.Unmarshal([]byte(raw), &perms)
	return perms
}

// parsePermissions přečte oprávnění z POST parametru "permissions", který
// očekáváme jako čárkou oddělený seznam.
func parsePermissions(r *http.Request) []string {
	raw := r.FormValue("permissions")
	if raw == "" {
		return []string{}
	}
	parts := strings.Split(raw, ",")
	out := []string{}
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func handleRoleCreate(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimSpace(r.FormValue("name"))
	description := strings.TrimSpace(r.FormValue("description"))
	permissions := parsePermissions(r)

	if name == "" {
		utils.SendErrorResponse(w, "role name cannot be empty")
		return
	}
	if authAgent.RoleExists(name) {
		utils.SendErrorResponse(w, "role already exists")
		return
	}

	role := &auth.Role{Name: name, Description: description, Permissions: permissions}
	if err := authAgent.SaveRole(role); err != nil {
		utils.SendErrorResponse(w, err.Error())
		return
	}
	utils.SendOK(w)
}

func handleRoleUpdate(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimSpace(r.FormValue("name"))
	if name == "" {
		utils.SendErrorResponse(w, "role name cannot be empty")
		return
	}
	if !authAgent.RoleExists(name) {
		utils.SendErrorResponse(w, "role not found")
		return
	}

	description := strings.TrimSpace(r.FormValue("description"))
	permissions := parsePermissions(r)
	if _, ok := r.Form["permissions"]; !ok {
		// Oprávnění nebyla poslaná, zachovej stávající.
		existing, err := authAgent.GetRole(name)
		if err == nil {
			permissions = existing.Permissions
		}
	}

	role := &auth.Role{Name: name, Description: description, Permissions: permissions}
	if err := authAgent.SaveRole(role); err != nil {
		utils.SendErrorResponse(w, err.Error())
		return
	}
	utils.SendOK(w)
}

func handleRoleDelete(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimSpace(r.FormValue("name"))
	if name == "" {
		utils.SendErrorResponse(w, "role name cannot be empty")
		return
	}
	if err := authAgent.DeleteRole(name); err != nil {
		utils.SendErrorResponse(w, err.Error())
		return
	}
	utils.SendOK(w)
}

