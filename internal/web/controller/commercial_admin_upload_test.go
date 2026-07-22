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
	webmiddleware "github.com/mhsanaei/3x-ui/v3/internal/web/middleware"
	"github.com/mhsanaei/3x-ui/v3/internal/web/service/commercial"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
)

func TestCommercialApplicationPackageRouteStreamsElevenMiB(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("XUI_COMMERCIAL_ENV", "test")
	t.Setenv("XUI_DB_FOLDER", dataDir)
	if err := database.InitDB(filepath.Join(dataDir, "commercial-upload.db")); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if sqlDB, err := database.GetDB().DB(); err == nil {
			_ = sqlDB.Close()
		}
	})
	application := model.ClientApplication{
		ID: uuid.NewString(), Slug: "stream-test", Name: "Stream test", Platform: "Windows", Active: true,
	}
	if err := database.GetDB().Create(&application).Error; err != nil {
		t.Fatal(err)
	}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("package", "client.zip")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.Copy(part, io.LimitReader(zeroReader{}, 11<<20)); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(webmiddleware.MaxBodyBytesBySuffix(10<<20, map[string]int64{
		"/applications/*/package": commercial.MaxClientPackageSize + (1 << 20),
	}))
	engine.Use(sessions.Sessions("3x-ui", cookie.NewStore([]byte("commercial-upload-test-session-key"))))
	controller := &CommercialAdminController{service: commercial.NewAdminService()}
	engine.POST("/applications/:id/package", controller.uploadApplicationPackage)
	request := httptest.NewRequest(http.MethodPost, "/applications/"+application.ID+"/package", bytes.NewReader(body.Bytes()))
	request.Header.Set("Content-Type", writer.FormDataContentType())
	response := httptest.NewRecorder()
	engine.ServeHTTP(response, request)
	if response.Code != http.StatusOK || !bytes.Contains(response.Body.Bytes(), []byte(`"success":true`)) {
		t.Fatalf("upload returned %d: %s", response.Code, response.Body.String())
	}
	var saved model.ClientApplication
	if err := database.GetDB().First(&saved, "id = ?", application.ID).Error; err != nil {
		t.Fatal(err)
	}
	if saved.PackageSize != 11<<20 || saved.PackageFileName != "client.zip" {
		t.Fatalf("saved package metadata = %+v", saved)
	}
}

type zeroReader struct{}

func (zeroReader) Read(p []byte) (int, error) {
	for index := range p {
		p[index] = 0
	}
	return len(p), nil
}
