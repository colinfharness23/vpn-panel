package database

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/mhsanaei/3x-ui/v3/internal/database/model"

	"gorm.io/gorm"
)

func seedCommercialDefaults() error {
	return db.Transaction(func(tx *gorm.DB) error {
		var planCount int64
		if err := tx.Model(&model.Plan{}).Count(&planCount).Error; err != nil {
			return err
		}
		if planCount == 0 {
			if err := seedCommercialPlans(tx); err != nil {
				return err
			}
		}
		const renewalUpgradeSeeder = "CommercialRenewalUpgradePolicies"
		var renewalUpgradeSeeded int64
		if err := tx.Model(&model.HistoryOfSeeders{}).Where("seeder_name = ?", renewalUpgradeSeeder).Count(&renewalUpgradeSeeded).Error; err != nil {
			return err
		}
		if renewalUpgradeSeeded == 0 {
			if err := tx.Model(&model.Plan{}).
				Where("slug IN ?", []string{"starter", "pro", "ultimate"}).
				Updates(map[string]any{"renewable": true, "upgradable": true}).Error; err != nil {
				return err
			}
			if err := tx.Create(&model.HistoryOfSeeders{SeederName: renewalUpgradeSeeder}).Error; err != nil {
				return err
			}
		}

		var appCount int64
		if err := tx.Model(&model.ClientApplication{}).Count(&appCount).Error; err != nil {
			return err
		}
		if appCount == 0 {
			if err := seedCommercialApplications(tx); err != nil {
				return err
			}
		}

		var articleCount int64
		if err := tx.Model(&model.KnowledgeArticle{}).Count(&articleCount).Error; err != nil {
			return err
		}
		if articleCount == 0 {
			if err := seedCommercialArticles(tx); err != nil {
				return err
			}
		}

		var noticeCount int64
		if err := tx.Model(&model.Notice{}).Count(&noticeCount).Error; err != nil {
			return err
		}
		if noticeCount == 0 {
			titles := localizedText("服务公告", "Service notice")
			contents := localizedText("欢迎使用 NOVA。选购前可先查看套餐说明与使用文档，开通后重要信息会集中显示在账户中。", "Welcome to NOVA. Review the plan details and documentation before purchase; important setup information will stay available in your account.")
			now := time.Now().UTC()
			notice := model.Notice{ID: uuid.NewString(), Slug: "welcome", Level: "info", TitleI18n: titles, ContentI18n: contents, Published: true, PublishedAt: &now}
			if err := tx.Create(&notice).Error; err != nil {
				return err
			}
		}

		var user model.User
		if err := tx.Order("id asc").First(&user).Error; err == nil {
			binding := model.AdminRoleBinding{UserID: user.Id, Role: "owner"}
			if err := tx.Where("user_id = ?", user.Id).FirstOrCreate(&binding).Error; err != nil {
				return err
			}
			if err := syncFirstAdminCustomer(tx, &user); err != nil {
				return err
			}
		}
		return nil
	})
}

func syncFirstAdminCustomer(tx *gorm.DB, user *model.User) error {
	var linked model.Customer
	if err := tx.Where("admin_user_id = ?", user.Id).First(&linked).Error; err == nil {
		return nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	var deletionMarker model.CommercialSetting
	if err := tx.Where("key = ?", AdminCustomerDeletionMarkerKey(user.Id)).First(&deletionMarker).Error; err == nil {
		if strings.EqualFold(strings.TrimSpace(deletionMarker.Value), "true") {
			return nil
		}
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	username := strings.TrimSpace(user.Username)
	email := strings.ToLower(username)
	if !strings.HasSuffix(email, "@gmail.com") {
		email = fmt.Sprintf("admin-%d@admin.local", user.Id)
	}
	var existing model.Customer
	if err := tx.Where("email = ?", email).First(&existing).Error; err == nil {
		return tx.Model(&existing).Update("admin_user_id", user.Id).Error
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	now := time.Now().UTC()
	createdAt := now
	var oldest model.Customer
	if err := tx.Order("created_at asc").First(&oldest).Error; err == nil && !oldest.CreatedAt.IsZero() {
		createdAt = oldest.CreatedAt.Add(-time.Second)
	}
	customer := model.Customer{
		ID:              uuid.NewString(),
		AdminUserID:     &user.Id,
		Email:           email,
		PasswordHash:    user.Password,
		DisplayName:     username,
		Locale:          "zh-CN",
		Status:          "active",
		InviteCode:      strings.ToUpper(strings.ReplaceAll(uuid.NewString()[:8], "-", "")),
		EmailVerifiedAt: &now,
		CreatedAt:       createdAt,
	}
	return tx.Create(&customer).Error
}

func AdminCustomerDeletionMarkerKey(userID int) string {
	return fmt.Sprintf("system.admin_customer_deleted.%d", userID)
}

func seedCommercialPlans(tx *gorm.DB) error {
	type planSeed struct {
		slug        string
		name        string
		description string
		traffic     int64
		devices     int
		prices      []struct {
			period string
			months int
			amount int64
		}
	}
	seeds := []planSeed{
		{slug: "starter", name: "轻量套餐", description: "适合轻度浏览与临时使用", traffic: 100 * 1024 * 1024 * 1024, devices: 3, prices: []struct {
			period string
			months int
			amount int64
		}{{"monthly", 1, 1000}, {"quarterly", 3, 2700}, {"yearly", 12, 9600}}},
		{slug: "pro", name: "专业套餐", description: "适合多设备的日常稳定使用", traffic: 300 * 1024 * 1024 * 1024, devices: 5, prices: []struct {
			period string
			months int
			amount int64
		}{{"monthly", 1, 2000}, {"quarterly", 3, 5400}, {"yearly", 12, 19200}}},
		{slug: "ultimate", name: "旗舰套餐", description: "适合多设备与高流量场景", traffic: 1024 * 1024 * 1024 * 1024, devices: 10, prices: []struct {
			period string
			months int
			amount int64
		}{{"monthly", 1, 4500}, {"half_yearly", 6, 22900}, {"yearly", 12, 40500}}},
	}
	for index, seed := range seeds {
		// Seed editable examples, but never sell them before an operator binds
		// real 3X-UI inbound IDs and explicitly publishes the plan.
		plan := model.Plan{ID: uuid.NewString(), Slug: seed.slug, Name: seed.name, Description: seed.description, TrafficBytes: seed.traffic, DeviceLimit: seed.devices, ResetCycle: "monthly", NodeGroup: "default", Visibility: "public", Renewable: true, Upgradable: true, Active: false, SortOrder: index + 1, ProvisionInboundIDs: "[]"}
		if err := tx.Create(&plan).Error; err != nil {
			return err
		}
		for _, price := range seed.prices {
			row := model.PlanPrice{ID: uuid.NewString(), PlanID: plan.ID, BillingPeriod: price.period, Months: price.months, AmountFen: price.amount, Active: true}
			if err := tx.Create(&row).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

func seedCommercialApplications(tx *gorm.DB) error {
	apps := []model.ClientApplication{
		{ID: uuid.NewString(), Slug: "v2rayn", Name: "v2rayN", Platform: "Windows / macOS / Linux", OfficialURL: "https://github.com/2dust/v2rayN/releases", SourceURL: "https://github.com/2dust/v2rayN", Description: "适合桌面设备使用，支持订阅导入与更新", Active: true, SortOrder: 1},
		{ID: uuid.NewString(), Slug: "clash-verge-rev", Name: "Clash Verge Rev", Platform: "Windows / macOS / Linux", OfficialURL: "https://github.com/clash-verge-rev/clash-verge-rev/releases", SourceURL: "https://github.com/clash-verge-rev/clash-verge-rev", Description: "界面清晰的跨平台桌面客户端", Active: true, SortOrder: 2},
		{ID: uuid.NewString(), Slug: "shadowrocket", Name: "Shadowrocket", Platform: "iOS", OfficialURL: "https://apps.apple.com/us/app/shadowrocket/id932747118", Description: "请从 Apple App Store 获取正版客户端", Active: true, SortOrder: 3},
		{ID: uuid.NewString(), Slug: "hiddify", Name: "Hiddify", Platform: "Android / iOS / Desktop", OfficialURL: "https://github.com/hiddify/hiddify-app/releases", SourceURL: "https://github.com/hiddify/hiddify-app", Description: "开源多平台订阅客户端", Active: true, SortOrder: 4},
	}
	return tx.Create(&apps).Error
}

func seedCommercialArticles(tx *gorm.DB) error {
	type articleSeed struct {
		slug     string
		category string
		zh       string
		en       string
		bodyZH   string
		bodyEN   string
	}
	seeds := []articleSeed{
		{slug: "windows-import", category: "Windows", zh: "Windows 导入教程", en: "Windows setup", bodyZH: "安装 v2rayN 或 Clash Verge Rev，复制订阅链接，在客户端中选择从剪贴板导入并更新订阅。", bodyEN: "Install v2rayN or Clash Verge Rev, copy the subscription URL, import it from the clipboard, then refresh the profile."},
		{slug: "macos-import", category: "macOS", zh: "macOS 导入教程", en: "macOS setup", bodyZH: "安装 Clash Verge Rev，添加新的订阅配置，粘贴链接并启用系统代理。", bodyEN: "Install Clash Verge Rev, add a remote profile, paste the URL, and enable the system proxy."},
		{slug: "android-import", category: "Android", zh: "Android 导入教程", en: "Android setup", bodyZH: "从官方发布页安装 Hiddify，在首页点击添加配置并从剪贴板导入订阅。", bodyEN: "Install Hiddify from its official release page, add a profile, and import the subscription from the clipboard."},
		{slug: "ios-import", category: "iOS", zh: "iOS 导入教程", en: "iOS setup", bodyZH: "从 App Store 安装 Shadowrocket，点击右上角加号，选择 Subscribe 并粘贴订阅链接。", bodyEN: "Install Shadowrocket from the App Store, tap the plus button, choose Subscribe, and paste the subscription URL."},
		{slug: "linux-import", category: "Linux", zh: "Linux 导入教程", en: "Linux setup", bodyZH: "安装 v2rayN、Clash Verge Rev 或 Hiddify，从 URL 导入订阅并选择可用节点。", bodyEN: "Install v2rayN, Clash Verge Rev, or Hiddify, import the profile by URL, and select an available endpoint."},
	}
	for index, seed := range seeds {
		row := model.KnowledgeArticle{ID: uuid.NewString(), Slug: seed.slug, Category: seed.category, TitleI18n: localizedText(seed.zh, seed.en), ContentI18n: localizedText(seed.bodyZH, seed.bodyEN), Published: true, SortOrder: index + 1}
		if err := tx.Create(&row).Error; err != nil {
			return err
		}
	}
	return nil
}

func localizedText(zh, en string) string {
	value, _ := json.Marshal(map[string]string{"zh-CN": zh, "en-US": en})
	return string(value)
}
