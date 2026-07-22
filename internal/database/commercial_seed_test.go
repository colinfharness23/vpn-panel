package database

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
)

func TestSyncFirstAdminCustomerHonorsDeletionMarker(t *testing.T) {
	initMigrateDB(t)

	db := GetDB()
	admin := model.User{Username: "marker-admin", Password: "password-hash"}
	if err := db.Create(&admin).Error; err != nil {
		t.Fatalf("create admin user: %v", err)
	}
	marker := model.CommercialSetting{
		Key:   AdminCustomerDeletionMarkerKey(admin.Id),
		Value: "true",
	}
	if err := db.Create(&marker).Error; err != nil {
		t.Fatalf("create deletion marker: %v", err)
	}

	if err := syncFirstAdminCustomer(db, &admin); err != nil {
		t.Fatalf("sync first admin customer: %v", err)
	}

	var count int64
	if err := db.Model(&model.Customer{}).Where("admin_user_id = ?", admin.Id).Count(&count).Error; err != nil {
		t.Fatalf("count linked customers: %v", err)
	}
	if count != 0 {
		t.Fatalf("linked customer count = %d, want 0 after explicit deletion", count)
	}
}

func TestRepairWelcomeNoticeEnglishAfterLazyTabDataLoss(t *testing.T) {
	initMigrateDB(t)
	db := GetDB()
	if err := db.Save(&model.CommercialSetting{Key: "site.name", Value: "PHEERO"}).Error; err != nil {
		t.Fatal(err)
	}
	zh := "欢迎使用 PHEERO。选购前可先查看套餐说明与使用文档，开通后重要信息会集中显示在账户中。"
	if err := db.Model(&model.Notice{}).Where("slug = ?", "welcome").Updates(map[string]any{
		"title_i18n":   `{"zh-CN":"服务公告"}`,
		"content_i18n": `{"zh-CN":"` + zh + `"}`,
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := repairWelcomeNoticeEnglish(db); err != nil {
		t.Fatal(err)
	}
	var row model.Notice
	if err := db.First(&row, "slug = ?", "welcome").Error; err != nil {
		t.Fatal(err)
	}
	var titles, contents map[string]string
	if json.Unmarshal([]byte(row.TitleI18n), &titles) != nil || json.Unmarshal([]byte(row.ContentI18n), &contents) != nil {
		t.Fatal("repaired notice contains invalid localized JSON")
	}
	if titles["en-US"] != "Service notice" || !strings.Contains(contents["en-US"], "PHEERO") {
		t.Fatalf("English repair missing: titles=%v contents=%v", titles, contents)
	}
	if contents["zh-CN"] != zh {
		t.Fatalf("Chinese content was changed: %q", contents["zh-CN"])
	}
}

func TestRepairWelcomeNoticeDoesNotTranslateCustomContent(t *testing.T) {
	initMigrateDB(t)
	db := GetDB()
	if err := db.Model(&model.Notice{}).Where("slug = ?", "welcome").Updates(map[string]any{
		"title_i18n":   `{"zh-CN":"自定义"}`,
		"content_i18n": `{"zh-CN":"完全自定义的运营通知"}`,
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := repairWelcomeNoticeEnglish(db); err != nil {
		t.Fatal(err)
	}
	var row model.Notice
	if err := db.First(&row, "slug = ?", "welcome").Error; err != nil {
		t.Fatal(err)
	}
	if strings.Contains(row.TitleI18n, "en-US") || strings.Contains(row.ContentI18n, "en-US") {
		t.Fatalf("custom content must not be rewritten: %+v", row)
	}
}
