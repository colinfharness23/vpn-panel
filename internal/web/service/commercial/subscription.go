package commercial

import (
	"errors"
	"strconv"
	"strings"

	"github.com/mhsanaei/3x-ui/v3/internal/web/entity"
)

const (
	SubscriptionEventNone         = "none"
	SubscriptionEventResetTraffic = "reset_traffic"

	MonthlyResetCalendar    = "calendar_month"
	MonthlyResetAnniversary = "billing_cycle"
	MonthlyResetNever       = "never"
)

type SubscriptionPolicy struct {
	AllowUserChange        bool
	MonthlyResetMode       string
	OffsetEnabled          bool
	PurchaseEvent          string
	RenewalEvent           string
	ChangeEvent            string
	ShowSubscriptionInfo   bool
	ShowProtocolInNodeName bool
}

func normalizeSubscriptionEvent(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == SubscriptionEventResetTraffic {
		return value
	}
	return SubscriptionEventNone
}

func normalizeMonthlyResetMode(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	switch value {
	case MonthlyResetCalendar, MonthlyResetAnniversary, MonthlyResetNever:
		return value
	default:
		return MonthlyResetCalendar
	}
}

func (s *ConfigStore) SubscriptionPolicy() SubscriptionPolicy {
	return SubscriptionPolicy{
		AllowUserChange:        configBool(s, "subscription.allow_user_change", true),
		MonthlyResetMode:       normalizeMonthlyResetMode(s.GetDefault("subscription.monthly_reset_mode", MonthlyResetCalendar)),
		OffsetEnabled:          configBool(s, "subscription.offset_enabled", false),
		PurchaseEvent:          normalizeSubscriptionEvent(s.GetDefault("subscription.purchase_event", SubscriptionEventNone)),
		RenewalEvent:           normalizeSubscriptionEvent(s.GetDefault("subscription.renewal_event", SubscriptionEventNone)),
		ChangeEvent:            normalizeSubscriptionEvent(s.GetDefault("subscription.change_event", SubscriptionEventNone)),
		ShowSubscriptionInfo:   configBool(s, "subscription.show_info", false),
		ShowProtocolInNodeName: configBool(s, "subscription.show_protocol_in_name", false),
	}
}

func (p SubscriptionPolicy) EventFor(orderKind string) string {
	switch orderKind {
	case "renewal":
		return p.RenewalEvent
	case "upgrade":
		return p.ChangeEvent
	default:
		return p.PurchaseEvent
	}
}

func (s *AdminService) SubscriptionSettings() entity.CommercialSubscriptionSettings {
	policy := s.config.SubscriptionPolicy()
	return entity.CommercialSubscriptionSettings{
		AllowUserChange:        policy.AllowUserChange,
		MonthlyResetMode:       policy.MonthlyResetMode,
		OffsetEnabled:          policy.OffsetEnabled,
		PurchaseEvent:          policy.PurchaseEvent,
		RenewalEvent:           policy.RenewalEvent,
		ChangeEvent:            policy.ChangeEvent,
		ShowSubscriptionInfo:   policy.ShowSubscriptionInfo,
		ShowProtocolInNodeName: policy.ShowProtocolInNodeName,
	}
}

func (s *AdminService) SaveSubscriptionSettings(request entity.CommercialSubscriptionSettings) error {
	resetMode := normalizeMonthlyResetMode(request.MonthlyResetMode)
	if resetMode != strings.ToLower(strings.TrimSpace(request.MonthlyResetMode)) {
		return errors.New("月流量重置方式无效")
	}
	for _, event := range []string{request.PurchaseEvent, request.RenewalEvent, request.ChangeEvent} {
		normalized := strings.ToLower(strings.TrimSpace(event))
		if normalized != SubscriptionEventNone && normalized != SubscriptionEventResetTraffic {
			return errors.New("订阅触发事件无效")
		}
	}
	return s.config.SetMany(map[string]string{
		"subscription.allow_user_change":     strconv.FormatBool(request.AllowUserChange),
		"subscription.monthly_reset_mode":    resetMode,
		"subscription.offset_enabled":        strconv.FormatBool(request.OffsetEnabled),
		"subscription.purchase_event":        normalizeSubscriptionEvent(request.PurchaseEvent),
		"subscription.renewal_event":         normalizeSubscriptionEvent(request.RenewalEvent),
		"subscription.change_event":          normalizeSubscriptionEvent(request.ChangeEvent),
		"subscription.show_info":             strconv.FormatBool(request.ShowSubscriptionInfo),
		"subscription.show_protocol_in_name": strconv.FormatBool(request.ShowProtocolInNodeName),
	})
}

func resetDaysForPolicy(cycle, monthlyMode string) int {
	if cycle == "monthly" && normalizeMonthlyResetMode(monthlyMode) == MonthlyResetNever {
		return 0
	}
	return resetDays(cycle)
}
