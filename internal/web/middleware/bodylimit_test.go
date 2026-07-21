package middleware

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestMaxBodyBytes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	const limit = 16

	r := gin.New()
	r.Use(MaxBodyBytes(limit))
	r.POST("/x", func(c *gin.Context) {
		if _, err := io.ReadAll(c.Request.Body); err != nil {
			c.String(http.StatusRequestEntityTooLarge, "too big")
			return
		}
		c.String(http.StatusOK, "ok")
	})
	r.GET("/x", func(c *gin.Context) { c.String(http.StatusOK, "ok") })

	// Body within the limit is read normally.
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/x", strings.NewReader("0123456789")))
	if w.Code != http.StatusOK {
		t.Errorf("under-limit POST: got %d, want 200", w.Code)
	}

	// Body over the limit makes the handler's read fail (no unbounded buffer).
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/x", bytes.NewReader(make([]byte, limit*4))))
	if w.Code == http.StatusOK {
		t.Errorf("over-limit POST should not succeed, got 200")
	}

	// Bodyless methods pass through untouched.
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/x", nil))
	if w.Code != http.StatusOK {
		t.Errorf("GET should pass through, got %d", w.Code)
	}
}

func TestMaxBodyBytesSkipSuffix(t *testing.T) {
	gin.SetMode(gin.TestMode)
	const limit = 10 << 20

	r := gin.New()
	r.Use(MaxBodyBytes(limit, "/server/importDB"))
	read := func(c *gin.Context) {
		if _, err := io.ReadAll(c.Request.Body); err != nil {
			c.String(http.StatusRequestEntityTooLarge, "too big")
			return
		}
		c.String(http.StatusOK, "ok")
	}
	r.POST("/prefix/panel/api/server/importDB", read)
	r.POST("/prefix/panel/api/server/importDB/other", read)
	r.POST("/x", read)

	large := bytes.Repeat([]byte("x"), limit+1)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/prefix/panel/api/server/importDB", bytes.NewReader(large)))
	if w.Code != http.StatusOK {
		t.Fatalf("restore route should accept an over-limit body, got %d", w.Code)
	}

	for _, path := range []string{"/x", "/prefix/panel/api/server/importDB/other"} {
		w = httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, path, bytes.NewReader(large)))
		if w.Code == http.StatusOK {
			t.Fatalf("non-exempt path %q accepted an over-limit body", path)
		}
	}
}

func TestMaxBodyBytesBySuffixWildcardOverride(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(MaxBodyBytesBySuffix(10, map[string]int64{
		"/panel/api/commercial/applications/*/package": 20,
	}))
	read := func(c *gin.Context) {
		if _, err := io.ReadAll(c.Request.Body); err != nil {
			c.Status(http.StatusRequestEntityTooLarge)
			return
		}
		c.Status(http.StatusOK)
	}
	paths := []string{
		"/hidden/panel/api/commercial/applications/app-1/package",
		"/hidden/panel/api/commercial/applications/app-2/package",
		"/hidden/panel/api/commercial/applications/app-1/extra/package",
		"/hidden/panel/api/commercial/other/app-1/package",
	}
	for _, path := range paths {
		r.POST(path, read)
	}
	for index, path := range paths {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, path, bytes.NewReader(make([]byte, 15))))
		want := http.StatusRequestEntityTooLarge
		if index < 2 {
			want = http.StatusOK
		}
		if w.Code != want {
			t.Fatalf("path %q returned %d, want %d", path, w.Code, want)
		}
	}
}

func TestCommercialPackageRouteBodyLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	const ordinaryLimit = 10 << 20
	const packageLimit = (1 << 30) + (1 << 20)
	r := gin.New()
	r.Use(MaxBodyBytesBySuffix(ordinaryLimit, map[string]int64{
		"/panel/api/commercial/applications/*/package": packageLimit,
	}))
	read := func(c *gin.Context) {
		if _, err := io.Copy(io.Discard, c.Request.Body); err != nil {
			c.Status(http.StatusRequestEntityTooLarge)
			return
		}
		c.Status(http.StatusOK)
	}
	packagePath := "/123456789012345678/panel/api/commercial/applications/client/package"
	r.POST(packagePath, read)
	r.POST("/ordinary", read)
	elevenMiB := bytes.Repeat([]byte("x"), 11<<20)
	response := httptest.NewRecorder()
	r.ServeHTTP(response, httptest.NewRequest(http.MethodPost, packagePath, bytes.NewReader(elevenMiB)))
	if response.Code != http.StatusOK {
		t.Fatalf("11 MiB package upload returned %d", response.Code)
	}
	response = httptest.NewRecorder()
	r.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/ordinary", bytes.NewReader(elevenMiB)))
	if response.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("11 MiB ordinary request returned %d", response.Code)
	}
	request := httptest.NewRequest(http.MethodPost, packagePath, http.NoBody)
	request.ContentLength = packageLimit + 1
	response = httptest.NewRecorder()
	r.ServeHTTP(response, request)
	if response.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("oversized package request returned %d", response.Code)
	}
}
