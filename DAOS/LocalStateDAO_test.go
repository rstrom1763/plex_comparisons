package DAOS

import (
	"database/sql"
	"path/filepath"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/rstrom1763/plex_comparisons/structs"
)

func newTestDAO(t *testing.T) *LocalStateDAO {
	t.Helper()

	dao, err := NewLocalStateDAO(filepath.Join(t.TempDir(), "local_state.db"))
	if err != nil {
		t.Fatalf("NewLocalStateDAO() error = %v", err)
	}
	t.Cleanup(func() {
		if err := dao.Close(); err != nil {
			t.Errorf("Close() error = %v", err)
		}
	})

	return dao
}

func TestLocalStateDAOStoresServers(t *testing.T) {
	dao := newTestDAO(t)

	if err := dao.AddServer(structs.Server{Name: "Remote", Address: "https://remote.example", Token: "token"}); err != nil {
		t.Fatalf("AddServer() error = %v", err)
	}

	servers, err := dao.GetServers()
	if err != nil {
		t.Fatalf("GetServers() error = %v", err)
	}
	if len(servers) != 1 {
		t.Fatalf("len(servers) = %d, want 1", len(servers))
	}
	if servers[0].Name != "Remote" || servers[0].Address != "https://remote.example" || servers[0].Token != "token" {
		t.Fatalf("server = %+v, want inserted values", servers[0])
	}

	servers[0].Name = "Updated"
	servers[0].Token = "updated-token"
	if err := dao.UpdateServer(servers[0]); err != nil {
		t.Fatalf("UpdateServer() error = %v", err)
	}

	servers, err = dao.GetServers()
	if err != nil {
		t.Fatalf("GetServers() after update error = %v", err)
	}
	if servers[0].Name != "Updated" || servers[0].Token != "updated-token" {
		t.Fatalf("updated server = %+v, want updated values", servers[0])
	}

	if err := dao.DeleteServer(servers[0].ID); err != nil {
		t.Fatalf("DeleteServer() error = %v", err)
	}
	servers, err = dao.GetServers()
	if err != nil {
		t.Fatalf("GetServers() after delete error = %v", err)
	}
	if len(servers) != 0 {
		t.Fatalf("len(servers) = %d after delete, want 0", len(servers))
	}
}

func TestLocalStateDAOStoresSavedFilters(t *testing.T) {
	dao := newTestDAO(t)

	id, err := dao.AddSavedFilter(structs.SavedFilter{Name: "Large movies", FilterData: "eyJtaW4iOjEwfQ=="})
	if err != nil {
		t.Fatalf("AddSavedFilter() error = %v", err)
	}
	if id == 0 {
		t.Fatal("AddSavedFilter() id = 0, want non-zero")
	}

	filters, err := dao.GetSavedFilters()
	if err != nil {
		t.Fatalf("GetSavedFilters() error = %v", err)
	}
	if len(filters) != 1 {
		t.Fatalf("len(filters) = %d, want 1", len(filters))
	}
	if filters[0].Name != "Large movies" || filters[0].FilterData != "eyJtaW4iOjEwfQ==" {
		t.Fatalf("filter = %+v, want inserted values", filters[0])
	}

	filters[0].Name = "Updated filter"
	if err := dao.UpdateSavedFilter(filters[0]); err != nil {
		t.Fatalf("UpdateSavedFilter() error = %v", err)
	}
	filters, err = dao.GetSavedFilters()
	if err != nil {
		t.Fatalf("GetSavedFilters() after update error = %v", err)
	}
	if filters[0].Name != "Updated filter" {
		t.Fatalf("updated filter name = %q, want %q", filters[0].Name, "Updated filter")
	}

	if err := dao.DeleteSavedFilter(filters[0].ID); err != nil {
		t.Fatalf("DeleteSavedFilter() error = %v", err)
	}
	filters, err = dao.GetSavedFilters()
	if err != nil {
		t.Fatalf("GetSavedFilters() after delete error = %v", err)
	}
	if len(filters) != 0 {
		t.Fatalf("len(filters) = %d after delete, want 0", len(filters))
	}
}

func TestLocalStateDAOStoresTrustedServers(t *testing.T) {
	dao := newTestDAO(t)

	if err := dao.AddTrustedServer("remote", "hashed-token"); err != nil {
		t.Fatalf("AddTrustedServer() error = %v", err)
	}

	servers, err := dao.GetTrustedServers()
	if err != nil {
		t.Fatalf("GetTrustedServers() error = %v", err)
	}
	if len(servers) != 1 {
		t.Fatalf("len(servers) = %d, want 1", len(servers))
	}
	if servers[0].Name != "remote" || servers[0].TokenHash != "hashed-token" {
		t.Fatalf("trusted server = %+v, want inserted values", servers[0])
	}

	server, err := dao.GetTrustedServerByName("remote")
	if err != nil {
		t.Fatalf("GetTrustedServerByName() error = %v", err)
	}
	if server == nil {
		t.Fatal("GetTrustedServerByName() = nil, want server")
	}
	if server.ID != servers[0].ID {
		t.Fatalf("GetTrustedServerByName() ID = %d, want %d", server.ID, servers[0].ID)
	}

	missing, err := dao.GetTrustedServerByName("missing")
	if err != nil {
		t.Fatalf("GetTrustedServerByName(missing) error = %v", err)
	}
	if missing != nil {
		t.Fatalf("GetTrustedServerByName(missing) = %+v, want nil", missing)
	}

	if err := dao.DeleteTrustedServer(servers[0].ID); err != nil {
		t.Fatalf("DeleteTrustedServer() error = %v", err)
	}
	servers, err = dao.GetTrustedServers()
	if err != nil {
		t.Fatalf("GetTrustedServers() after delete error = %v", err)
	}
	if len(servers) != 0 {
		t.Fatalf("len(servers) = %d after delete, want 0", len(servers))
	}
}

func TestLocalStateDAOStoresUsers(t *testing.T) {
	dao := newTestDAO(t)

	count, err := dao.GetUserCount()
	if err != nil {
		t.Fatalf("GetUserCount() initial error = %v", err)
	}
	if count != 0 {
		t.Fatalf("initial user count = %d, want 0", count)
	}

	if err := dao.AddUser("ryan", "password-hash"); err != nil {
		t.Fatalf("AddUser() error = %v", err)
	}

	count, err = dao.GetUserCount()
	if err != nil {
		t.Fatalf("GetUserCount() after add error = %v", err)
	}
	if count != 1 {
		t.Fatalf("user count = %d, want 1", count)
	}

	user, err := dao.GetUserByUsername("ryan")
	if err != nil {
		t.Fatalf("GetUserByUsername() error = %v", err)
	}
	if user == nil {
		t.Fatal("GetUserByUsername() = nil, want user")
	}
	if user.Username != "ryan" || user.PasswordHash != "password-hash" {
		t.Fatalf("user = %+v, want inserted values", user)
	}

	missing, err := dao.GetUserByUsername("missing")
	if err != nil {
		t.Fatalf("GetUserByUsername(missing) error = %v", err)
	}
	if missing != nil {
		t.Fatalf("GetUserByUsername(missing) = %+v, want nil", missing)
	}
}

func TestNewLocalStateDAOReturnsErrorForInaccessiblePath(t *testing.T) {
	if _, err := NewLocalStateDAO(t.TempDir()); err == nil {
		t.Fatal("NewLocalStateDAO() error = nil, want error")
	}
}

func TestNewLocalStateDAOReturnsOpenError(t *testing.T) {
	if _, err := newLocalStateDAO(filepath.Join(t.TempDir(), "local_state.db"), "missing-driver"); err == nil {
		t.Fatal("newLocalStateDAO() error = nil, want open error")
	}
}

func TestLocalStateDAOClosedDatabaseReturnsErrors(t *testing.T) {
	dao := newTestDAO(t)
	if err := dao.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	if err := dao.AddServer(structs.Server{Name: "Remote", Address: "https://remote.example"}); err == nil {
		t.Fatal("AddServer() error = nil after Close(), want error")
	}
	if _, err := dao.GetServers(); err == nil {
		t.Fatal("GetServers() error = nil after Close(), want error")
	}
	if err := dao.DeleteServer(1); err == nil {
		t.Fatal("DeleteServer() error = nil after Close(), want error")
	}
	if err := dao.UpdateServer(structs.Server{ID: 1, Name: "Remote", Address: "https://remote.example"}); err == nil {
		t.Fatal("UpdateServer() error = nil after Close(), want error")
	}
	if _, err := dao.AddSavedFilter(structs.SavedFilter{Name: "Filter", FilterData: "{}"}); err == nil {
		t.Fatal("AddSavedFilter() error = nil after Close(), want error")
	}
	if _, err := dao.GetSavedFilters(); err == nil {
		t.Fatal("GetSavedFilters() error = nil after Close(), want error")
	}
	if err := dao.DeleteSavedFilter(1); err == nil {
		t.Fatal("DeleteSavedFilter() error = nil after Close(), want error")
	}
	if err := dao.UpdateSavedFilter(structs.SavedFilter{ID: 1, Name: "Filter", FilterData: "{}"}); err == nil {
		t.Fatal("UpdateSavedFilter() error = nil after Close(), want error")
	}
	if err := dao.AddTrustedServer("remote", "hash"); err == nil {
		t.Fatal("AddTrustedServer() error = nil after Close(), want error")
	}
	if _, err := dao.GetTrustedServers(); err == nil {
		t.Fatal("GetTrustedServers() error = nil after Close(), want error")
	}
	if _, err := dao.GetTrustedServerByName("remote"); err == nil {
		t.Fatal("GetTrustedServerByName() error = nil after Close(), want error")
	}
	if err := dao.DeleteTrustedServer(1); err == nil {
		t.Fatal("DeleteTrustedServer() error = nil after Close(), want error")
	}
	if err := dao.AddUser("ryan", "hash"); err == nil {
		t.Fatal("AddUser() error = nil after Close(), want error")
	}
	if _, err := dao.GetUserByUsername("ryan"); err == nil {
		t.Fatal("GetUserByUsername() error = nil after Close(), want error")
	}
	if _, err := dao.GetUserCount(); err == nil {
		t.Fatal("GetUserCount() error = nil after Close(), want error")
	}
}

func TestNewLocalStateDAOReturnsTableSetupError(t *testing.T) {
	originalTables := localStateTables
	localStateTables = []localStateTable{{name: "broken", sql: "not valid sql"}}
	t.Cleanup(func() {
		localStateTables = originalTables
	})

	if _, err := NewLocalStateDAO(filepath.Join(t.TempDir(), "local_state.db")); err == nil {
		t.Fatal("NewLocalStateDAO() error = nil, want table setup error")
	}
}

func TestEnsureLocalStateTablesAttemptsAllTables(t *testing.T) {
	originalTables := localStateTables
	localStateTables = []localStateTable{
		{name: "broken", sql: "not valid sql"},
		{name: "created later", sql: "CREATE TABLE IF NOT EXISTS created_later (id INTEGER PRIMARY KEY);"},
	}
	t.Cleanup(func() {
		localStateTables = originalTables
	})

	db, err := sql.Open("sqlite3", filepath.Join(t.TempDir(), "local_state.db"))
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("Close() error = %v", err)
		}
	})

	if err := ensureLocalStateTables(db); err == nil {
		t.Fatal("ensureLocalStateTables() error = nil, want error")
	}
	if _, err := db.Exec("INSERT INTO created_later DEFAULT VALUES;"); err != nil {
		t.Fatalf("created_later table was not created: %v", err)
	}
}

func TestLocalStateDAOScanErrors(t *testing.T) {
	t.Run("servers", func(t *testing.T) {
		dao := newDAOWithSchema(t, `CREATE TABLE servers (id TEXT, name TEXT, address TEXT, token TEXT);`, `INSERT INTO servers VALUES ('bad', 'n', 'a', 't');`)
		if _, err := dao.GetServers(); err == nil {
			t.Fatal("GetServers() error = nil, want scan error")
		}
	})

	t.Run("saved filters", func(t *testing.T) {
		dao := newDAOWithSchema(t, `CREATE TABLE saved_filters (id TEXT, name TEXT, filter_data TEXT);`, `INSERT INTO saved_filters VALUES ('bad', 'n', 'd');`)
		if _, err := dao.GetSavedFilters(); err == nil {
			t.Fatal("GetSavedFilters() error = nil, want scan error")
		}
	})

	t.Run("trusted servers", func(t *testing.T) {
		dao := newDAOWithSchema(t, `CREATE TABLE trusted_servers (id TEXT, name TEXT, token_hash TEXT);`, `INSERT INTO trusted_servers VALUES ('bad', 'n', 'h');`)
		if _, err := dao.GetTrustedServers(); err == nil {
			t.Fatal("GetTrustedServers() error = nil, want scan error")
		}
	})
}

func newDAOWithSchema(t *testing.T, statements ...string) *LocalStateDAO {
	t.Helper()

	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}
	for _, stmt := range statements {
		if _, err := db.Exec(stmt); err != nil {
			t.Fatalf("Exec() error = %v", err)
		}
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("Close() error = %v", err)
		}
	})

	return &LocalStateDAO{db: db}
}
