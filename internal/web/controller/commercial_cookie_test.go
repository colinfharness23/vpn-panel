package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestCustomerSessionCookieIsAlwaysSecure(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "http://example.test/", nil)
	context.Set("public_base_path", "/")

	controller := &CommercialPublicController{}
	controller.setCustomerCookie(context, "session-token")

	cookies := recorder.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("got %d cookies, want 1", len(cookies))
	}
	if !cookies[0].Secure || !cookies[0].HttpOnly || cookies[0].SameSite != http.SameSiteLaxMode {
		t.Fatalf("customer cookie missing security attributes: %+v", cookies[0])
	}
}
