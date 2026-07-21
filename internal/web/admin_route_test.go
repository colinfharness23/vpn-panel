package web

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/robfig/cron/v3"

	"github.com/mhsanaei/3x-ui/v3/internal/database"
	"github.com/mhsanaei/3x-ui/v3/internal/web/global"
)

func TestPrivateAdminRouteIsIndependentFromPortal(t *testing.T) {
	t.Setenv("XUI_COMMERCIAL_ENV", "test")
	t.Setenv("XUI_ADMIN_BASE_PATH", "/583104927618350492/")
	t.Setenv("XUI_DB_TYPE", "sqlite")
	if err := database.InitDB(filepath.Join(t.TempDir(), "admin-route.db")); err != nil {
		t.Fatalf("InitDB: %v", err)
	}
	t.Cleanup(func() { _ = database.CloseDB() })

	server := NewServer()
	server.cron = cron.New()
	global.SetWebServer(server)
	engine, err := server.initRouter()
	if err != nil {
		t.Fatalf("initRouter: %v", err)
	}
	t.Cleanup(func() {
		server.cancel()
		server.cron.Stop()
		if server.wsHub != nil {
			server.wsHub.Stop()
		}
		global.SetWebServer(nil)
	})

	cases := []struct {
		path       string
		wantStatus int
		wantBody   string
	}{
		{path: "/", wantStatus: http.StatusOK, wantBody: `<div id="portal"></div>`},
		{path: "/portal/", wantStatus: http.StatusPermanentRedirect},
		{path: "/583104927618350492/", wantStatus: http.StatusOK, wantBody: "X_UI_BASE_PATH=\"/583104927618350492/\""},
		{path: "/panel/", wantStatus: http.StatusNotFound},
		{path: "/admin/", wantStatus: http.StatusNotFound},
		{path: "/123456789012345678/", wantStatus: http.StatusNotFound},
	}

	for _, tc := range cases {
		t.Run(tc.path, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, tc.path, nil)
			request.Header.Set("Accept", "text/html")
			engine.ServeHTTP(recorder, request)
			if recorder.Code != tc.wantStatus {
				t.Fatalf("GET %s status = %d, want %d; body=%s", tc.path, recorder.Code, tc.wantStatus, recorder.Body.String())
			}
			if tc.wantBody != "" && !strings.Contains(recorder.Body.String(), tc.wantBody) {
				t.Fatalf("GET %s did not inject private base path", tc.path)
			}
		})
	}
}
