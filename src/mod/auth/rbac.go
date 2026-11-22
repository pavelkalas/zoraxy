package auth

/*
	rbac.go

	Role-based access control pro management rozhraní Zoraxy.

	Tady je definovaný jednoduchý RBAC systém. Každý uživatel má přiřazenou
	roli a každá role má seznam oprávnění (permission). Oprávnění odpovídají
	jednotlivým sekcím management rozhraní (Status, Ingress, Reverse Proxy, ...).

	Role jsou uložené v databázové tabulce "roles" pod klíčem "role/<název>".
	Přiřazení role a flag "musí si změnit heslo" jsou uložené v tabulce "auth"
	pod klíči "role/<username>" a "mustchangepw/<username>".
*/

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
)

// Tabulka pro ukládání definic rolí
const TableRoles = "roles"

// Předpona klíče pro role v tabulce "roles"
const roleKeyPrefix = "role/"

// Název výchozí administrátorské role. Tahle role má vždycky všechna oprávnění.
const DefaultRoleName = "admin"

// Jednotlivá oprávnění. Každé odpovídá jedné sekci v management rozhraní.
const (
	PermissionStatus       = "status"
	PermissionIngress      = "ingress"
	PermissionReverseProxy = "reverseproxy"
	PermissionAccess       = "access"
	PermissionSecurity     = "security"
	PermissionServices     = "services"
	PermissionNetworking   = "networking"
	PermissionStats        = "stats"
	PermissionUtilities    = "utils"
	PermissionAppStore     = "appstore"
	PermissionPlugins      = "plugins"
	PermissionUsers        = "users"
)

// AllPermissions je seznam všech známých oprávnění. Používá se hlavně pro
// výchozí admin roli a pro vykreslení zaškrtávacích políček ve webovém UI.
var AllPermissions = []string{
	PermissionStatus,
	PermissionIngress,
	PermissionReverseProxy,
	PermissionAccess,
	PermissionSecurity,
	PermissionServices,
	PermissionNetworking,
	PermissionStats,
	PermissionUtilities,
	PermissionAppStore,
	PermissionPlugins,
	PermissionUsers,
}

// Role reprezentuje jednu roli v systému.
type Role struct {
	Name        string   `json:"name"`        // Jedinečný název role
	Description string   `json:"description"` // Volitelný popis
	Permissions []string `json:"permissions"` // Seznam oprávnění
}

// UserInfo je datová struktura, kterou vracíme do UI pro seznam uživatelů.
type UserInfo struct {
	Username           string `json:"username"`
	Email              string `json:"email"`
	Role               string `json:"role"`
	MustChangePassword bool   `json:"mustChangePassword"`
}

func (a *AuthAgent) roleKey(name string) string {
	return roleKeyPrefix + name
}

func isKnownPermission(perm string) bool {
	for _, p := range AllPermissions {
		if p == perm {
			return true
		}
	}
	return false
}

// normalizePermissions pročistí seznam oprávnění: odstraní duplicity, prázdné
// hodnoty a neznámá oprávnění.
func normalizePermissions(perms []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, p := range perms {
		p = strings.TrimSpace(p)
		if p == "" || seen[p] || !isKnownPermission(p) {
			continue
		}
		seen[p] = true
		out = append(out, p)
	}
	return out
}

// ensureDefaultRole vytvoří výchozí admin roli, pokud ještě neexistuje.
func (a *AuthAgent) ensureDefaultRole() {
	if a.RoleExists(DefaultRoleName) {
		return
	}
	adminRole := &Role{
		Name:        DefaultRoleName,
		Description: "Built-in administrator role with full access",
		Permissions: append([]string{}, AllPermissions...),
	}
	_ = a.SaveRole(adminRole)
}

// backfillExistingUsers doplní chybějící role a flag "musí si změnit heslo"
// uživatelům, kteří vznikli ještě před zavedením RBAC (např. starý admin účet).
func (a *AuthAgent) backfillExistingUsers() {
	for _, username := range a.ListUsers() {
		if !a.Database.KeyExists("auth", "role/"+username) {
			_ = a.Database.Write("auth", "role/"+username, DefaultRoleName)
		}
		if !a.Database.KeyExists("auth", "mustchangepw/"+username) {
			_ = a.Database.Write("auth", "mustchangepw/"+username, false)
		}
	}
}

func (a *AuthAgent) RoleExists(name string) bool {
	return a.Database.KeyExists(TableRoles, a.roleKey(name))
}

func (a *AuthAgent) GetRole(name string) (*Role, error) {
	if !a.RoleExists(name) {
		return nil, errors.New("role not found")
	}
	role := &Role{}
	if err := a.Database.Read(TableRoles, a.roleKey(name), role); err != nil {
		return nil, err
	}
	if role.Name == "" {
		role.Name = name
	}
	return role, nil
}

func (a *AuthAgent) SaveRole(role *Role) error {
	if role == nil || strings.TrimSpace(role.Name) == "" {
		return errors.New("role name cannot be empty")
	}
	role.Name = strings.TrimSpace(role.Name)
	if strings.ContainsAny(role.Name, "/\\") {
		return errors.New("role name contains invalid characters")
	}
	role.Permissions = normalizePermissions(role.Permissions)
	return a.Database.Write(TableRoles, a.roleKey(role.Name), role)
}

func (a *AuthAgent) DeleteRole(name string) error {
	if name == DefaultRoleName {
		return errors.New("the default admin role cannot be deleted")
	}
	if !a.RoleExists(name) {
		return errors.New("role not found")
	}

	// Nesmíme smazat roli, která je pořád přiřazená nějakému uživateli.
	for _, username := range a.ListUsers() {
		if a.GetUserRole(username) == name {
			return errors.New("role is still assigned to one or more users")
		}
	}

	return a.Database.Delete(TableRoles, a.roleKey(name))
}

func (a *AuthAgent) ListRoles() ([]*Role, error) {
	entries, err := a.Database.ListTable(TableRoles)
	if err != nil {
		// Tabulka zatím neexistuje, vrať prázdný seznam.
		return []*Role{}, nil
	}

	roles := []*Role{}
	for _, keypairs := range entries {
		if len(keypairs) < 2 {
			continue
		}
		key := string(keypairs[0])
		if !strings.HasPrefix(key, roleKeyPrefix) {
			continue
		}
		name := strings.TrimPrefix(key, roleKeyPrefix)
		role := &Role{}
		if err := json.Unmarshal(keypairs[1], role); err != nil {
			continue
		}
		if role.Name == "" {
			role.Name = name
		}
		roles = append(roles, role)
	}
	return roles, nil
}

// GetUserRole vrátí název role přiřazené uživateli. Když role chybí, vrací admin.
func (a *AuthAgent) GetUserRole(username string) string {
	var role string
	if err := a.Database.Read("auth", "role/"+username, &role); err != nil {
		return DefaultRoleName
	}
	if strings.TrimSpace(role) == "" {
		return DefaultRoleName
	}
	return role
}

func (a *AuthAgent) SetUserRole(username, role string) error {
	role = strings.TrimSpace(role)
	if !a.RoleExists(role) {
		return errors.New("role does not exist")
	}
	return a.Database.Write("auth", "role/"+username, role)
}

func (a *AuthAgent) GetUserMustChangePassword(username string) bool {
	var must bool
	if err := a.Database.Read("auth", "mustchangepw/"+username, &must); err != nil {
		return false
	}
	return must
}

func (a *AuthAgent) SetUserMustChangePassword(username string, must bool) error {
	return a.Database.Write("auth", "mustchangepw/"+username, must)
}

// GetUserPermissions vrátí oprávnění pro daného uživatele. Admin role má
// vždycky všechno, bez ohledu na to, co je uložené v roli.
func (a *AuthAgent) GetUserPermissions(username string) []string {
	roleName := a.GetUserRole(username)
	if roleName == DefaultRoleName {
		return append([]string{}, AllPermissions...)
	}
	role, err := a.GetRole(roleName)
	if err != nil {
		return []string{}
	}
	return append([]string{}, role.Permissions...)
}

// GetUsernameFromRequest vytáhne jméno přihlášeného uživatele ze session,
// aniž by cokoli zapisovala do odpovědi. Hodí se pro kontrolu oprávnění.
func (a *AuthAgent) GetUsernameFromRequest(r *http.Request) (string, error) {
	session, err := a.SessionStore.Get(r, a.SessionName)
	if err != nil {
		return "", err
	}
	if auth, ok := session.Values["authenticated"].(bool); !ok || !auth {
		return "", errors.New("user not logged in")
	}
	username, ok := session.Values["username"].(string)
	if !ok || username == "" {
		return "", errors.New("username not found in session")
	}
	return username, nil
}

// UserHasPermission zkontroluje, jestli má přihlášený uživatel dané oprávnění.
func (a *AuthAgent) UserHasPermission(r *http.Request, permission string) bool {
	if permission == "" {
		return true
	}
	username, err := a.GetUsernameFromRequest(r)
	if err != nil {
		return false
	}
	if a.GetUserRole(username) == DefaultRoleName {
		return true
	}
	for _, p := range a.GetUserPermissions(username) {
		if p == permission {
			return true
		}
	}
	return false
}

// IsAdmin vrátí true, pokud je přihlášený uživatel v admin roli.
func (a *AuthAgent) IsAdmin(r *http.Request) bool {
	username, err := a.GetUsernameFromRequest(r)
	if err != nil {
		return false
	}
	return a.GetUserRole(username) == DefaultRoleName
}

// MustChangePassword vrátí true, pokud má přihlášený uživatel nastavenou
// povinnou změnu hesla při příštím přihlášení.
func (a *AuthAgent) MustChangePassword(r *http.Request) bool {
	username, err := a.GetUsernameFromRequest(r)
	if err != nil {
		return false
	}
	return a.GetUserMustChangePassword(username)
}

// IsPasswordChangeAllowedEndpoint říká, které API endpointy smí volat uživatel,
// který si ještě nezměnil heslo.
func (a *AuthAgent) IsPasswordChangeAllowedEndpoint(endpoint string) bool {
	switch endpoint {
	case "/api/auth/changePassword",
		"/api/auth/logout",
		"/api/auth/checkLogin",
		"/api/auth/username",
		"/api/auth/authEnabled",
		"/api/auth/myPermissions",
		"/api/auth/myRole",
		"/api/auth/mustChangePassword":
		return true
	}
	return false
}

// PermissionForPath namapuje URL endpoint na oprávnění, které je pro něj potřeba.
// Vrací prázdný string u endpointů, které nepotřebují žádné konkrétní oprávnění
// (ty jsou registrované přímo na targetMux a kontrolují si auth samy).
func (a *AuthAgent) PermissionForPath(path string) string {
	// Endpointy pod /api/auth/ jsou registrované zvlášť (RegisterAuthAPIs)
	// a oprávnění nepotřebují.
	if strings.HasPrefix(path, "/api/auth/") {
		return ""
	}

	// Správa uživatelů a rolí je jen pro adminy / roli s oprávněním "users".
	if strings.HasPrefix(path, "/api/users/") {
		return PermissionUsers
	}

	// Ingress (globální nastavení) – konkrétní proxy endpointy.
	switch path {
	case "/api/proxy/setIncoming",
		"/api/proxy/useHttpsRedirect",
		"/api/proxy/listenPort80",
		"/api/proxy/requestIsProxied",
		"/api/proxy/developmentMode",
		"/api/proxy/proxyProtocol",
		"/api/proxy/quic",
		"/api/proxy/timeouts":
		return PermissionIngress
	}
	if strings.HasPrefix(path, "/api/proxy/listeningPorts/") {
		return PermissionIngress
	}

	if strings.HasPrefix(path, "/api/proxy/") || strings.HasPrefix(path, "/api/streamprox/") {
		return PermissionReverseProxy
	}
	if strings.HasPrefix(path, "/api/cert/") ||
		strings.HasPrefix(path, "/api/acme/") ||
		strings.HasPrefix(path, "/api/sso/") {
		return PermissionSecurity
	}
	if strings.HasPrefix(path, "/api/redirect/") ||
		strings.HasPrefix(path, "/api/access/") ||
		strings.HasPrefix(path, "/api/blacklist/") ||
		strings.HasPrefix(path, "/api/whitelist/") ||
		strings.HasPrefix(path, "/api/quickban/") ||
		strings.HasPrefix(path, "/api/trustedproxy/") ||
		strings.HasPrefix(path, "/api/pathrule/") {
		return PermissionAccess
	}
	if strings.HasPrefix(path, "/api/stats/") || strings.HasPrefix(path, "/api/sysinfo/") {
		return PermissionStatus
	}
	if strings.HasPrefix(path, "/api/analytic/") {
		return PermissionStats
	}
	if strings.HasPrefix(path, "/api/webserv/") || strings.HasPrefix(path, "/api/utm/") {
		return PermissionServices
	}
	if strings.HasPrefix(path, "/api/tools/") ||
		strings.HasPrefix(path, "/api/mdns/") ||
		strings.HasPrefix(path, "/api/info/geoip") ||
		strings.HasPrefix(path, "/api/info/ipcheck") ||
		strings.HasPrefix(path, "/api/geodb/") {
		return PermissionNetworking
	}
	if strings.HasPrefix(path, "/api/plugins/store/") {
		return PermissionAppStore
	}
	if strings.HasPrefix(path, "/api/plugins/") {
		return PermissionPlugins
	}
	if strings.HasPrefix(path, "/api/conf/") ||
		strings.HasPrefix(path, "/api/log/") ||
		strings.HasPrefix(path, "/api/logger/") {
		return PermissionUtilities
	}
	if strings.HasPrefix(path, "/api/docker/") {
		return PermissionReverseProxy
	}

	// Neznámý endpoint: vyžaduje admina. Router to pozná podle prázdného
	// stringu a zařídí, že se k němu dostane jen admin.
	return ""
}

// ListUserObjects vrátí seznam uživatelů včetně role, emailu a flagu
// "musí si změnit heslo" pro vykreslení v UI.
func (a *AuthAgent) ListUserObjects() []UserInfo {
	results := []UserInfo{}
	for _, username := range a.ListUsers() {
		email := ""
		_ = a.Database.Read("auth", "email/"+username, &email)
		results = append(results, UserInfo{
			Username:           username,
			Email:              email,
			Role:               a.GetUserRole(username),
			MustChangePassword: a.GetUserMustChangePassword(username),
		})
	}
	return results
}

// CreateUserAccountFull vytvoří uživatele i s rolí a flagem povinné změny hesla.
func (a *AuthAgent) CreateUserAccountFull(newusername, password, email, role string, mustChangePassword bool) error {
	if a.UserExists(newusername) {
		return errors.New("user with same name already exists")
	}

	role = strings.TrimSpace(role)
	if role == "" {
		role = DefaultRoleName
	}
	if !a.RoleExists(role) {
		return errors.New("role does not exist")
	}

	key := newusername
	if err := a.Database.Write("auth", "passhash/"+key, Hash(password)); err != nil {
		return err
	}
	if email != "" {
		if err := a.Database.Write("auth", "email/"+key, email); err != nil {
			return err
		}
	}
	if err := a.Database.Write("auth", "role/"+key, role); err != nil {
		return err
	}
	if err := a.Database.Write("auth", "mustchangepw/"+key, mustChangePassword); err != nil {
		return err
	}
	return nil
}

// ChangePassword změní heslo uživatele a zruší flag "musí si změnit heslo".
func (a *AuthAgent) ChangePassword(username, newPassword string) error {
	if !a.UserExists(username) {
		return errors.New("user does not exist")
	}
	if err := a.Database.Write("auth", "passhash/"+username, Hash(newPassword)); err != nil {
		return err
	}
	return a.SetUserMustChangePassword(username, false)
}
