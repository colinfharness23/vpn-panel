package controller

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/google/uuid"

	"github.com/mhsanaei/3x-ui/v3/internal/database"
	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
	"github.com/mhsanaei/3x-ui/v3/internal/web/service/commercial"

	"github.com/gin-gonic/gin"
)

func TestCommercialPublicAuthenticationBoundary(t *testing.T) {
	t.Setenv("XUI_COMMERCIAL_ENV", "test")
	if err := database.InitDB(filepath.Join(t.TempDir(), "commercial-public.db")); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if sqlDB, err := database.GetDB().DB(); err == nil {
			_ = sqlDB.Close()
		}
	})
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(func(c *gin.Context) {
		c.Set("base_path", "/")
		c.Next()
	})
	NewCommercialPublicController(engine.Group(""), nil)

	for _, test := range []struct {
		path string
		want int
	}{
		{"/api/v1/guest/auth-config", http.StatusOK},
		{"/api/v1/user/bootstrap", http.StatusUnauthorized},
		{"/api/v1/user/applications/not-known/download", http.StatusUnauthorized},
	} {
		request := httptest.NewRequest(http.MethodGet, test.path, nil)
		response := httptest.NewRecorder()
		engine.ServeHTTP(response, request)
		if response.Code != test.want {
			t.Fatalf("GET %s returned %d, want %d: %s", test.path, response.Code, test.want, response.Body.String())
		}
	}
}

func TestAuthenticatedApplicationDownloadSupportsRange(t *testing.T) {
	t.Setenv("XUI_COMMERCIAL_ENV", "test")
	t.Setenv("XUI_DB_FOLDER", t.TempDir())
	if err := database.InitDB(filepath.Join(t.TempDir(), "commercial-download.db")); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if sqlDB, err := database.GetDB().DB(); err == nil {
			_ = sqlDB.Close()
		}
	})
	application := model.ClientApplication{ID: uuid.NewString(), Slug: "range-app", Name: "Range App", Platform: "test", Active: true}
	if err := database.GetDB().Create(&application).Error; err != nil {
		t.Fatal(err)
	}
	var upload bytes.Buffer
	writer := multipart.NewWriter(&upload)
	part, err := writer.CreateFormFile("package", "range.zip")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.WriteString(part, "abcdefghij"); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	uploadRequest := httptest.NewRequest(http.MethodPost, "/upload", &upload)
	uploadRequest.Header.Set("Content-Type", writer.FormDataContentType())
	if err := uploadRequest.ParseMultipartForm(1 << 20); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if uploadRequest.MultipartForm != nil {
			_ = uploadRequest.MultipartForm.RemoveAll()
		}
	})
	file, header, err := uploadRequest.FormFile("package")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if _, err := commercial.NewAdminService().SaveApplicationPackage(application.ID, file, header); err != nil {
		t.Fatal(err)
	}
	customer := model.Customer{ID: uuid.NewString(), Email: "range@example.com", PasswordHash: "unused", Status: "active", InviteCode: "RANGE001"}
	if err := database.GetDB().Create(&customer).Error; err != nil {
		t.Fatal(err)
	}
	token, _, err := commercial.NewAuthService(nil).CreateSession(customer.ID, "203.0.113.1", "test")
	if err != nil {
		t.Fatal(err)
	}
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(func(c *gin.Context) {
		c.Set("base_path", "/")
		c.Next()
	})
	NewCommercialPublicController(engine.Group(""), nil)
	request := httptest.NewRequest(http.MethodGet, "/api/v1/user/applications/"+application.ID+"/download", nil)
	request.AddCookie(&http.Cookie{Name: customerSessionCookie, Value: token})
	request.Header.Set("Range", "bytes=2-4")
	response := httptest.NewRecorder()
	engine.ServeHTTP(response, request)
	if response.Code != http.StatusPartialContent || response.Body.String() != "cde" {
		t.Fatalf("range response = %d %q", response.Code, response.Body.String())
	}
}
