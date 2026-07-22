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
		if err := syncCommercialPlanCatalog(tx); err != nil {
			return err
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
		if err := normalizeCommercialClientCatalog(tx); err != nil {
			return err
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
		if err := repairWelcomeNoticeEnglish(tx); err != nil {
			return err
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

type commercialPlanPriceSeed struct {
	period string
	months int
	amount int64
}

type commercialPlanSeed struct {
	slug         string
	name         string
	description  string
	traffic      int64
	devices      int
	speedMbps    int
	relayEnabled bool
	relayLimit   int
	benefits     map[string]string
	prices       []commercialPlanPriceSeed
}

func commercialPlanSeeds() []commercialPlanSeed {
	const (
		included    = "包含"
		notIncluded = "不包含"
		gb          = int64(1024 * 1024 * 1024)
	)
	return []commercialPlanSeed{
		{
			slug: "starter", name: "全球畅游版", description: "适合日常个人使用",
			traffic: 150 * gb, devices: 3, speedMbps: 200,
			benefits: map[string]string{
				"globalCoverage": included, "standardNodes": included, "advancedNodes": notIncluded,
				"premiumRoutes": notIncluded, "residentialIpSale": notIncluded,
				"socialMedia": "日常使用", "crossBorderWork": "基础使用", "liveStreaming": "不推荐",
				"uploadOptimization": notIncluded, "peakPriority": "标准", "failover": "基础", "support": "标准",
			},
			prices: []commercialPlanPriceSeed{{"monthly", 1, 1000}, {"quarterly", 3, 2700}, {"yearly", 12, 9600}},
		},
		{
			slug: "pro", name: "跨境增长版", description: "适合跨境业务运营",
			traffic: 500 * gb, devices: 8, speedMbps: 500, relayEnabled: true, relayLimit: 1,
			benefits: map[string]string{
				"globalCoverage": included, "standardNodes": included, "advancedNodes": "部分开放",
				"premiumRoutes": "精选线路", "residentialIpSale": notIncluded,
				"socialMedia": "高频运营", "crossBorderWork": "推荐", "liveStreaming": "1080P直播",
				"uploadOptimization": included, "peakPriority": "优先", "failover": "快速切换", "support": "优先",
			},
			prices: []commercialPlanPriceSeed{{"monthly", 1, 2000}, {"quarterly", 3, 5400}, {"yearly", 12, 19200}},
		},
		{
			slug: "ultimate", name: "直播旗舰版", description: "适合专业直播及团队",
			traffic: 1500 * gb, devices: 15, speedMbps: 1000, relayEnabled: true, relayLimit: 5,
			benefits: map[string]string{
				"globalCoverage": included, "standardNodes": included, "advancedNodes": "全部开放",
				"premiumRoutes": "全部线路", "residentialIpSale": notIncluded,
				"socialMedia": "专业运营", "crossBorderWork": "专业级", "liveStreaming": "4K及长时间直播",
				"uploadOptimization": "高优先级", "peakPriority": "最高", "failover": "优先切换", "support": "专属优先",
			},
			prices: []commercialPlanPriceSeed{{"monthly", 1, 4500}, {"half_yearly", 6, 22900}, {"yearly", 12, 40500}},
		},
	}
}

func seedCommercialPlans(tx *gorm.DB) error {
	for index, seed := range commercialPlanSeeds() {
		if _, err := createCommercialPlanSeed(tx, seed, index+1); err != nil {
			return err
		}
	}
	return nil
}

func createCommercialPlanSeed(tx *gorm.DB, seed commercialPlanSeed, sortOrder int) (*model.Plan, error) {
	plan := &model.Plan{
		ID: uuid.NewString(), Slug: seed.slug, Name: seed.name, Description: seed.description,
		TrafficBytes: seed.traffic, DeviceLimit: seed.devices, UploadLimitMbps: seed.speedMbps,
		DownloadLimitMbps: seed.speedMbps, ResidentialRelayEnabled: seed.relayEnabled,
		ResidentialRelayLimit: seed.relayLimit, ResetCycle: "monthly", NodeGroup: "default",
		Visibility: "public", Renewable: true, Upgradable: true, Active: false,
		SortOrder: sortOrder, ProvisionInboundIDs: "[]", DisplayBenefits: seed.benefits,
	}
	if err := tx.Create(plan).Error; err != nil {
		return nil, err
	}
	if err := tx.Model(plan).Update("active", false).Error; err != nil {
		return nil, err
	}
	for _, price := range seed.prices {
		row := model.PlanPrice{ID: uuid.NewString(), PlanID: plan.ID, BillingPeriod: price.period, Months: price.months, AmountFen: price.amount, Active: true}
		if err := tx.Create(&row).Error; err != nil {
			return nil, err
		}
	}
	return plan, nil
}

func syncCommercialPlanCatalog(tx *gorm.DB) error {
	const seederName = "CommercialPlanCatalogV2"
	var seeded int64
	if err := tx.Model(&model.HistoryOfSeeders{}).Where("seeder_name = ?", seederName).Count(&seeded).Error; err != nil || seeded > 0 {
		return err
	}
	legacyTraffic := map[string]int64{
		"starter":  100 * 1024 * 1024 * 1024,
		"pro":      300 * 1024 * 1024 * 1024,
		"ultimate": 1024 * 1024 * 1024 * 1024,
	}
	legacyDevices := map[string]int{"starter": 3, "pro": 5, "ultimate": 10}
	for index, seed := range commercialPlanSeeds() {
		var plan model.Plan
		err := tx.Where("slug = ?", seed.slug).First(&plan).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			if _, err := createCommercialPlanSeed(tx, seed, index+1); err != nil {
				return err
			}
			continue
		}
		if err != nil {
			return err
		}
		benefitsJSON, _ := json.Marshal(seed.benefits)
		updates := map[string]any{"display_benefits": string(benefitsJSON)}
		isUntouchedLegacy := plan.TrafficBytes == legacyTraffic[seed.slug] &&
			plan.DeviceLimit == legacyDevices[seed.slug] && plan.UploadLimitMbps == 0 && plan.DownloadLimitMbps == 0
		if isUntouchedLegacy {
			updates["name"] = seed.name
			updates["description"] = seed.description
			updates["traffic_bytes"] = seed.traffic
			updates["device_limit"] = seed.devices
			updates["upload_limit_mbps"] = seed.speedMbps
			updates["download_limit_mbps"] = seed.speedMbps
			updates["residential_relay_enabled"] = seed.relayEnabled
			updates["residential_relay_limit"] = seed.relayLimit
		}
		if err := tx.Model(&plan).Updates(updates).Error; err != nil {
			return err
		}
		var priceCount int64
		if err := tx.Model(&model.PlanPrice{}).Where("plan_id = ?", plan.ID).Count(&priceCount).Error; err != nil {
			return err
		}
		if priceCount == 0 {
			for _, price := range seed.prices {
				row := model.PlanPrice{ID: uuid.NewString(), PlanID: plan.ID, BillingPeriod: price.period, Months: price.months, AmountFen: price.amount, Active: true}
				if err := tx.Create(&row).Error; err != nil {
					return err
				}
			}
		}
	}
	return tx.Create(&model.HistoryOfSeeders{SeederName: seederName}).Error
}

func seedCommercialApplications(tx *gorm.DB) error {
	apps := []model.ClientApplication{
		{ID: uuid.NewString(), Slug: "v2rayn", Name: "v2rayN", Platform: "Windows / macOS / Linux", Description: "适合桌面设备使用，支持订阅导入与更新", Active: true, SortOrder: 1},
		{ID: uuid.NewString(), Slug: "clash-verge-rev", Name: "Clash Verge Rev", Platform: "Windows / macOS / Linux", Description: "界面清晰的跨平台桌面客户端", Active: true, SortOrder: 2},
		{ID: uuid.NewString(), Slug: "shadowrocket", Name: "Shadowrocket", Platform: "iOS", Description: "适合 iPhone 和 iPad 使用的订阅客户端", Active: true, SortOrder: 3},
		{ID: uuid.NewString(), Slug: "v2rayng", Name: "v2rayNG", Platform: "Android", Description: "适合 Android 设备使用，支持订阅导入与更新", Active: true, SortOrder: 4},
	}
	return tx.Create(&apps).Error
}

func normalizeCommercialClientCatalog(tx *gorm.DB) error {
	var hiddify model.ClientApplication
	if err := tx.Where("slug = ?", "hiddify").First(&hiddify).Error; err == nil {
		var v2rayNGCount int64
		if err := tx.Model(&model.ClientApplication{}).Where("slug = ?", "v2rayng").Count(&v2rayNGCount).Error; err != nil {
			return err
		}
		if v2rayNGCount == 0 {
			if err := tx.Model(&hiddify).Updates(map[string]any{
				"slug": "v2rayng", "name": "v2rayNG", "platform": "Android",
				"official_url": "", "source_url": "",
				"description": "适合 Android 设备使用，支持订阅导入与更新",
			}).Error; err != nil {
				return err
			}
		} else if err := tx.Model(&hiddify).Update("active", false).Error; err != nil {
			return err
		}
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	androidZH := "从本站下载并安装 v2rayNG，在首页点击添加配置并从剪贴板导入订阅。"
	androidEN := "Download and install v2rayNG from this site, add a profile, and import the subscription from the clipboard."
	if err := tx.Model(&model.KnowledgeArticle{}).
		Where("slug = ? AND content_i18n LIKE ?", "android-import", "%Hiddify%").
		Update("content_i18n", localizedText(androidZH, androidEN)).Error; err != nil {
		return err
	}
	linuxZH := "从本站下载并安装 v2rayN 或 Clash Verge Rev，从 URL 导入订阅并选择可用节点。"
	linuxEN := "Download and install v2rayN or Clash Verge Rev from this site, import the profile by URL, and select an available endpoint."
	return tx.Model(&model.KnowledgeArticle{}).
		Where("slug = ? AND content_i18n LIKE ?", "linux-import", "%Hiddify%").
		Update("content_i18n", localizedText(linuxZH, linuxEN)).Error
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
		{slug: "android-import", category: "Android", zh: "Android 导入教程", en: "Android setup", bodyZH: "从本站下载并安装 v2rayNG，在首页点击添加配置并从剪贴板导入订阅。", bodyEN: "Download and install v2rayNG from this site, add a profile, and import the subscription from the clipboard."},
		{slug: "ios-import", category: "iOS", zh: "iOS 导入教程", en: "iOS setup", bodyZH: "从 App Store 安装 Shadowrocket，点击右上角加号，选择 Subscribe 并粘贴订阅链接。", bodyEN: "Install Shadowrocket from the App Store, tap the plus button, choose Subscribe, and paste the subscription URL."},
		{slug: "linux-import", category: "Linux", zh: "Linux 导入教程", en: "Linux setup", bodyZH: "从本站下载并安装 v2rayN 或 Clash Verge Rev，从 URL 导入订阅并选择可用节点。", bodyEN: "Download and install v2rayN or Clash Verge Rev from this site, import the profile by URL, and select an available endpoint."},
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

// A lazy-tab bug in the commercial editor used to erase every locale except
// the currently visible one. Repair only the recognizable built-in welcome
// notice and never overwrite an administrator-authored English translation.
func repairWelcomeNoticeEnglish(tx *gorm.DB) error {
	var row model.Notice
	if err := tx.Where("slug = ?", "welcome").First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}
	var titles, contents map[string]string
	if json.Unmarshal([]byte(row.TitleI18n), &titles) != nil || json.Unmarshal([]byte(row.ContentI18n), &contents) != nil {
		return nil
	}
	zh := strings.TrimSpace(contents["zh-CN"])
	if !strings.Contains(zh, "选购前可先查看套餐说明与使用文档") || !strings.Contains(zh, "重要信息会集中显示在账户中") {
		return nil
	}
	changed := false
	if strings.TrimSpace(titles["en-US"]) == "" {
		titles["en-US"] = "Service notice"
		changed = true
	}
	if strings.TrimSpace(contents["en-US"]) == "" {
		siteName := "NOVA"
		var setting model.CommercialSetting
		if err := tx.Where("key = ?", "site.name").First(&setting).Error; err == nil && strings.TrimSpace(setting.Value) != "" {
			siteName = strings.TrimSpace(setting.Value)
		} else if rest, ok := strings.CutPrefix(zh, "欢迎使用 "); ok {
			if inferred, _, found := strings.Cut(rest, "。"); found && strings.TrimSpace(inferred) != "" {
				siteName = strings.TrimSpace(inferred)
			}
		}
		contents["en-US"] = fmt.Sprintf("Welcome to %s. Review the plan details and documentation before purchase; important setup information will stay available in your account.", siteName)
		changed = true
	}
	if !changed {
		return nil
	}
	titleJSON, err := json.Marshal(titles)
	if err != nil {
		return err
	}
	contentJSON, err := json.Marshal(contents)
	if err != nil {
		return err
	}
	return tx.Model(&model.Notice{}).Where("id = ?", row.ID).Updates(map[string]any{
		"title_i18n": string(titleJSON), "content_i18n": string(contentJSON),
	}).Error
}
