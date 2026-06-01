package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"github.com/rstrom1763/plex_comparisons/DAOS"
	"github.com/rstrom1763/plex_comparisons/auth"
	"github.com/rstrom1763/plex_comparisons/structs"
)

func newTestLocalDAO(t *testing.T) *DAOS.LocalStateDAO {
	t.Helper()

	dao, err := DAOS.NewLocalStateDAO(filepath.Join(t.TempDir(), "local_state.db"))
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

func newAuthTestRouter(dao *DAOS.LocalStateDAO) *gin.Engine {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(AuthMiddleware(dao))
	router.GET("/api/protected", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	router.GET("/dashboard", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	router.POST("/api/protected", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	router.GET("/login", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	return router
}

func TestAuthMiddlewareAllowsTrustedServerToken(t *testing.T) {
	t.Setenv("PROTOCOL", "http")
	dao := newTestLocalDAO(t)

	const sharedToken = "shared-token"
	tokenHash, err := auth.HashSecret(sharedToken)
	if err != nil {
		t.Fatalf("HashSecret() error = %v", err)
	}
	if err := dao.AddTrustedServer("remote", tokenHash); err != nil {
		t.Fatalf("AddTrustedServer() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/protected", nil)
	req.Header.Set("X-Server-Token", sharedToken)
	rec := httptest.NewRecorder()

	newAuthTestRouter(dao).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestAuthMiddlewareRejectsUnauthenticatedAPIRequest(t *testing.T) {
	t.Setenv("PROTOCOL", "http")
	dao := newTestLocalDAO(t)

	req := httptest.NewRequest(http.MethodGet, "/api/protected", nil)
	rec := httptest.NewRecorder()

	newAuthTestRouter(dao).ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestAuthMiddlewareRedirectsUnauthenticatedUIRequestToLogin(t *testing.T) {
	t.Setenv("PROTOCOL", "http")
	dao := newTestLocalDAO(t)

	req := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
	rec := httptest.NewRecorder()

	newAuthTestRouter(dao).ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusSeeOther)
	}
	if location := rec.Header().Get("Location"); location != "/login" {
		t.Fatalf("Location = %q, want %q", location, "/login")
	}
}

func TestAuthMiddlewareAllowsLoginRouteWithoutSession(t *testing.T) {
	t.Setenv("PROTOCOL", "http")
	dao := newTestLocalDAO(t)

	req := httptest.NewRequest(http.MethodGet, "/login", nil)
	rec := httptest.NewRecorder()

	newAuthTestRouter(dao).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestAuthMiddlewareRequiresCSRFForSessionPOST(t *testing.T) {
	t.Setenv("PROTOCOL", "http")
	dao := newTestLocalDAO(t)
	sessionToken, err := auth.CreateSession("ryan")
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}
	t.Cleanup(func() {
		auth.DeleteSession(sessionToken)
	})

	req := httptest.NewRequest(http.MethodPost, "/api/protected", nil)
	req.AddCookie(&http.Cookie{Name: "session_token", Value: sessionToken})
	rec := httptest.NewRecorder()

	newAuthTestRouter(dao).ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
}

func TestAuthMiddlewareAllowsSessionPOSTWithMatchingCSRF(t *testing.T) {
	t.Setenv("PROTOCOL", "http")
	dao := newTestLocalDAO(t)
	sessionToken, err := auth.CreateSession("ryan")
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}
	t.Cleanup(func() {
		auth.DeleteSession(sessionToken)
	})

	req := httptest.NewRequest(http.MethodPost, "/api/protected", nil)
	req.AddCookie(&http.Cookie{Name: "session_token", Value: sessionToken})
	req.AddCookie(&http.Cookie{Name: "csrf_token", Value: "csrf-token"})
	req.Header.Set("X-CSRF-Token", "csrf-token")
	rec := httptest.NewRecorder()

	newAuthTestRouter(dao).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestAuthMiddlewareRefreshesCSRFCookie(t *testing.T) {
	t.Setenv("PROTOCOL", "http")
	dao := newTestLocalDAO(t)
	sessionToken, err := auth.CreateSession("ryan")
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}
	t.Cleanup(func() {
		auth.DeleteSession(sessionToken)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/protected", nil)
	req.AddCookie(&http.Cookie{Name: "session_token", Value: sessionToken})
	req.AddCookie(&http.Cookie{Name: "csrf_token", Value: "csrf-token"})
	rec := httptest.NewRecorder()

	newAuthTestRouter(dao).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	cookies := rec.Result().Cookies()
	if !hasCookie(cookies, "session_token", sessionToken) {
		t.Fatalf("session_token cookie was not refreshed: %v", cookies)
	}
	if !hasCookie(cookies, "csrf_token", "csrf-token") {
		t.Fatalf("csrf_token cookie was not refreshed: %v", cookies)
	}
}

func TestAuthMiddlewareRegeneratesMissingCSRFOnSafeRequest(t *testing.T) {
	t.Setenv("PROTOCOL", "http")
	dao := newTestLocalDAO(t)
	sessionToken, err := auth.CreateSession("ryan")
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}
	t.Cleanup(func() {
		auth.DeleteSession(sessionToken)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/protected", nil)
	req.AddCookie(&http.Cookie{Name: "session_token", Value: sessionToken})
	rec := httptest.NewRecorder()

	newAuthTestRouter(dao).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if !hasCookieWithValue(cookiesByName(rec.Result().Cookies()), "csrf_token") {
		t.Fatalf("csrf_token cookie was not regenerated: %v", rec.Result().Cookies())
	}
}

func TestPlexMoviePosterDir(t *testing.T) {
	got := plexMoviePosterDir(filepath.Join("plex", "data"), "abcdef")
	want := filepath.Join("plex", "data", "Metadata", "Movies", "a", "bcdef.bundle", "Contents", "_combined", "posters")

	if got != want {
		t.Fatalf("plexMoviePosterDir() = %q, want %q", got, want)
	}
}

func TestPlexMoviePosterDirEmptyHash(t *testing.T) {
	if got := plexMoviePosterDir("plex-data", ""); got != "" {
		t.Fatalf("plexMoviePosterDir() = %q, want empty string", got)
	}
}

func TestPlexMoviePosterDirShortHash(t *testing.T) {
	if got := plexMoviePosterDir("plex-data", "a"); got != "" {
		t.Fatalf("plexMoviePosterDir() = %q, want empty string", got)
	}
}

func TestFindPosterFileReturnsLargestFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "nested"), 0755); err != nil {
		t.Fatalf("Mkdir() error = %v", err)
	}
	files := map[string]string{
		"small.jpg":  "small",
		"large.jpg":  "this is the largest poster",
		"medium.jpg": "medium poster",
	}
	for name, contents := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(contents), 0644); err != nil {
			t.Fatalf("WriteFile(%q) error = %v", name, err)
		}
	}

	got, err := findPosterFile(dir)
	if err != nil {
		t.Fatalf("findPosterFile() error = %v", err)
	}
	want := filepath.Join(dir, "large.jpg")

	if got != want {
		t.Fatalf("findPosterFile() = %q, want %q", got, want)
	}
}

func TestFindPosterFileReturnsErrorWhenDirectoryIsEmpty(t *testing.T) {
	_, err := findPosterFile(t.TempDir())
	if err == nil {
		t.Fatal("findPosterFile() error = nil, want error")
	}
}

func TestFindPosterFileReturnsReadDirError(t *testing.T) {
	_, err := findPosterFile(filepath.Join(t.TempDir(), "missing"))
	if err == nil {
		t.Fatal("findPosterFile() error = nil, want error")
	}
}

func TestServerMediaHandlers(t *testing.T) {
	db := newServerFixtureDB(t)

	cases := []struct {
		name        string
		path        string
		handler     gin.HandlerFunc
		wantStatus  int
		wantHeader  string
		wantContent string
	}{
		{name: "movies json", path: "/api/movies", handler: compressedMoviesHandler(db), wantStatus: http.StatusOK, wantHeader: "gzip"},
		{name: "episodes json", path: "/api/episodes", handler: compressedEpisodesHandler(db), wantStatus: http.StatusOK, wantHeader: "gzip"},
		{name: "songs json", path: "/api/songs", handler: compressedSongsHandler(db), wantStatus: http.StatusOK, wantHeader: "gzip"},
		{name: "movies csv", path: "/api/downloads/movies", handler: movieDownloadHandler(db), wantStatus: http.StatusOK, wantContent: "Alien"},
		{name: "episodes csv", path: "/api/downloads/episodes", handler: episodeDownloadHandler(db), wantStatus: http.StatusOK, wantContent: "Episode"},
		{name: "songs csv", path: "/api/downloads/songs", handler: songDownloadHandler(db), wantStatus: http.StatusOK, wantContent: "Song"},
		{name: "duplicates", path: "/api/duplicates", handler: duplicatesHandler(db), wantStatus: http.StatusOK, wantContent: "null"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := serveServerHandler(http.MethodGet, tc.path, tc.path, tc.handler, nil)
			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d; body = %s", rec.Code, tc.wantStatus, rec.Body.String())
			}
			if tc.wantHeader != "" && rec.Header().Get("Content-Encoding") != tc.wantHeader {
				t.Fatalf("Content-Encoding = %q, want %q", rec.Header().Get("Content-Encoding"), tc.wantHeader)
			}
			if tc.wantContent != "" && !bytes.Contains(rec.Body.Bytes(), []byte(tc.wantContent)) {
				t.Fatalf("body = %q, want content %q", rec.Body.String(), tc.wantContent)
			}
		})
	}
}

func TestDuplicatesHandlerReturnsDuplicateGroups(t *testing.T) {
	db := newServerFixtureDB(t)
	if _, err := db.Exec(`
		INSERT INTO metadata_items (
			id, metadata_type, title, content_rating, year, tags_genre,
			rating, audience_rating, hash
		) VALUES (
			11, 1, 'Alien', 'R', 1979, 'Horror', 8.5, 9.1, 'movie-metadata-2'
		);
		INSERT INTO media_items (
			id, metadata_item_id, library_section_id, container, bitrate,
			video_codec, height, width, audio_codec
		) VALUES (
			11, 11, 1, 'mkv', 5000, 'h264', 1080, 1920, 'aac'
		);
		INSERT INTO media_parts (id, media_item_id, file, hash, size, duration)
			VALUES (11, 11, '/media/Alien2.mkv', 'movie-file-2', 123456, 7000000);
	`); err != nil {
		t.Fatalf("insert duplicate fixture error = %v", err)
	}

	rec := serveServerHandler(http.MethodGet, "/api/duplicates", "/api/duplicates", duplicatesHandler(db), nil)
	if rec.Code != http.StatusOK || !bytes.Contains(rec.Body.Bytes(), []byte("Alien")) {
		t.Fatalf("duplicates response = %d %q, want Alien duplicate", rec.Code, rec.Body.String())
	}
}

func TestLoginSetupAndLogoutHandlers(t *testing.T) {
	dao := newTestLocalDAO(t)
	userCount := 0

	rec := serveServerHandler(http.MethodGet, "/login", "/login", loginPageHandler(&userCount), nil)
	if rec.Code != http.StatusOK || !bytes.Contains(rec.Body.Bytes(), []byte("setup")) {
		t.Fatalf("setup page response = %d %q", rec.Code, rec.Body.String())
	}
	userCount = 1
	rec = serveServerHandler(http.MethodGet, "/login", "/login", loginPageHandler(&userCount), nil)
	if rec.Code != http.StatusOK || !bytes.Contains(rec.Body.Bytes(), []byte("login")) {
		t.Fatalf("login page response = %d %q", rec.Code, rec.Body.String())
	}

	userCount = 0
	rec = serveServerHandler(http.MethodPost, "/setup", "/setup", setupHandler(dao, &userCount), []byte(`{`))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("bad setup status = %d, want 400", rec.Code)
	}
	rec = serveServerHandler(http.MethodPost, "/setup", "/setup", setupHandler(dao, &userCount), []byte(`{"username":"ryan","password":"secret"}`))
	if rec.Code != http.StatusOK || userCount != 1 {
		t.Fatalf("setup response = %d userCount=%d, want success", rec.Code, userCount)
	}
	rec = serveServerHandler(http.MethodPost, "/setup", "/setup", setupHandler(dao, &userCount), []byte(`{"username":"ryan","password":"secret"}`))
	if rec.Code != http.StatusForbidden {
		t.Fatalf("setup completed status = %d, want 403", rec.Code)
	}

	rec = serveServerHandler(http.MethodPost, "/login", "/login", loginSubmitHandler(dao), []byte(`{`))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("bad login status = %d, want 400", rec.Code)
	}
	rec = serveServerHandler(http.MethodPost, "/login", "/login", loginSubmitHandler(dao), []byte(`{"username":"ryan","password":"wrong"}`))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("wrong login status = %d, want 401", rec.Code)
	}
	t.Setenv("PROTOCOL", "https")
	rec = serveServerHandler(http.MethodPost, "/login", "/login", loginSubmitHandler(dao), []byte(`{"username":"ryan","password":"secret"}`))
	if rec.Code != http.StatusOK || len(rec.Result().Cookies()) == 0 {
		t.Fatalf("login success response = %d cookies=%v", rec.Code, rec.Result().Cookies())
	}

	sessionToken, err := auth.CreateSession("ryan")
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}
	reqBody := []byte{}
	rec = serveServerHandlerWithCookies(http.MethodPost, "/logout", "/logout", logoutHandler, reqBody, []*http.Cookie{{Name: "session_token", Value: sessionToken}})
	if rec.Code != http.StatusOK {
		t.Fatalf("logout status = %d, want 200", rec.Code)
	}
	if _, ok := auth.ValidateSession(sessionToken); ok {
		t.Fatal("session still valid after logout")
	}

	closedDAO := newTestLocalDAO(t)
	if err := closedDAO.Close(); err != nil {
		t.Fatalf("Close(closedDAO) error = %v", err)
	}
	userCount = 0
	rec = serveServerHandler(http.MethodPost, "/setup", "/setup", setupHandler(closedDAO, &userCount), []byte(`{"username":"x","password":"secret"}`))
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("setup DAO error status = %d, want 500", rec.Code)
	}
}

func TestPageHandlers(t *testing.T) {
	rec := serveServerHandler(http.MethodGet, "/", "/", htmlPageHandler("login.html"), nil)
	if rec.Code != http.StatusOK || !bytes.Contains(rec.Body.Bytes(), []byte("login")) {
		t.Fatalf("html handler response = %d %q", rec.Code, rec.Body.String())
	}

	rec = serveServerHandler(http.MethodGet, "/", "/", htmlPageHandler("index.html"), nil)
	if rec.Code != http.StatusOK || rec.Header().Get("Location") != "" || !bytes.Contains(rec.Body.Bytes(), []byte("Movie Gallery")) {
		t.Fatalf("index handler response = %d location=%q body=%q, want embedded index without redirect", rec.Code, rec.Header().Get("Location"), rec.Body.String())
	}
}

func TestRegisterEmbeddedAssetsServesStaticFilesAndHTML(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	if err := registerEmbeddedAssets(router); err != nil {
		t.Fatalf("registerEmbeddedAssets() error = %v", err)
	}
	router.GET("/compare", htmlPageHandler("compare.html"))

	req := httptest.NewRequest(http.MethodGet, "/static/css/styles.css", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || !bytes.Contains(rec.Body.Bytes(), []byte("body")) {
		t.Fatalf("embedded static response = %d %q, want stylesheet", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/compare", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || !bytes.Contains(rec.Body.Bytes(), []byte("Compare Servers")) {
		t.Fatalf("embedded HTML response = %d %q, want compare page", rec.Code, rec.Body.String())
	}
}

func TestServerLocalStateHandlers(t *testing.T) {
	dao := newTestLocalDAO(t)

	rec := serveServerHandler(http.MethodGet, "/ping", "/ping", pingHandler, nil)
	if rec.Code != http.StatusOK || !bytes.Contains(rec.Body.Bytes(), []byte("pong")) {
		t.Fatalf("ping response = %d %q, want pong", rec.Code, rec.Body.String())
	}

	rec = serveServerHandler(http.MethodPost, "/api/trusted-servers", "/api/trusted-servers", createTrustedServerHandler(dao), []byte(`{"name":"remote"}`))
	if rec.Code != http.StatusOK {
		t.Fatalf("create trusted status = %d, want 200; body = %s", rec.Code, rec.Body.String())
	}
	var tokenResp struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &tokenResp); err != nil {
		t.Fatalf("trusted token JSON error = %v", err)
	}
	if tokenResp.Token == "" {
		t.Fatal("trusted token is empty")
	}
	rec = serveServerHandler(http.MethodGet, "/api/trusted-servers", "/api/trusted-servers", getTrustedServersHandler(dao), nil)
	if rec.Code != http.StatusOK || !bytes.Contains(rec.Body.Bytes(), []byte("remote")) {
		t.Fatalf("trusted list response = %d %q, want remote", rec.Code, rec.Body.String())
	}
	rec = serveServerHandler(http.MethodDelete, "/api/trusted-servers/:id", "/api/trusted-servers/1", deleteTrustedServerHandler(dao), nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("delete trusted status = %d, want 200", rec.Code)
	}

	rec = serveServerHandler(http.MethodPost, "/api/servers", "/api/servers", createServerHandler(dao), []byte(`{"name":"Remote","address":"http://remote","token":"token"}`))
	if rec.Code != http.StatusCreated {
		t.Fatalf("create server status = %d, want 201; body = %s", rec.Code, rec.Body.String())
	}
	rec = serveServerHandler(http.MethodGet, "/api/servers", "/api/servers", getServersHandler(dao), nil)
	if rec.Code != http.StatusOK || !bytes.Contains(rec.Body.Bytes(), []byte("Remote")) {
		t.Fatalf("server list response = %d %q, want Remote", rec.Code, rec.Body.String())
	}
	rec = serveServerHandler(http.MethodPut, "/api/servers/:id", "/api/servers/1", updateServerHandler(dao), []byte(`{"name":"Updated","address":"http://remote","token":"token2"}`))
	if rec.Code != http.StatusOK {
		t.Fatalf("update server status = %d, want 200; body = %s", rec.Code, rec.Body.String())
	}
	rec = serveServerHandler(http.MethodDelete, "/api/servers/:id", "/api/servers/1", deleteServerHandler(dao), nil)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("delete server status = %d, want 204", rec.Code)
	}

	rec = serveServerHandler(http.MethodPost, "/api/filters", "/api/filters", createFilterHandler(dao), []byte(`{"name":"Large","filter_data":"{}"}`))
	if rec.Code != http.StatusCreated {
		t.Fatalf("create filter status = %d, want 201; body = %s", rec.Code, rec.Body.String())
	}
	rec = serveServerHandler(http.MethodGet, "/api/filters", "/api/filters", getFiltersHandler(dao), nil)
	if rec.Code != http.StatusOK || !bytes.Contains(rec.Body.Bytes(), []byte("Large")) {
		t.Fatalf("filter list response = %d %q, want Large", rec.Code, rec.Body.String())
	}
	rec = serveServerHandler(http.MethodPut, "/api/filters/:id", "/api/filters/1", updateFilterHandler(dao), []byte(`{"name":"Updated","filter_data":"{}"}`))
	if rec.Code != http.StatusOK {
		t.Fatalf("update filter status = %d, want 200; body = %s", rec.Code, rec.Body.String())
	}
	rec = serveServerHandler(http.MethodDelete, "/api/filters/:id", "/api/filters/1", deleteFilterHandler(dao), nil)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("delete filter status = %d, want 204", rec.Code)
	}
}

func TestServerHandlersValidateBadRequests(t *testing.T) {
	dao := newTestLocalDAO(t)

	cases := []struct {
		name       string
		method     string
		route      string
		target     string
		handler    gin.HandlerFunc
		body       []byte
		wantStatus int
	}{
		{name: "create trusted bad json", method: http.MethodPost, route: "/api/trusted-servers", target: "/api/trusted-servers", handler: createTrustedServerHandler(dao), body: []byte(`{`), wantStatus: http.StatusBadRequest},
		{name: "create server bad json", method: http.MethodPost, route: "/api/servers", target: "/api/servers", handler: createServerHandler(dao), body: []byte(`{`), wantStatus: http.StatusBadRequest},
		{name: "create server missing token", method: http.MethodPost, route: "/api/servers", target: "/api/servers", handler: createServerHandler(dao), body: []byte(`{"name":"Remote","address":"http://remote"}`), wantStatus: http.StatusBadRequest},
		{name: "update server bad json", method: http.MethodPut, route: "/api/servers/:id", target: "/api/servers/1", handler: updateServerHandler(dao), body: []byte(`{`), wantStatus: http.StatusBadRequest},
		{name: "update server missing token", method: http.MethodPut, route: "/api/servers/:id", target: "/api/servers/1", handler: updateServerHandler(dao), body: []byte(`{"name":"Remote","address":"http://remote"}`), wantStatus: http.StatusBadRequest},
		{name: "create filter bad json", method: http.MethodPost, route: "/api/filters", target: "/api/filters", handler: createFilterHandler(dao), body: []byte(`{`), wantStatus: http.StatusBadRequest},
		{name: "update filter bad json", method: http.MethodPut, route: "/api/filters/:id", target: "/api/filters/1", handler: updateFilterHandler(dao), body: []byte(`{`), wantStatus: http.StatusBadRequest},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := serveServerHandler(tc.method, tc.route, tc.target, tc.handler, tc.body)
			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d; body = %s", rec.Code, tc.wantStatus, rec.Body.String())
			}
		})
	}
}

func TestServerHandlersReturnInternalErrors(t *testing.T) {
	dao, err := DAOS.NewLocalStateDAO(filepath.Join(t.TempDir(), "local_state.db"))
	if err != nil {
		t.Fatalf("NewLocalStateDAO() error = %v", err)
	}
	if err := dao.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	daoCases := []struct {
		name    string
		method  string
		route   string
		target  string
		handler gin.HandlerFunc
		body    []byte
	}{
		{name: "get trusted", method: http.MethodGet, route: "/api/trusted-servers", target: "/api/trusted-servers", handler: getTrustedServersHandler(dao)},
		{name: "create trusted", method: http.MethodPost, route: "/api/trusted-servers", target: "/api/trusted-servers", handler: createTrustedServerHandler(dao), body: []byte(`{"name":"remote"}`)},
		{name: "delete trusted", method: http.MethodDelete, route: "/api/trusted-servers/:id", target: "/api/trusted-servers/1", handler: deleteTrustedServerHandler(dao)},
		{name: "get servers", method: http.MethodGet, route: "/api/servers", target: "/api/servers", handler: getServersHandler(dao)},
		{name: "create server", method: http.MethodPost, route: "/api/servers", target: "/api/servers", handler: createServerHandler(dao), body: []byte(`{"name":"Remote","address":"http://remote","token":"token"}`)},
		{name: "update server", method: http.MethodPut, route: "/api/servers/:id", target: "/api/servers/1", handler: updateServerHandler(dao), body: []byte(`{"name":"Remote","address":"http://remote","token":"token"}`)},
		{name: "delete server", method: http.MethodDelete, route: "/api/servers/:id", target: "/api/servers/1", handler: deleteServerHandler(dao)},
		{name: "get filters", method: http.MethodGet, route: "/api/filters", target: "/api/filters", handler: getFiltersHandler(dao)},
		{name: "create filter", method: http.MethodPost, route: "/api/filters", target: "/api/filters", handler: createFilterHandler(dao), body: []byte(`{"name":"Large","filter_data":"{}"}`)},
		{name: "update filter", method: http.MethodPut, route: "/api/filters/:id", target: "/api/filters/1", handler: updateFilterHandler(dao), body: []byte(`{"name":"Large","filter_data":"{}"}`)},
		{name: "delete filter", method: http.MethodDelete, route: "/api/filters/:id", target: "/api/filters/1", handler: deleteFilterHandler(dao)},
	}

	for _, tc := range daoCases {
		t.Run(tc.name, func(t *testing.T) {
			rec := serveServerHandler(tc.method, tc.route, tc.target, tc.handler, tc.body)
			if rec.Code != http.StatusInternalServerError {
				t.Fatalf("status = %d, want 500; body = %s", rec.Code, rec.Body.String())
			}
		})
	}

	db := newServerFixtureDB(t)
	if err := db.Close(); err != nil {
		t.Fatalf("Close(db) error = %v", err)
	}

	dbCases := []struct {
		name    string
		route   string
		target  string
		handler gin.HandlerFunc
	}{
		{name: "movies json", route: "/api/movies", target: "/api/movies", handler: compressedMoviesHandler(db)},
		{name: "episodes json", route: "/api/episodes", target: "/api/episodes", handler: compressedEpisodesHandler(db)},
		{name: "songs json", route: "/api/songs", target: "/api/songs", handler: compressedSongsHandler(db)},
		{name: "movies csv", route: "/api/downloads/movies", target: "/api/downloads/movies", handler: movieDownloadHandler(db)},
		{name: "episodes csv", route: "/api/downloads/episodes", target: "/api/downloads/episodes", handler: episodeDownloadHandler(db)},
		{name: "songs csv", route: "/api/downloads/songs", target: "/api/downloads/songs", handler: songDownloadHandler(db)},
		{name: "duplicates", route: "/api/duplicates", target: "/api/duplicates", handler: duplicatesHandler(db)},
		{name: "video", route: "/video/:hash", target: "/video/movie-file", handler: videoHandler(db)},
	}

	for _, tc := range dbCases {
		t.Run(tc.name, func(t *testing.T) {
			rec := serveServerHandler(http.MethodGet, tc.route, tc.target, tc.handler, nil)
			if rec.Code != http.StatusInternalServerError {
				t.Fatalf("status = %d, want 500; body = %s", rec.Code, rec.Body.String())
			}
		})
	}
}

func TestThumbHandler(t *testing.T) {
	rec := serveHandlerWithParams(thumbHandler("plex-data"), gin.Params{{Key: "hash", Value: ""}})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("empty hash status = %d, want 400", rec.Code)
	}

	rec = serveServerHandler(http.MethodGet, "/thumb/:hash", "/thumb/abcdef", thumbHandler(""), nil)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("empty plex path status = %d, want 500", rec.Code)
	}

	rec = serveServerHandler(http.MethodGet, "/thumb/:hash", "/thumb/abcdef", thumbHandler(t.TempDir()), nil)
	if rec.Code != http.StatusOK || rec.Header().Get("Content-Type") != "image/svg+xml" {
		t.Fatalf("missing poster response = %d %q, want svg placeholder", rec.Code, rec.Header().Get("Content-Type"))
	}

	plexData := t.TempDir()
	posterDir := plexMoviePosterDir(plexData, "abcdef")
	if err := os.MkdirAll(posterDir, 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(posterDir, "poster.jpg"), []byte("poster"), 0644); err != nil {
		t.Fatalf("WriteFile(poster) error = %v", err)
	}

	rec = serveServerHandler(http.MethodGet, "/thumb/:hash", "/thumb/abcdef", thumbHandler(plexData), nil)
	if rec.Code != http.StatusOK || rec.Body.String() != "poster" {
		t.Fatalf("poster response = %d %q, want poster bytes", rec.Code, rec.Body.String())
	}
}

func TestVideoHandler(t *testing.T) {
	db := newServerFixtureDB(t)

	rec := serveHandlerWithParams(videoHandler(db), gin.Params{{Key: "hash", Value: ""}})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("empty hash status = %d, want 400", rec.Code)
	}

	rec = serveServerHandler(http.MethodGet, "/video/:hash", "/video/missing", videoHandler(db), nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("missing movie status = %d, want 404", rec.Code)
	}

	rec = serveServerHandler(http.MethodGet, "/video/:hash", "/video/movie-file", videoHandler(db), nil)
	if rec.Code != http.StatusNotFound || !bytes.Contains(rec.Body.Bytes(), []byte("video file not found")) {
		t.Fatalf("missing file response = %d %q, want missing file", rec.Code, rec.Body.String())
	}

	videoPath := filepath.Join(t.TempDir(), "movie.mkv")
	if err := os.WriteFile(videoPath, []byte("video"), 0644); err != nil {
		t.Fatalf("WriteFile(video) error = %v", err)
	}
	db = newSingleMovieDB(t, videoPath, "movie-file")
	rec = serveServerHandler(http.MethodGet, "/video/:hash", "/video/movie-file", videoHandler(db), nil)
	if rec.Code != http.StatusOK || rec.Header().Get("Accept-Ranges") != "bytes" || rec.Body.String() != "video" {
		t.Fatalf("video response = %d range=%q body=%q, want served video", rec.Code, rec.Header().Get("Accept-Ranges"), rec.Body.String())
	}
}

func TestDeleteFileHandler(t *testing.T) {
	db := newDeleteFileDB(t, "")

	rec := serveServerHandler(http.MethodDelete, "/api/file", "/api/file", deleteFileHandler(db), []byte(`{`))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("bad json status = %d, want 400", rec.Code)
	}

	rec = serveServerHandler(http.MethodDelete, "/api/file", "/api/file", deleteFileHandler(db), []byte(`{}`))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("empty request status = %d, want 400", rec.Code)
	}

	rec = serveServerHandler(http.MethodDelete, "/api/file", "/api/file", deleteFileHandler(db), []byte(`{"hash":"missing"}`))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("missing metadata status = %d, want 404", rec.Code)
	}

	missingDiskPath := filepath.Join(t.TempDir(), "missing.mkv")
	db = newDeleteFileDB(t, missingDiskPath)
	rec = serveServerHandler(http.MethodDelete, "/api/file", "/api/file", deleteFileHandler(db), []byte(`{"hash":"file-hash"}`))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("missing disk status = %d, want 404", rec.Code)
	}

	filePath := filepath.Join(t.TempDir(), "delete-me.mkv")
	if err := os.WriteFile(filePath, []byte("delete"), 0644); err != nil {
		t.Fatalf("WriteFile(delete target) error = %v", err)
	}
	db = newDeleteFileDB(t, filePath)
	rec = serveServerHandler(http.MethodDelete, "/api/file", "/api/file", deleteFileHandler(db), []byte(`{"hash":"file-hash"}`))
	if rec.Code != http.StatusOK {
		t.Fatalf("delete file status = %d, want 200; body = %s", rec.Code, rec.Body.String())
	}
	if _, err := os.Stat(filePath); !os.IsNotExist(err) {
		t.Fatalf("deleted file stat error = %v, want not exist", err)
	}

	pathDelete := filepath.Join(t.TempDir(), "delete-by-path.mkv")
	if err := os.WriteFile(pathDelete, []byte("delete"), 0644); err != nil {
		t.Fatalf("WriteFile(path delete target) error = %v", err)
	}
	db = newDeleteFileDB(t, pathDelete)
	body, err := json.Marshal(map[string]string{"path": pathDelete})
	if err != nil {
		t.Fatalf("Marshal(path body) error = %v", err)
	}
	rec = serveServerHandler(http.MethodDelete, "/api/file", "/api/file", deleteFileHandler(db), body)
	if rec.Code != http.StatusOK {
		t.Fatalf("delete by path status = %d, want 200; body = %s", rec.Code, rec.Body.String())
	}

	nonEmptyDir := filepath.Join(t.TempDir(), "non-empty")
	if err := os.Mkdir(nonEmptyDir, 0755); err != nil {
		t.Fatalf("Mkdir(non-empty) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(nonEmptyDir, "child"), []byte("x"), 0644); err != nil {
		t.Fatalf("WriteFile(child) error = %v", err)
	}
	db = newDeleteFileDB(t, nonEmptyDir)
	rec = serveServerHandler(http.MethodDelete, "/api/file", "/api/file", deleteFileHandler(db), []byte(`{"hash":"file-hash"}`))
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("remove error status = %d, want 500; body = %s", rec.Code, rec.Body.String())
	}

	if err := db.Close(); err != nil {
		t.Fatalf("Close(db) error = %v", err)
	}
	rec = serveServerHandler(http.MethodDelete, "/api/file", "/api/file", deleteFileHandler(db), []byte(`{"hash":"file-hash"}`))
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("closed db status = %d, want 500", rec.Code)
	}
}

func TestCompareServerHandler(t *testing.T) {
	dao := newTestLocalDAO(t)
	db := newServerFixtureDB(t)

	rec := serveServerHandler(http.MethodGet, "/api/compare/:id", "/api/compare/1", compareServerHandler(dao, db), nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("missing server status = %d, want 404", rec.Code)
	}

	remote := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("X-Server-Token"); got != "token" {
			t.Fatalf("X-Server-Token = %q, want token", got)
		}
		_ = json.NewEncoder(w).Encode([]*structs.Movie{{Title: "Remote", Year: 2024}})
	}))
	t.Cleanup(remote.Close)
	if err := dao.AddServer(structs.Server{Name: "Remote", Address: remote.URL, Token: "token"}); err != nil {
		t.Fatalf("AddServer() error = %v", err)
	}

	rec = serveServerHandler(http.MethodGet, "/api/compare/:id", "/api/compare/1", compareServerHandler(dao, db), nil)
	if rec.Code != http.StatusOK || !bytes.Contains(rec.Body.Bytes(), []byte("Remote")) || !bytes.Contains(rec.Body.Bytes(), []byte("Alien")) {
		t.Fatalf("compare response = %d %q, want remote and local differences", rec.Code, rec.Body.String())
	}

	unauthorizedRemote := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	t.Cleanup(unauthorizedRemote.Close)
	if err := dao.AddServer(structs.Server{Name: "Unauthorized", Address: unauthorizedRemote.URL, Token: "token"}); err != nil {
		t.Fatalf("AddServer(unauthorized) error = %v", err)
	}
	rec = serveServerHandler(http.MethodGet, "/api/compare/:id", "/api/compare/2", compareServerHandler(dao, db), nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("unauthorized compare status = %d, want 401", rec.Code)
	}

	badJSONRemote := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("not json"))
	}))
	t.Cleanup(badJSONRemote.Close)
	if err := dao.AddServer(structs.Server{Name: "Bad JSON", Address: badJSONRemote.URL, Token: "token"}); err != nil {
		t.Fatalf("AddServer(bad json) error = %v", err)
	}
	rec = serveServerHandler(http.MethodGet, "/api/compare/:id", "/api/compare/3", compareServerHandler(dao, db), nil)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("bad json compare status = %d, want 500", rec.Code)
	}

	if err := dao.AddServer(structs.Server{Name: "Bad URL", Address: "http://[::1", Token: "token"}); err != nil {
		t.Fatalf("AddServer(bad url) error = %v", err)
	}
	rec = serveServerHandler(http.MethodGet, "/api/compare/:id", "/api/compare/4", compareServerHandler(dao, db), nil)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("bad URL compare status = %d, want 500", rec.Code)
	}

	if err := dao.AddServer(structs.Server{Name: "Unreachable", Address: "http://127.0.0.1:1", Token: "token"}); err != nil {
		t.Fatalf("AddServer(unreachable) error = %v", err)
	}
	rec = serveServerHandler(http.MethodGet, "/api/compare/:id", "/api/compare/5", compareServerHandler(dao, db), nil)
	if rec.Code != http.StatusBadGateway {
		t.Fatalf("unreachable compare status = %d, want 502", rec.Code)
	}

	closedDB := newServerFixtureDB(t)
	if err := closedDB.Close(); err != nil {
		t.Fatalf("Close(closedDB) error = %v", err)
	}
	rec = serveServerHandler(http.MethodGet, "/api/compare/:id", "/api/compare/1", compareServerHandler(dao, closedDB), nil)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("local DB compare status = %d, want 500", rec.Code)
	}
}

func TestGetServerMoviesHandler(t *testing.T) {
	dao := newTestLocalDAO(t)

	rec := serveServerHandler(http.MethodGet, "/api/servers/:id/movies", "/api/servers/1/movies", getServerMoviesHandler(dao), nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("missing server status = %d, want 404", rec.Code)
	}

	remote := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/movies" {
			t.Fatalf("remote path = %q, want /api/movies", r.URL.Path)
		}
		if got := r.Header.Get("X-Server-Token"); got != "token" {
			t.Fatalf("X-Server-Token = %q, want token", got)
		}
		_ = json.NewEncoder(w).Encode([]*structs.Movie{{Title: "Remote Gallery", Year: 2026}})
	}))
	t.Cleanup(remote.Close)
	if err := dao.AddServer(structs.Server{Name: "Remote", Address: remote.URL + "/", Token: "token"}); err != nil {
		t.Fatalf("AddServer() error = %v", err)
	}

	rec = serveServerHandler(http.MethodGet, "/api/servers/:id/movies", "/api/servers/1/movies", getServerMoviesHandler(dao), nil)
	if rec.Code != http.StatusOK || !bytes.Contains(rec.Body.Bytes(), []byte("Remote Gallery")) {
		t.Fatalf("remote movies response = %d %q, want remote movie", rec.Code, rec.Body.String())
	}

	unauthorizedRemote := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	t.Cleanup(unauthorizedRemote.Close)
	if err := dao.AddServer(structs.Server{Name: "Unauthorized", Address: unauthorizedRemote.URL, Token: "token"}); err != nil {
		t.Fatalf("AddServer(unauthorized) error = %v", err)
	}
	rec = serveServerHandler(http.MethodGet, "/api/servers/:id/movies", "/api/servers/2/movies", getServerMoviesHandler(dao), nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("unauthorized remote status = %d, want 401", rec.Code)
	}
}

func TestRemoteThumbHandler(t *testing.T) {
	dao := newTestLocalDAO(t)
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	workDir := t.TempDir()
	if err := os.Chdir(workDir); err != nil {
		t.Fatalf("Chdir(workDir) error = %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(originalDir); err != nil {
			t.Errorf("Chdir(originalDir) error = %v", err)
		}
	})

	rec := serveHandlerWithParams(remoteThumbHandler(dao, ""), gin.Params{{Key: "id", Value: "1"}, {Key: "hash", Value: ""}})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("empty hash status = %d, want 400", rec.Code)
	}

	rec = serveServerHandler(http.MethodGet, "/remote-thumb/:id/:hash", "/remote-thumb/1/abcdef", remoteThumbHandler(dao, ""), nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("missing server status = %d, want 404", rec.Code)
	}

	if err := os.MkdirAll("thumb_cache", 0755); err != nil {
		t.Fatalf("MkdirAll(thumb_cache) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join("thumb_cache", "cached.jpg"), []byte("cached"), 0644); err != nil {
		t.Fatalf("WriteFile(cache) error = %v", err)
	}
	rec = serveServerHandler(http.MethodGet, "/remote-thumb/:id/:hash", "/remote-thumb/1/cached", remoteThumbHandler(dao, ""), nil)
	if rec.Code != http.StatusOK || rec.Body.String() != "cached" {
		t.Fatalf("cached response = %d %q, want cached", rec.Code, rec.Body.String())
	}

	plexDataPath := t.TempDir()
	localPosterDir := plexMoviePosterDir(plexDataPath, "localhash")
	if err := os.MkdirAll(localPosterDir, 0755); err != nil {
		t.Fatalf("MkdirAll(localPosterDir) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(localPosterDir, "poster.jpg"), []byte("local-poster"), 0644); err != nil {
		t.Fatalf("WriteFile(local poster) error = %v", err)
	}
	rec = serveServerHandler(http.MethodGet, "/remote-thumb/:id/:hash", "/remote-thumb/1/localhash", remoteThumbHandler(dao, plexDataPath), nil)
	if rec.Code != http.StatusOK || rec.Body.String() != "local-poster" {
		t.Fatalf("local poster response = %d %q, want local poster", rec.Code, rec.Body.String())
	}

	remote := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("X-Server-Token"); got != "token" {
			t.Fatalf("X-Server-Token = %q, want token", got)
		}
		w.Header().Set("Content-Type", "image/jpeg")
		_, _ = w.Write([]byte("remote-thumb"))
	}))
	t.Cleanup(remote.Close)
	if err := dao.AddServer(structs.Server{Name: "Remote", Address: remote.URL, Token: "token"}); err != nil {
		t.Fatalf("AddServer() error = %v", err)
	}

	rec = serveServerHandler(http.MethodGet, "/remote-thumb/:id/:hash", "/remote-thumb/1/remotehash", remoteThumbHandler(dao, ""), nil)
	if rec.Code != http.StatusOK || rec.Body.String() != "remote-thumb" {
		t.Fatalf("remote response = %d %q, want remote thumb", rec.Code, rec.Body.String())
	}
	if data, err := os.ReadFile(filepath.Join("thumb_cache", "remotehash.jpg")); err != nil || string(data) != "remote-thumb" {
		t.Fatalf("cached remote thumb = %q, err = %v; want remote-thumb", string(data), err)
	}

	failingRemote := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	t.Cleanup(failingRemote.Close)
	if err := dao.AddServer(structs.Server{Name: "Failing", Address: failingRemote.URL, Token: "token"}); err != nil {
		t.Fatalf("AddServer(failing) error = %v", err)
	}
	rec = serveServerHandler(http.MethodGet, "/remote-thumb/:id/:hash", "/remote-thumb/2/failinghash", remoteThumbHandler(dao, ""), nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("remote failure status = %d, want 401", rec.Code)
	}

	if err := dao.AddServer(structs.Server{Name: "Bad URL", Address: "http://[::1", Token: "token"}); err != nil {
		t.Fatalf("AddServer(bad url) error = %v", err)
	}
	rec = serveServerHandler(http.MethodGet, "/remote-thumb/:id/:hash", "/remote-thumb/3/badurlhash", remoteThumbHandler(dao, ""), nil)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("bad url status = %d, want 500", rec.Code)
	}

	if err := dao.AddServer(structs.Server{Name: "Unreachable", Address: "http://127.0.0.1:1", Token: "token"}); err != nil {
		t.Fatalf("AddServer(unreachable) error = %v", err)
	}
	rec = serveServerHandler(http.MethodGet, "/remote-thumb/:id/:hash", "/remote-thumb/4/unreachablehash", remoteThumbHandler(dao, ""), nil)
	if rec.Code != http.StatusBadGateway {
		t.Fatalf("unreachable status = %d, want 502", rec.Code)
	}
}

func TestRunServerReturnsEarlyConfigurationErrors(t *testing.T) {
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	defer func() {
		if err := os.Chdir(originalDir); err != nil {
			t.Errorf("Chdir(originalDir) error = %v", err)
		}
	}()

	noEnvDir := t.TempDir()
	if err := os.Chdir(noEnvDir); err != nil {
		t.Fatalf("Chdir(noEnvDir) error = %v", err)
	}
	if err := RunServer(); err == nil {
		t.Fatal("RunServer() error = nil without .env, want error")
	}

	badLocalDir := t.TempDir()
	if err := os.Chdir(badLocalDir); err != nil {
		t.Fatalf("Chdir(badLocalDir) error = %v", err)
	}
	if err := os.WriteFile(".env", []byte("PORT=0\nPROTOCOL=http\nLOCAL_DB_PATH=.\nPLEX_DB_PATH=plex.db\n"), 0644); err != nil {
		t.Fatalf("WriteFile(.env) error = %v", err)
	}
	if err := RunServer(); err == nil {
		t.Fatal("RunServer() error = nil with directory local DB path, want error")
	}

	badPlexDir := t.TempDir()
	if err := os.Chdir(badPlexDir); err != nil {
		t.Fatalf("Chdir(badPlexDir) error = %v", err)
	}
	env := "PORT=0\nPROTOCOL=http\nLOCAL_DB_PATH=local_state.db\nPLEX_DB_PATH=missing/plex.db\n"
	if err := os.WriteFile(".env", []byte(env), 0644); err != nil {
		t.Fatalf("WriteFile(.env) error = %v", err)
	}
	if err := RunServer(); err == nil {
		t.Fatal("RunServer() error = nil with inaccessible Plex DB path, want error")
	}
}

func serveServerHandler(method string, route string, target string, handler gin.HandlerFunc, body []byte) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Handle(method, route, handler)

	req := httptest.NewRequest(method, target, bytes.NewReader(body))
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func serveServerHandlerWithCookies(method string, route string, target string, handler gin.HandlerFunc, body []byte, cookies []*http.Cookie) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Handle(method, route, handler)

	req := httptest.NewRequest(method, target, bytes.NewReader(body))
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func hasCookie(cookies []*http.Cookie, name string, value string) bool {
	for _, cookie := range cookies {
		if cookie.Name == name && cookie.Value == value {
			return true
		}
	}
	return false
}

func cookiesByName(cookies []*http.Cookie) map[string]*http.Cookie {
	byName := make(map[string]*http.Cookie, len(cookies))
	for _, cookie := range cookies {
		byName[cookie.Name] = cookie
	}
	return byName
}

func hasCookieWithValue(cookies map[string]*http.Cookie, name string) bool {
	cookie, ok := cookies[name]
	return ok && cookie.Value != ""
}

func serveHandlerWithParams(handler gin.HandlerFunc, params gin.Params) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	c.Params = params
	handler(c)
	return rec
}

func newServerFixtureDB(t *testing.T) *sql.DB {
	t.Helper()

	path := filepath.Join(t.TempDir(), "plex.db")
	createDumpFixtureDB(t, path)
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("Close() error = %v", err)
		}
	})
	return db
}

func newSingleMovieDB(t *testing.T, filePath string, hash string) *sql.DB {
	t.Helper()

	db := newServerFixtureDB(t)
	if _, err := db.Exec("UPDATE media_parts SET file = ?, hash = ? WHERE id = 10", filePath, hash); err != nil {
		t.Fatalf("update movie fixture error = %v", err)
	}
	return db
}

func newDeleteFileDB(t *testing.T, filePath string) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("Close() error = %v", err)
		}
	})

	if _, err := db.Exec("CREATE TABLE media_parts (file TEXT, hash TEXT);"); err != nil {
		t.Fatalf("create media_parts error = %v", err)
	}
	if filePath != "" {
		if _, err := db.Exec("INSERT INTO media_parts (file, hash) VALUES (?, 'file-hash')", filePath); err != nil {
			t.Fatalf("insert media_part error = %v", err)
		}
	}

	return db
}
