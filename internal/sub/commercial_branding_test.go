package sub

import (
	"encoding/base64"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/mhsanaei/3x-ui/v3/internal/database"
	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
)

func TestManagedLineSubscriptionHidesImportedProviderRemark(t *testing.T) {
	initSubDB(t)
	db := database.GetDB()
	if err := db.Create(&model.CommercialSetting{Key: "site.name", Value: "Pheero VPN"}).Error; err != nil {
		t.Fatalf("seed site name: %v", err)
	}
	inbound := &model.Inbound{
		UserId: 1, Remark: "A机场 香港 IPLC", Enable: true, Port: 24443,
		Protocol: model.VLESS, Settings: `{"clients":[]}`, StreamSettings: `{"network":"tcp","security":"none"}`,
	}
	if err := db.Create(inbound).Error; err != nil {
		t.Fatalf("seed inbound: %v", err)
	}
	client := &model.ClientRecord{Email: "customer@example.com", SubID: "brand-sub", UUID: "11111111-2222-4333-8444-555555555555", Enable: true}
	if err := db.Create(client).Error; err != nil {
		t.Fatalf("seed client: %v", err)
	}
	if err := db.Create(&model.ClientInbound{ClientId: client.Id, InboundId: inbound.Id}).Error; err != nil {
		t.Fatalf("seed client inbound: %v", err)
	}
	lineID := "abcd1234-1111-4222-8333-abcdefabcdef"
	if err := db.Create(&model.LineNode{
		ID: lineID, Fingerprint: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		Remark: "A机场 香港 IPLC", Protocol: "vless", OutboundTag: "commercial-line-test",
		OutboundCiphertext: "encrypted", InboundID: &inbound.Id, Status: "healthy", HealthStatus: "healthy",
	}).Error; err != nil {
		t.Fatalf("seed line: %v", err)
	}

	rows, err := (&SubService{}).getInboundsBySubId("brand-sub")
	if err != nil {
		t.Fatalf("get inbounds: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("got %d inbounds, want 1", len(rows))
	}
	if got, want := rows[0].Remark, "Pheero VPN 线路 ABCD12"; got != want {
		t.Fatalf("managed line remark = %q, want %q", got, want)
	}

	links, _, _, _, err := NewSubService("").GetSubs("brand-sub", "vpn.pheero.com")
	if err != nil || len(links) != 1 {
		t.Fatalf("render managed subscription: links=%v err=%v", links, err)
	}
	if strings.Contains(links[0], "A机场") || strings.Contains(links[0], "IPLC") {
		t.Fatalf("rendered subscription leaked upstream branding: %q", links[0])
	}
	parsed, err := url.Parse(links[0])
	if err != nil {
		t.Fatalf("parse managed subscription link: %v", err)
	}
	if got, wantPrefix := parsed.Fragment, "Pheero VPN 线路 ABCD12"; !strings.HasPrefix(got, wantPrefix) {
		t.Fatalf("rendered node name = %q, want prefix %q", got, wantPrefix)
	}
}

func TestCommercialSubscriptionHeadersUseSiteBrandOnly(t *testing.T) {
	initSubDB(t)
	t.Setenv("XUI_COMMERCIAL_ENV", "production")
	db := database.GetDB()
	for key, value := range map[string]string{
		"site.name":        "Pheero VPN",
		"site.url":         "https://vpn.pheero.com",
		"site.support_url": "https://vpn.pheero.com/tickets",
	} {
		if err := db.Create(&model.CommercialSetting{Key: key, Value: value}).Error; err != nil {
			t.Fatalf("seed %s: %v", key, err)
		}
	}

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	controller := &SUBController{subTitle: "A机场", subSupportUrl: "https://a.example/support", subProfileUrl: "https://a.example"}
	controller.ApplyCommonHeaders(context, "", "30", "A机场", "https://a.example/support", "https://a.example", "", false, "", false)

	wantTitle := "base64:" + base64.StdEncoding.EncodeToString([]byte("Pheero VPN"))
	if got := recorder.Header().Get("Profile-Title"); got != wantTitle {
		t.Fatalf("Profile-Title = %q, want %q", got, wantTitle)
	}
	if got := recorder.Header().Get("Profile-Web-Page-Url"); got != "https://vpn.pheero.com" {
		t.Fatalf("Profile-Web-Page-Url = %q", got)
	}
	if got := recorder.Header().Get("Support-Url"); got != "https://vpn.pheero.com/tickets" {
		t.Fatalf("Support-Url = %q", got)
	}
}
