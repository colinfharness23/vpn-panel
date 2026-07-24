package controller

import (
	"bytes"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"

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

func TestManagedLineIngressOnlyProxiesTheRegisteredPortAndToken(t *testing.T) {
	t.Setenv("XUI_COMMERCIAL_ENV", "test")
	if err := database.InitDB(filepath.Join(t.TempDir(), "commercial-line-ingress.db")); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if sqlDB, err := database.GetDB().DB(); err == nil {
			_ = sqlDB.Close()
		}
	})
	db := database.GetDB()
	for key, value := range map[string]string{
		"site.url":           "https://vpn.pheero.com",
		"security.safe_mode": "true",
	} {
		if err := db.Create(&model.CommercialSetting{Key: key, Value: value}).Error; err != nil {
			t.Fatal(err)
		}
	}

	listener := reserveManagedLineIngressPort(t)
	port := listener.Addr().(*net.TCPAddr).Port
	upgrader := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	backend := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		if strings.EqualFold(request.Header.Get("Upgrade"), "websocket") {
			connection, err := upgrader.Upgrade(w, request, nil)
			if err != nil {
				return
			}
			defer connection.Close()
			messageType, message, err := connection.ReadMessage()
			if err == nil {
				_ = connection.WriteMessage(messageType, append([]byte("echo:"), message...))
			}
			return
		}
		_, _ = io.WriteString(w, "managed-line-proxy-ok")
	})}
	go func() { _ = backend.Serve(listener) }()
	t.Cleanup(func() { _ = backend.Close() })

	token := strings.Repeat("a", 16)
	path := "/nova-line/" + strconv.Itoa(port) + "/" + token
	inbound := model.Inbound{
		UserId: 1, Remark: "IPLC", Enable: true, Listen: "127.0.0.1", Port: port,
		Protocol: model.VLESS, Settings: `{"clients":[],"decryption":"none"}`,
		StreamSettings: `{"network":"ws","security":"none","wsSettings":{"path":"` + path + `","host":"vpn.pheero.com"}}`,
		Tag:            "commercial-in-public-ingress-test",
	}
	if err := db.Create(&inbound).Error; err != nil {
		t.Fatal(err)
	}
	node := model.LineNode{
		ID: uuid.NewString(), Fingerprint: strings.Repeat("a", 64), Remark: "provider",
		PublicName: "IPLC", Protocol: "vless", OutboundTag: "commercial-line-public-ingress-test",
		OutboundCiphertext: "encrypted", PublicPort: &port, InboundID: &inbound.Id,
		// Use a deliberately future-proof version here: the public controller
		// only cares that this fixture represents a fully provisioned line.
		Status: "ready", HealthStatus: "healthy", ProvisionVersion: 1 << 30,
	}
	if err := db.Create(&node).Error; err != nil {
		t.Fatal(err)
	}

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(func(c *gin.Context) {
		c.Set("base_path", "/")
		c.Next()
	})
	NewCommercialPublicController(engine.Group(""), nil)

	publicServer := httptest.NewServer(engine)
	t.Cleanup(publicServer.Close)
	if err := db.Model(&model.CommercialSetting{}).Where("key = ?", "site.url").Update("value", publicServer.URL).Error; err != nil {
		t.Fatal(err)
	}
	request, err := http.NewRequest(http.MethodGet, publicServer.URL+path, nil)
	if err != nil {
		t.Fatal(err)
	}
	response, err := publicServer.Client().Do(request)
	if err != nil {
		t.Fatal(err)
	}
	body, err := io.ReadAll(response.Body)
	response.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusOK || string(body) != "managed-line-proxy-ok" {
		t.Fatalf("valid line ingress = %d %q", response.StatusCode, body)
	}

	webSocketURL := "ws" + strings.TrimPrefix(publicServer.URL, "http") + path
	connection, _, err := websocket.DefaultDialer.Dial(webSocketURL, nil)
	if err != nil {
		t.Fatalf("upgrade managed line WebSocket: %v", err)
	}
	if err := connection.WriteMessage(websocket.TextMessage, []byte("ping")); err != nil {
		connection.Close()
		t.Fatal(err)
	}
	_, message, err := connection.ReadMessage()
	connection.Close()
	if err != nil || string(message) != "echo:ping" {
		t.Fatalf("managed line WebSocket echo = %q, %v", message, err)
	}

	request, err = http.NewRequest(http.MethodGet, publicServer.URL+"/nova-line/"+strconv.Itoa(port)+"/"+strings.Repeat("b", 16), nil)
	if err != nil {
		t.Fatal(err)
	}
	response, err = publicServer.Client().Do(request)
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusNotFound {
		t.Fatalf("invalid line token returned %d", response.StatusCode)
	}
}

func reserveManagedLineIngressPort(t *testing.T) net.Listener {
	t.Helper()
	for port := 20000; port <= 59999; port++ {
		listener, err := net.Listen("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(port)))
		if err == nil {
			return listener
		}
	}
	t.Fatal("no managed line test port available")
	return nil
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
	if _, err := commercial.NewAdminService().SaveApplicationPackage(application.ID, file, header.Filename, header.Header.Get("Content-Type")); err != nil {
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

func TestAuthenticatedApplicationDownloadUsesInternalRedirectInProduction(t *testing.T) {
	t.Setenv("XUI_COMMERCIAL_ENV", "test")
	t.Setenv("XUI_DB_FOLDER", t.TempDir())
	t.Setenv("XUI_APPLICATION_ACCEL_PREFIX", "/_nova_internal/client-applications/")
	if err := database.InitDB(filepath.Join(t.TempDir(), "commercial-accelerated-download.db")); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if sqlDB, err := database.GetDB().DB(); err == nil {
			_ = sqlDB.Close()
		}
	})

	application := model.ClientApplication{ID: uuid.NewString(), Slug: "accelerated-app", Name: "Accelerated App", Platform: "test", Active: true}
	if err := database.GetDB().Create(&application).Error; err != nil {
		t.Fatal(err)
	}
	var upload bytes.Buffer
	writer := multipart.NewWriter(&upload)
	part, err := writer.CreateFormFile("package", "client.zip")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.WriteString(part, "accelerated-content"); err != nil {
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
	saved, err := commercial.NewAdminService().SaveApplicationPackage(application.ID, file, header.Filename, header.Header.Get("Content-Type"))
	if err != nil {
		t.Fatal(err)
	}

	customer := model.Customer{ID: uuid.NewString(), Email: "accel@example.com", PasswordHash: "unused", Status: "active", InviteCode: "ACCEL001"}
	if err := database.GetDB().Create(&customer).Error; err != nil {
		t.Fatal(err)
	}
	token, _, err := commercial.NewAuthService(nil).CreateSession(customer.ID, "203.0.113.2", "test")
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
	response := httptest.NewRecorder()
	engine.ServeHTTP(response, request)

	want := "/_nova_internal/client-applications/" + url.PathEscape(saved.PackageStoredName)
	if response.Code != http.StatusOK || response.Header().Get("X-Accel-Redirect") != want {
		t.Fatalf("accelerated response = %d redirect=%q, want %q", response.Code, response.Header().Get("X-Accel-Redirect"), want)
	}
	if response.Body.Len() != 0 {
		t.Fatalf("accelerated response must not stream through Go, got %d bytes", response.Body.Len())
	}
}
