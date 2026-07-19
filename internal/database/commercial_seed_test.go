package database

import (
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
