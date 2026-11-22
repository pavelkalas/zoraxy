package auth

import (
	"net/http"
	"os"
	"reflect"
	"testing"

	"imuslab.com/zoraxy/mod/database"
	"imuslab.com/zoraxy/mod/info/logger"
)

// newTestAuthAgent vytvoří AuthAgent s čerstvou databází v dočasné složce.
func newTestAuthAgent(t *testing.T) (*AuthAgent, string) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "zoraxy-rbac-test")
	if err != nil {
		t.Fatalf("nepodařilo se vytvořit temp složku: %v", err)
	}

	db, err := database.NewDatabase(tmpDir+"/test.db", database.GetRecommendedBackendType())
	if err != nil {
		t.Fatalf("nepodařilo se otevřít databázi: %v", err)
	}

	lg, err := logger.NewLogger("test", tmpDir)
	if err != nil {
		t.Fatalf("nepodařilo se vytvořit logger: %v", err)
	}

	agent := NewAuthenticationAgent("test-session", []byte("0123456789abcdef0123456789abcdef"), db, false, lg, func(w http.ResponseWriter, r *http.Request) {})
	return agent, tmpDir
}

// TestDefaultAdminRole ověří, že se po startu vytvoří výchozí admin role.
func TestDefaultAdminRole(t *testing.T) {
	agent, tmpDir := newTestAuthAgent(t)
	defer os.RemoveAll(tmpDir)

	if !agent.RoleExists(DefaultRoleName) {
		t.Fatal("výchozí admin role nebyla vytvořena")
	}

	perms := agent.GetUserPermissions("admin")
	if !reflect.DeepEqual(perms, AllPermissions) {
		t.Fatalf("admin by měl mít všechna oprávnění, dostal: %v", perms)
	}
}

// TestRoleCRUD ověří vytvoření, čtení, úpravu a mazání role.
func TestRoleCRUD(t *testing.T) {
	agent, tmpDir := newTestAuthAgent(t)
	defer os.RemoveAll(tmpDir)

	err := agent.SaveRole(&Role{Name: "operator", Description: "test", Permissions: []string{PermissionStatus, PermissionReverseProxy}})
	if err != nil {
		t.Fatalf("SaveRole selhalo: %v", err)
	}

	role, err := agent.GetRole("operator")
	if err != nil {
		t.Fatalf("GetRole selhalo: %v", err)
	}
	if role.Name != "operator" {
		t.Fatalf("špatný název role: %s", role.Name)
	}
	if len(role.Permissions) != 2 {
		t.Fatalf("role by měla mít 2 oprávnění, má: %v", role.Permissions)
	}

	// Neznámé oprávnění se má zahodit.
	err = agent.SaveRole(&Role{Name: "operator", Permissions: []string{PermissionStatus, "neznamy", PermissionStatus}})
	if err != nil {
		t.Fatalf("SaveRole (update) selhalo: %v", err)
	}
	role, _ = agent.GetRole("operator")
	if len(role.Permissions) != 1 {
		t.Fatalf("normalizace oprávnění selhala: %v", role.Permissions)
	}

	// Admin roli nejde smazat.
	if err := agent.DeleteRole(DefaultRoleName); err == nil {
		t.Fatal("smazání admin role by mělo selhat")
	}

	// Roli přiřazenou uživateli nejde smazat.
	_ = agent.CreateUserAccountFull("bob", "pass", "", "operator", false)
	if err := agent.DeleteRole("operator"); err == nil {
		t.Fatal("smazání přiřazené role by mělo selhat")
	}

	// Po přesunu uživatele do admin role jde role smazat.
	if err := agent.SetUserRole("bob", DefaultRoleName); err != nil {
		t.Fatalf("SetUserRole selhalo: %v", err)
	}
	if err := agent.DeleteRole("operator"); err != nil {
		t.Fatalf("DeleteRole selhalo: %v", err)
	}
}

// TestUserCreateAndPermissions ověří uživatele, roli a povinnou změnu hesla.
func TestUserCreateAndPermissions(t *testing.T) {
	agent, tmpDir := newTestAuthAgent(t)
	defer os.RemoveAll(tmpDir)

	_ = agent.SaveRole(&Role{Name: "viewer", Permissions: []string{PermissionStatus}})

	err := agent.CreateUserAccountFull("alice", "tajne", "alice@example.com", "viewer", true)
	if err != nil {
		t.Fatalf("CreateUserAccountFull selhalo: %v", err)
	}

	if !agent.UserExists("alice") {
		t.Fatal("alice měla být vytvořena")
	}
	if agent.GetUserRole("alice") != "viewer" {
		t.Fatalf("alice má špatnou roli: %s", agent.GetUserRole("alice"))
	}
	if !agent.GetUserMustChangePassword("alice") {
		t.Fatal("alice by měla mít povinnou změnu hesla")
	}

	perms := agent.GetUserPermissions("alice")
	if len(perms) != 1 || perms[0] != PermissionStatus {
		t.Fatalf("alice má špatná oprávnění: %v", perms)
	}

	if !agent.ValidateUsernameAndPassword("alice", "tajne") {
		t.Fatal("alice by se měla přihlásit s původním heslem")
	}
	if agent.ValidateUsernameAndPassword("alice", "spatne") {
		t.Fatal("alice by se neměla přihlásit se špatným heslem")
	}

	if err := agent.ChangePassword("alice", "noveheslo"); err != nil {
		t.Fatalf("ChangePassword selhalo: %v", err)
	}
	if agent.GetUserMustChangePassword("alice") {
		t.Fatal("po změně hesla by flag měl zmizet")
	}
	if !agent.ValidateUsernameAndPassword("alice", "noveheslo") {
		t.Fatal("alice by se měla přihlásit s novým heslem")
	}
}

// TestListUserObjects ověří výstup seznamu uživatelů.
func TestListUserObjects(t *testing.T) {
	agent, tmpDir := newTestAuthAgent(t)
	defer os.RemoveAll(tmpDir)

	_ = agent.CreateUserAccountFull("carol", "pass", "carol@example.com", DefaultRoleName, false)

	users := agent.ListUserObjects()
	if len(users) != 1 {
		t.Fatalf("měl by být 1 uživatel, je jich: %d", len(users))
	}
	if users[0].Username != "carol" || users[0].Email != "carol@example.com" || users[0].Role != DefaultRoleName {
		t.Fatalf("špatný výstup ListUserObjects: %+v", users[0])
	}
}

// TestUnregisterUser ověří, že se smažou i role a flag.
func TestUnregisterUser(t *testing.T) {
	agent, tmpDir := newTestAuthAgent(t)
	defer os.RemoveAll(tmpDir)

	_ = agent.SaveRole(&Role{Name: "viewer", Permissions: []string{PermissionStatus}})
	_ = agent.CreateUserAccountFull("dave", "pass", "", "viewer", true)

	if err := agent.UnregisterUser("dave"); err != nil {
		t.Fatalf("UnregisterUser selhalo: %v", err)
	}
	if agent.UserExists("dave") {
		t.Fatal("dave měl být smazán")
	}
	if agent.Database.KeyExists("auth", "role/dave") {
		t.Fatal("role záznam měl být smazán")
	}
	if agent.Database.KeyExists("auth", "mustchangepw/dave") {
		t.Fatal("mustchangepw záznam měl být smazán")
	}
}

// TestPermissionForPath ověří mapování endpointů na oprávnění.
func TestPermissionForPath(t *testing.T) {
	agent, _ := newTestAuthAgent(t)

	cases := map[string]string{
		"/api/users/list":             PermissionUsers,
		"/api/proxy/list":             PermissionReverseProxy,
		"/api/proxy/setIncoming":      PermissionIngress,
		"/api/cert/list":              PermissionSecurity,
		"/api/redirect/list":          PermissionAccess,
		"/api/stats/summary":          PermissionStatus,
		"/api/analytic/list":          PermissionStats,
		"/api/webserv/status":         PermissionServices,
		"/api/tools/ping":             PermissionNetworking,
		"/api/plugins/list":           PermissionPlugins,
		"/api/plugins/store/list":     PermissionAppStore,
		"/api/conf/export":            PermissionUtilities,
		"/api/auth/login":             "",
		"/api/streamprox/config/list": PermissionReverseProxy,
	}

	for path, want := range cases {
		got := agent.PermissionForPath(path)
		if got != want {
			t.Errorf("PermissionForPath(%q) = %q, chtěl jsem %q", path, got, want)
		}
	}
}
