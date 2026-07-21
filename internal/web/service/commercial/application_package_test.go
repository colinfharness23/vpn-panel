package commercial

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"github.com/mhsanaei/3x-ui/v3/internal/database"
	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
)

func TestApplicationPackageUploadAndDownload(t *testing.T) {
	t.Setenv("XUI_DB_FOLDER", t.TempDir())
	initCommercialTestDB(t)

	row := model.ClientApplication{
		ID: uuid.NewString(), Slug: "v2rayng-test", Name: "v2rayNG", Platform: "Android",
		Description: "Android client", Active: true,
	}
	if err := database.GetDB().Create(&row).Error; err != nil {
		t.Fatal(err)
	}

	content := []byte("fake-apk-content")
	file, header := newMultipartTestFile(t, "v2rayNG.apk", content)
	saved, err := NewAdminService().SaveApplicationPackage(row.ID, file, header)
	if err != nil {
		t.Fatalf("SaveApplicationPackage: %v", err)
	}
	if saved.PackageFileName != "v2rayNG.apk" || saved.PackageSize != int64(len(content)) {
		t.Fatalf("unexpected package metadata: %+v", saved)
	}
	encoded, err := json.Marshal(saved)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(encoded, []byte(saved.PackageStoredName)) {
		t.Fatal("stored package name must not be exposed in API responses")
	}
	if saved.DownloadURL != "/api/v1/user/applications/"+row.ID+"/download" {
		t.Fatalf("downloadUrl=%q", saved.DownloadURL)
	}

	openedRow, opened, info, err := NewPortalService().OpenApplicationPackage(row.ID)
	if err != nil {
		t.Fatalf("OpenApplicationPackage: %v", err)
	}
	defer opened.Close()
	got, err := io.ReadAll(opened)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, content) || info.Size() != int64(len(content)) || openedRow.PackageFileName != "v2rayNG.apk" {
		t.Fatalf("downloaded package mismatch: name=%q size=%d content=%q", openedRow.PackageFileName, info.Size(), got)
	}

	if err := database.GetDB().Model(&row).Update("active", false).Error; err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := NewPortalService().OpenApplicationPackage(row.ID); err == nil {
		t.Fatal("disabled application package must not be downloadable")
	}
}

func TestApplicationPackageValidation(t *testing.T) {
	for _, name := range []string{"client.txt", "client.apk.exe.txt", ".."} {
		if _, _, err := normalizedClientPackageName(name); err == nil {
			t.Fatalf("expected %q to be rejected", name)
		}
	}
	name, suffix, err := normalizedClientPackageName(`..\folder\v2rayNG.APK`)
	if err != nil || name != "v2rayNG.APK" || suffix != ".apk" {
		t.Fatalf("normalized name=%q suffix=%q err=%v", name, suffix, err)
	}
	if _, err := storedClientPackagePath("../escape.apk"); err == nil {
		t.Fatal("stored path traversal must be rejected")
	}
}

func newMultipartTestFile(t *testing.T, name string, content []byte) (multipart.File, *multipart.FileHeader) {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("package", name)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write(content); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/upload", bytes.NewReader(body.Bytes()))
	req.Header.Set("Content-Type", writer.FormDataContentType())
	if err := req.ParseMultipartForm(int64(body.Len()) + 1024); err != nil {
		t.Fatal(err)
	}
	file, header, err := req.FormFile("package")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = file.Close()
		if req.MultipartForm != nil {
			_ = req.MultipartForm.RemoveAll()
		}
	})
	return file, header
}
