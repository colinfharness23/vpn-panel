package commercial

import (
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
	"github.com/mhsanaei/3x-ui/v3/internal/web/entity"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	defaultCommissionPercent   = 10
	defaultMaxInviteCodes      = 5
	commissionAutoConfirmDelay = 3 * 24 * time.Hour
)

type InvitationPolicy struct {
	ForcedInvitation           bool
	CommissionPercent          int
	MaxInviteCodes             int
	InviteCodesNeverExpire     bool
	CommissionFirstPaymentOnly bool
	CommissionAutoConfirm      bool
	MultiLevelEnabled          bool
}

type commissionShare struct {
	CustomerID string `json:"customerId"`
	Level      int    `json:"level"`
	AmountFen  int64  `json:"amountFen"`
}

func (s *ConfigStore) InvitationPolicy() InvitationPolicy {
	percent := configInt(s, "invitation.commission_percent", defaultCommissionPercent)
	if percent < 0 || percent > 100 {
		percent = defaultCommissionPercent
	}
	maxCodes := configInt(s, "invitation.max_codes", defaultMaxInviteCodes)
	if maxCodes < 1 || maxCodes > 100 {
		maxCodes = defaultMaxInviteCodes
	}
	multiLevel := configBool(s, "invitation.multi_level", false)
	if multiLevel && percent*3 > 100 {
		multiLevel = false
	}
	return InvitationPolicy{
		ForcedInvitation:           configBool(s, "invitation.forced", false),
		CommissionPercent:          percent,
		MaxInviteCodes:             maxCodes,
		InviteCodesNeverExpire:     configBool(s, "invitation.codes_never_expire", true),
		CommissionFirstPaymentOnly: configBool(s, "invitation.first_payment_only", true),
		CommissionAutoConfirm:      configBool(s, "invitation.auto_confirm", true),
		MultiLevelEnabled:          multiLevel,
	}
}

func (s *AdminService) InvitationSettings() entity.CommercialInvitationSettings {
	policy := s.config.InvitationPolicy()
	return entity.CommercialInvitationSettings{
		ForcedInvitation:           policy.ForcedInvitation,
		CommissionPercent:          policy.CommissionPercent,
		MaxInviteCodes:             policy.MaxInviteCodes,
		InviteCodesNeverExpire:     policy.InviteCodesNeverExpire,
		CommissionFirstPaymentOnly: policy.CommissionFirstPaymentOnly,
		CommissionAutoConfirm:      policy.CommissionAutoConfirm,
		MultiLevelEnabled:          policy.MultiLevelEnabled,
	}
}

func (s *AdminService) SaveInvitationSettings(request entity.CommercialInvitationSettings) error {
	if request.CommissionPercent < 0 || request.CommissionPercent > 100 {
		return errors.New("邀请佣金百分比必须在 0 到 100 之间")
	}
	if request.MaxInviteCodes < 1 || request.MaxInviteCodes > 100 {
		return errors.New("用户可创建邀请码上限必须在 1 到 100 之间")
	}
	if request.MultiLevelEnabled && request.CommissionPercent*3 > 100 {
		return errors.New("开启三级分销时，三级佣金比例合计不能超过 100%")
	}
	return s.config.SetMany(map[string]string{
		"invitation.forced":             strconv.FormatBool(request.ForcedInvitation),
		"invitation.commission_percent": strconv.Itoa(request.CommissionPercent),
		"invitation.max_codes":          strconv.Itoa(request.MaxInviteCodes),
		"invitation.codes_never_expire": strconv.FormatBool(request.InviteCodesNeverExpire),
		"invitation.first_payment_only": strconv.FormatBool(request.CommissionFirstPaymentOnly),
		"invitation.auto_confirm":       strconv.FormatBool(request.CommissionAutoConfirm),
		"invitation.multi_level":        strconv.FormatBool(request.MultiLevelEnabled),
	})
}

func commissionShares(row *model.InvitationCommission) []commissionShare {
	var shares []commissionShare
	if strings.TrimSpace(row.Distribution) != "" && json.Unmarshal([]byte(row.Distribution), &shares) == nil && len(shares) > 0 {
		return shares
	}
	return []commissionShare{{CustomerID: row.InviterID, Level: 1, AmountFen: row.AmountFen}}
}

func settleCommission(db *gorm.DB, id string) error {
	now := time.Now().UTC()
	return db.Transaction(func(tx *gorm.DB) error {
		var row model.InvitationCommission
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND status IN ?", id, []string{"pending", "confirmed"}).First(&row).Error; err != nil {
			return err
		}
		for _, share := range commissionShares(&row) {
			if share.CustomerID == "" || share.AmountFen <= 0 {
				continue
			}
			result := tx.Model(&model.Customer{}).Where("id = ?", share.CustomerID).UpdateColumn("balance_fen", gorm.Expr("balance_fen + ?", share.AmountFen))
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected != 1 {
				return errors.New("佣金收款客户不存在")
			}
		}
		return tx.Model(&row).Updates(map[string]any{"status": "settled", "settled_at": now}).Error
	})
}
