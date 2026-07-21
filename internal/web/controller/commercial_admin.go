package controller

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
	"github.com/mhsanaei/3x-ui/v3/internal/web/entity"
	"github.com/mhsanaei/3x-ui/v3/internal/web/service/commercial"
	"github.com/mhsanaei/3x-ui/v3/internal/web/service/panel"
	"github.com/mhsanaei/3x-ui/v3/internal/web/session"

	"github.com/gin-gonic/gin"
)

type CommercialAdminController struct {
	BaseController
	service      *commercial.AdminService
	orderService *commercial.OrderService
	lineService  *commercial.LineService
	userService  panel.UserService
}

func NewCommercialAdminController(g *gin.RouterGroup) *CommercialAdminController {
	controller := &CommercialAdminController{service: commercial.NewAdminService(), orderService: commercial.NewOrderService(), lineService: commercial.NewLineService()}
	controller.initRouter(g)
	return controller
}

func (a *CommercialAdminController) initRouter(g *gin.RouterGroup) {
	g.GET("/overview", a.require("read"), a.overview)
	g.GET("/customers", a.require("customers.read"), a.customers)
	g.POST("/customers", a.require("customers.manage"), a.createCustomer)
	g.PATCH("/customers/:id", a.require("customers.manage"), a.updateCustomer)
	g.PUT("/customers/:id/subscription", a.require("customers.manage"), a.requireReauth, a.upsertCustomerSubscription)
	g.DELETE("/customers/:id/subscription", a.require("customers.manage"), a.requireReauth, a.deleteCustomerSubscription)
	g.DELETE("/customers/:id", a.require("customers.manage"), a.requireReauth, a.deleteCustomer)
	g.DELETE("/customers", a.require("customers.manage"), a.requireReauth, a.deleteCustomers)
	g.GET("/plans", a.require("plans.read"), a.plans)
	g.POST("/plans", a.require("plans.manage"), a.savePlan)
	g.POST("/plan-prices", a.require("plans.manage"), a.savePlanPrice)
	g.GET("/line-sources", a.require("lines.read"), a.lineSources)
	g.POST("/line-sources", a.require("lines.manage"), a.saveLineSource)
	g.POST("/line-sources/:id/refresh", a.require("lines.manage"), a.refreshLineSource)
	g.DELETE("/line-sources/:id", a.require("lines.manage"), a.requireReauth, a.deleteLineSource)
	g.POST("/line-imports/preview", a.require("lines.read"), a.previewLineImport)
	g.POST("/line-imports/commit", a.require("lines.manage"), a.commitLineImport)
	g.GET("/line-nodes", a.require("lines.read"), a.lineNodes)
	g.PUT("/line-nodes/groups", a.require("lines.manage"), a.assignLineNodeGroups)
	g.POST("/line-nodes/:id/probe", a.require("lines.manage"), a.probeLineNode)
	g.GET("/line-groups", a.require("lines.read"), a.lineGroups)
	g.POST("/line-groups", a.require("lines.manage"), a.saveLineGroup)
	g.DELETE("/line-groups/:id", a.require("lines.manage"), a.requireReauth, a.deleteLineGroup)
	g.GET("/orders", a.require("orders.read"), a.orders)
	g.POST("/orders/:id/retry", a.require("orders.manage"), a.retryOrder)
	g.GET("/notices", a.require("content.read"), a.notices)
	g.POST("/notices", a.require("content.manage"), a.saveNotice)
	g.GET("/articles", a.require("content.read"), a.articles)
	g.POST("/articles", a.require("content.manage"), a.saveArticle)
	g.GET("/applications", a.require("content.read"), a.applications)
	g.POST("/applications", a.require("content.manage"), a.saveApplication)
	g.POST("/applications/:id/package", a.require("content.manage"), a.uploadApplicationPackage)
	g.GET("/tickets", a.require("tickets.read"), a.tickets)
	g.GET("/tickets/:id/messages", a.require("tickets.read"), a.ticketMessages)
	g.POST("/tickets/:id/reply", a.require("tickets.manage"), a.replyTicket)
	g.GET("/coupons", a.require("marketing.read"), a.coupons)
	g.POST("/coupons", a.require("marketing.manage"), a.saveCoupon)
	g.GET("/gift-cards", a.require("finance.read"), a.giftCards)
	g.POST("/gift-cards", a.require("finance.manage"), a.requireReauth, a.createGiftCards)
	g.GET("/commissions", a.require("finance.read"), a.commissions)
	g.POST("/commissions/:id/settle", a.require("finance.manage"), a.requireReauth, a.settleCommission)
	g.GET("/settings", a.require("settings.read"), a.settings)
	g.POST("/settings", a.require("settings.manage"), a.requireReauth, a.saveSetting)
	g.PUT("/payment-settings", a.require("settings.manage"), a.requireReauth, a.savePaymentSettings)
	g.GET("/site-settings", a.require("settings.read"), a.siteSettings)
	g.PUT("/site-settings", a.require("settings.manage"), a.saveSiteSettings)
	g.PUT("/site-settings/logo", a.require("settings.manage"), a.saveSiteLogo)
	g.GET("/security-settings", a.require("settings.read"), a.securitySettings)
	g.PUT("/security-settings", a.require("settings.manage"), a.saveSecuritySettings)
	g.GET("/subscription-settings", a.require("settings.read"), a.subscriptionSettings)
	g.PUT("/subscription-settings", a.require("settings.manage"), a.saveSubscriptionSettings)
	g.GET("/invitation-settings", a.require("settings.read"), a.invitationSettings)
	g.PUT("/invitation-settings", a.require("settings.manage"), a.saveInvitationSettings)
	g.GET("/email-templates", a.require("settings.read"), a.emailTemplates)
	g.PUT("/email-templates/:key", a.require("settings.manage"), a.saveEmailTemplate)
	g.POST("/emails/send", a.require("settings.manage"), a.requireReauth, a.sendCustomerEmail)
	g.GET("/audit", a.require("audit.read"), a.auditLogs)
	g.GET("/role", a.require("read"), a.currentRole)
	g.GET("/roles", a.require("roles.read"), a.adminUsers)
	g.PUT("/roles/:userId", a.require("roles.manage"), a.requireReauth, a.setRole)
}

func (a *CommercialAdminController) require(permission string) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := session.GetLoginUser(c)
		if user == nil {
			commercialJSON(c, nil, errors.New("管理员登录已失效"))
			c.Abort()
			return
		}
		role := a.service.Role(user.Id)
		if !roleAllows(role, permission) {
			c.AbortWithStatusJSON(403, entity.Msg{Success: false, Msg: "当前角色无权执行此操作"})
			return
		}
		c.Set("commercial_admin_role", role)
		c.Next()
	}
}

func (a *CommercialAdminController) requireReauth(c *gin.Context) {
	user := session.GetLoginUser(c)
	password := c.GetHeader("X-Admin-Password")
	twoFactorCode := c.GetHeader("X-Admin-2FA")
	if user == nil || password == "" {
		c.AbortWithStatusJSON(428, entity.Msg{Success: false, Msg: "敏感操作需要重新验证管理员密码与 2FA"})
		return
	}
	checked, err := a.userService.CheckUser(user.Username, password, twoFactorCode)
	if err != nil || checked == nil || checked.Id != user.Id {
		c.AbortWithStatusJSON(403, entity.Msg{Success: false, Msg: "管理员重新验证失败"})
		return
	}
	c.Next()
}

func (a *CommercialAdminController) overview(c *gin.Context) {
	data, err := a.service.Overview()
	commercialJSON(c, data, err)
}

func (a *CommercialAdminController) customers(c *gin.Context) {
	data, err := a.service.Customers(c.Query("search"), c.Query("status"), queryInt(c, "page"), queryInt(c, "pageSize"))
	commercialJSON(c, data, err)
}

func (a *CommercialAdminController) createCustomer(c *gin.Context) {
	var request entity.CommercialCustomerCreateRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		commercialJSON(c, nil, errors.New("请求格式无效"))
		return
	}
	row, err := a.service.CreateCustomer(request)
	if row != nil {
		a.audit(c, "customer.create", "customer", row.ID, gin.H{"email": row.Email})
	}
	commercialJSON(c, row, err)
}

func (a *CommercialAdminController) updateCustomer(c *gin.Context) {
	var request entity.CommercialCustomerUpdateRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		commercialJSON(c, nil, errors.New("请求格式无效"))
		return
	}
	err := a.service.UpdateCustomer(c.Param("id"), request.Status, request.BalanceFen)
	a.audit(c, "customer.update", "customer", c.Param("id"), request)
	commercialJSON(c, gin.H{"updated": err == nil}, err)
}

func (a *CommercialAdminController) upsertCustomerSubscription(c *gin.Context) {
	var request entity.CommercialSubscriptionUpdateRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		commercialJSON(c, nil, errors.New("请求格式无效"))
		return
	}
	row, err := a.service.UpsertSubscription(c.Param("id"), request)
	if err == nil {
		a.audit(c, "customer.subscription_upsert", "customer", c.Param("id"), gin.H{"planId": request.PlanID, "resetTraffic": request.ResetTraffic})
	}
	commercialJSON(c, row, err)
}

func (a *CommercialAdminController) deleteCustomerSubscription(c *gin.Context) {
	err := a.service.DeleteSubscription(c.Param("id"))
	if err == nil {
		a.audit(c, "customer.subscription_delete", "customer", c.Param("id"), nil)
	}
	commercialJSON(c, gin.H{"deleted": err == nil}, err)
}

func (a *CommercialAdminController) deleteCustomer(c *gin.Context) {
	result, err := a.service.DeleteCustomers([]string{c.Param("id")})
	if err == nil && len(result.Deleted) > 0 {
		a.audit(c, "customer.delete", "customer_deleted", "redacted", gin.H{"count": 1})
	}
	commercialJSON(c, result, err)
}

func (a *CommercialAdminController) deleteCustomers(c *gin.Context) {
	var request entity.CommercialCustomerDeleteRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		commercialJSON(c, nil, errors.New("请求格式无效"))
		return
	}
	result, err := a.service.DeleteCustomers(request.IDs)
	if err == nil && len(result.Deleted) > 0 {
		a.audit(c, "customer.delete_batch", "customer_deleted", "redacted", gin.H{"count": len(result.Deleted), "failed": len(result.Failed)})
	}
	commercialJSON(c, result, err)
}

func (a *CommercialAdminController) plans(c *gin.Context) {
	data, err := a.orderService.Catalog(false)
	commercialJSON(c, data, err)
}

func (a *CommercialAdminController) savePlan(c *gin.Context) {
	var request entity.CommercialPlanRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		commercialJSON(c, nil, errors.New("请求格式无效"))
		return
	}
	row, err := a.service.SavePlan(request)
	if row != nil {
		a.audit(c, "plan.save", "plan", row.ID, request)
	}
	commercialJSON(c, row, err)
}

func (a *CommercialAdminController) savePlanPrice(c *gin.Context) {
	var request entity.CommercialPlanPriceRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		commercialJSON(c, nil, errors.New("请求格式无效"))
		return
	}
	row, err := a.service.SavePlanPrice(request)
	if row != nil {
		a.audit(c, "plan_price.save", "plan_price", row.ID, request)
	}
	commercialJSON(c, row, err)
}

func (a *CommercialAdminController) lineSources(c *gin.Context) {
	rows, err := a.lineService.Sources()
	commercialJSON(c, rows, err)
}

func (a *CommercialAdminController) saveLineSource(c *gin.Context) {
	var request entity.CommercialLineSourceRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		commercialJSON(c, nil, errors.New("请求格式无效"))
		return
	}
	row, err := a.lineService.SaveURLSource(c.Request.Context(), request)
	if err == nil && row != nil {
		a.audit(c, "line_source.save", "line_source", row.ID, gin.H{"name": row.Name, "groups": request.GroupIDs, "plans": request.PlanIDs})
	}
	commercialJSON(c, row, err)
}

func (a *CommercialAdminController) refreshLineSource(c *gin.Context) {
	err := a.lineService.QueueRefresh(c.Param("id"))
	if err == nil {
		a.audit(c, "line_source.refresh", "line_source", c.Param("id"), nil)
	}
	commercialJSON(c, gin.H{"queued": err == nil}, err)
}

func (a *CommercialAdminController) deleteLineSource(c *gin.Context) {
	err := a.lineService.DeleteSource(c.Param("id"))
	if err == nil {
		a.audit(c, "line_source.delete", "line_source", c.Param("id"), nil)
	}
	commercialJSON(c, gin.H{"deleted": err == nil}, err)
}

func (a *CommercialAdminController) previewLineImport(c *gin.Context) {
	var request entity.CommercialLineImportRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		commercialJSON(c, nil, errors.New("请求格式无效"))
		return
	}
	result, err := a.lineService.PreviewImport(request.Links)
	commercialJSON(c, result, err)
}

func (a *CommercialAdminController) commitLineImport(c *gin.Context) {
	var request entity.CommercialLineImportRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		commercialJSON(c, nil, errors.New("请求格式无效"))
		return
	}
	row, err := a.lineService.CommitImport(request)
	if err == nil && row != nil {
		a.audit(c, "line_import.commit", "line_source", row.ID, gin.H{"name": row.Name, "nodes": row.NodeCount})
	}
	commercialJSON(c, row, err)
}

func (a *CommercialAdminController) lineNodes(c *gin.Context) {
	rows, err := a.lineService.Nodes(c.Query("sourceId"), c.Query("groupId"), c.Query("status"))
	commercialJSON(c, rows, err)
}

func (a *CommercialAdminController) assignLineNodeGroups(c *gin.Context) {
	var request entity.CommercialLineNodeGroupsRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		commercialJSON(c, nil, errors.New("请求格式无效"))
		return
	}
	err := a.lineService.AssignNodeGroups(request)
	if err == nil {
		a.audit(c, "line_node.assign_groups", "line_node", "batch", gin.H{"nodes": request.NodeIDs, "groups": request.GroupIDs})
	}
	commercialJSON(c, gin.H{"updated": err == nil}, err)
}

func (a *CommercialAdminController) probeLineNode(c *gin.Context) {
	err := a.lineService.QueueProbe(c.Param("id"))
	if err == nil {
		a.audit(c, "line_node.probe", "line_node", c.Param("id"), nil)
	}
	commercialJSON(c, gin.H{"queued": err == nil}, err)
}

func (a *CommercialAdminController) lineGroups(c *gin.Context) {
	rows, err := a.lineService.Groups()
	commercialJSON(c, rows, err)
}

func (a *CommercialAdminController) saveLineGroup(c *gin.Context) {
	var request entity.CommercialLineGroupRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		commercialJSON(c, nil, errors.New("请求格式无效"))
		return
	}
	row, err := a.lineService.SaveGroup(request)
	if err == nil && row != nil {
		a.audit(c, "line_group.save", "line_group", row.ID, gin.H{"name": row.Name, "active": row.Active})
	}
	commercialJSON(c, row, err)
}

func (a *CommercialAdminController) deleteLineGroup(c *gin.Context) {
	err := a.lineService.DeleteGroup(c.Param("id"))
	if err == nil {
		a.audit(c, "line_group.delete", "line_group", c.Param("id"), nil)
	}
	commercialJSON(c, gin.H{"deleted": err == nil}, err)
}

func (a *CommercialAdminController) orders(c *gin.Context) {
	data, err := a.service.Orders(c.Query("search"), c.Query("status"), queryInt(c, "page"), queryInt(c, "pageSize"))
	commercialJSON(c, data, err)
}

func (a *CommercialAdminController) retryOrder(c *gin.Context) {
	err := a.service.RetryProvisioning(c.Param("id"))
	a.audit(c, "order.provisioning_retry", "order", c.Param("id"), nil)
	commercialJSON(c, gin.H{"queued": err == nil}, err)
}

func (a *CommercialAdminController) notices(c *gin.Context) {
	rows, err := a.service.Notices()
	commercialJSON(c, rows, err)
}

func (a *CommercialAdminController) saveNotice(c *gin.Context) {
	var row model.Notice
	if err := c.ShouldBindJSON(&row); err != nil {
		commercialJSON(c, nil, errors.New("请求格式无效"))
		return
	}
	err := a.service.SaveNotice(&row)
	a.audit(c, "notice.save", "notice", row.ID, nil)
	commercialJSON(c, row, err)
}

func (a *CommercialAdminController) articles(c *gin.Context) {
	rows, err := a.service.Articles()
	commercialJSON(c, rows, err)
}

func (a *CommercialAdminController) saveArticle(c *gin.Context) {
	var row model.KnowledgeArticle
	if err := c.ShouldBindJSON(&row); err != nil {
		commercialJSON(c, nil, errors.New("请求格式无效"))
		return
	}
	err := a.service.SaveArticle(&row)
	a.audit(c, "article.save", "article", row.ID, nil)
	commercialJSON(c, row, err)
}

func (a *CommercialAdminController) applications(c *gin.Context) {
	rows, err := a.service.Applications()
	commercialJSON(c, rows, err)
}

func (a *CommercialAdminController) saveApplication(c *gin.Context) {
	var row model.ClientApplication
	if err := c.ShouldBindJSON(&row); err != nil {
		commercialJSON(c, nil, errors.New("请求格式无效"))
		return
	}
	err := a.service.SaveApplication(&row)
	a.audit(c, "application.save", "client_application", row.ID, nil)
	commercialJSON(c, row, err)
}

func (a *CommercialAdminController) uploadApplicationPackage(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, commercial.MaxClientPackageSize+(1<<20))
	header, err := c.FormFile("package")
	if err != nil {
		commercialJSON(c, nil, errors.New("请选择要上传的安装包"))
		return
	}
	file, err := header.Open()
	if err != nil {
		commercialJSON(c, nil, errors.New("读取安装包失败"))
		return
	}
	defer file.Close()

	row, err := a.service.SaveApplicationPackage(c.Param("id"), file, header)
	if err == nil {
		a.audit(c, "application.package.upload", "client_application", c.Param("id"), gin.H{"fileName": row.PackageFileName, "size": row.PackageSize})
	}
	commercialJSON(c, row, err)
}

func (a *CommercialAdminController) tickets(c *gin.Context) {
	rows, err := a.service.Tickets(c.Query("status"))
	commercialJSON(c, rows, err)
}

func (a *CommercialAdminController) ticketMessages(c *gin.Context) {
	rows, err := a.service.TicketMessages(c.Param("id"))
	commercialJSON(c, rows, err)
}

func (a *CommercialAdminController) replyTicket(c *gin.Context) {
	var request entity.CommercialTicketReplyRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		commercialJSON(c, nil, errors.New("请求格式无效"))
		return
	}
	user := session.GetLoginUser(c)
	err := a.service.ReplyTicket(c.Param("id"), user.Id, request.Body, request.Status)
	a.audit(c, "ticket.reply", "ticket", c.Param("id"), nil)
	commercialJSON(c, gin.H{"replied": err == nil}, err)
}

func (a *CommercialAdminController) coupons(c *gin.Context) {
	rows, err := a.service.Coupons()
	commercialJSON(c, rows, err)
}

func (a *CommercialAdminController) saveCoupon(c *gin.Context) {
	var row model.Coupon
	if err := c.ShouldBindJSON(&row); err != nil {
		commercialJSON(c, nil, errors.New("请求格式无效"))
		return
	}
	err := a.service.SaveCoupon(&row)
	a.audit(c, "coupon.save", "coupon", row.ID, nil)
	commercialJSON(c, row, err)
}

func (a *CommercialAdminController) giftCards(c *gin.Context) {
	rows, err := a.service.GiftCards()
	commercialJSON(c, rows, err)
}

func (a *CommercialAdminController) createGiftCards(c *gin.Context) {
	var request entity.CommercialGiftCardBatchRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		commercialJSON(c, nil, errors.New("请求格式无效"))
		return
	}
	var expiresAt *time.Time
	if request.ExpiresAt != "" {
		value, err := time.Parse(time.RFC3339, request.ExpiresAt)
		if err != nil {
			commercialJSON(c, nil, errors.New("到期时间格式无效"))
			return
		}
		expiresAt = &value
	}
	codes, err := a.service.CreateGiftCards(request.ValueFen, request.Count, expiresAt)
	a.audit(c, "gift_card.create_batch", "gift_card", "batch", gin.H{"count": request.Count, "valueFen": request.ValueFen})
	commercialJSON(c, gin.H{"codes": codes}, err)
}

func (a *CommercialAdminController) commissions(c *gin.Context) {
	rows, err := a.service.Commissions()
	commercialJSON(c, rows, err)
}

func (a *CommercialAdminController) settleCommission(c *gin.Context) {
	err := a.service.SettleCommission(c.Param("id"))
	a.audit(c, "commission.settle", "commission", c.Param("id"), nil)
	commercialJSON(c, gin.H{"settled": err == nil}, err)
}

func (a *CommercialAdminController) settings(c *gin.Context) {
	commercialJSON(c, a.service.Settings(), nil)
}

func (a *CommercialAdminController) savePaymentSettings(c *gin.Context) {
	var request entity.CommercialPaymentSettings
	if err := c.ShouldBindJSON(&request); err != nil {
		commercialJSON(c, nil, errors.New("支付设置格式无效"))
		return
	}
	err := a.service.SavePaymentSettings(request)
	if err == nil {
		a.audit(c, "payment_settings.update", "settings", "payment", gin.H{
			"epayEnabled": request.EpayEnabled, "alipayEnabled": request.AlipayEnabled, "codepayEnabled": request.CodepayEnabled,
		})
	}
	commercialJSON(c, gin.H{"saved": err == nil}, err)
}

func (a *CommercialAdminController) siteSettings(c *gin.Context) {
	settings, err := a.service.SiteSettings()
	commercialJSON(c, settings, err)
}

func (a *CommercialAdminController) saveSiteSettings(c *gin.Context) {
	var request entity.CommercialSiteSettings
	if err := c.ShouldBindJSON(&request); err != nil {
		commercialJSON(c, nil, errors.New("站点设置格式无效"))
		return
	}
	err := a.service.SaveSiteSettings(request)
	if err == nil {
		a.audit(c, "site_settings.update", "settings", "site", gin.H{"registrationClosed": request.RegistrationClosed, "trialPlanId": request.TrialPlanID})
	}
	commercialJSON(c, gin.H{"saved": err == nil}, err)
}

func (a *CommercialAdminController) saveSiteLogo(c *gin.Context) {
	var request entity.CommercialSiteLogoRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		commercialJSON(c, nil, errors.New("LOGO 设置格式无效"))
		return
	}
	err := a.service.SaveSiteLogo(request.LogoURL)
	if err == nil {
		a.audit(c, "site_logo.update", "settings", "site_logo", nil)
	}
	commercialJSON(c, gin.H{"saved": err == nil}, err)
}

func (a *CommercialAdminController) securitySettings(c *gin.Context) {
	commercialJSON(c, a.service.SecuritySettings(), nil)
}

func (a *CommercialAdminController) saveSecuritySettings(c *gin.Context) {
	var request entity.CommercialSecuritySettings
	if err := c.ShouldBindJSON(&request); err != nil {
		commercialJSON(c, nil, errors.New("安全设置格式无效"))
		return
	}
	err := a.service.SaveSecuritySettings(request)
	if err == nil {
		a.audit(c, "security_settings.update", "settings", "security", gin.H{
			"safeMode":                    request.SafeMode,
			"emailSuffixWhitelistEnabled": request.EmailSuffixWhitelistEnabled,
			"passwordAttemptLimitEnabled": request.PasswordAttemptLimitEnabled,
		})
	}
	commercialJSON(c, gin.H{"saved": err == nil}, err)
}

func (a *CommercialAdminController) subscriptionSettings(c *gin.Context) {
	commercialJSON(c, a.service.SubscriptionSettings(), nil)
}

func (a *CommercialAdminController) saveSubscriptionSettings(c *gin.Context) {
	var request entity.CommercialSubscriptionSettings
	if err := c.ShouldBindJSON(&request); err != nil {
		commercialJSON(c, nil, errors.New("订阅设置格式无效"))
		return
	}
	err := a.service.SaveSubscriptionSettings(request)
	if err == nil {
		a.audit(c, "subscription_settings.update", "settings", "subscription", gin.H{
			"allowUserChange":        request.AllowUserChange,
			"monthlyResetMode":       request.MonthlyResetMode,
			"offsetEnabled":          request.OffsetEnabled,
			"showSubscriptionInfo":   request.ShowSubscriptionInfo,
			"showProtocolInNodeName": request.ShowProtocolInNodeName,
		})
	}
	commercialJSON(c, gin.H{"saved": err == nil}, err)
}

func (a *CommercialAdminController) invitationSettings(c *gin.Context) {
	commercialJSON(c, a.service.InvitationSettings(), nil)
}

func (a *CommercialAdminController) saveInvitationSettings(c *gin.Context) {
	var request entity.CommercialInvitationSettings
	if err := c.ShouldBindJSON(&request); err != nil {
		commercialJSON(c, nil, errors.New("邀请与佣金设置格式无效"))
		return
	}
	err := a.service.SaveInvitationSettings(request)
	if err == nil {
		a.audit(c, "invitation_settings.update", "settings", "invitation", gin.H{
			"forcedInvitation":           request.ForcedInvitation,
			"commissionPercent":          request.CommissionPercent,
			"commissionFirstPaymentOnly": request.CommissionFirstPaymentOnly,
			"commissionAutoConfirm":      request.CommissionAutoConfirm,
			"multiLevelEnabled":          request.MultiLevelEnabled,
		})
	}
	commercialJSON(c, gin.H{"saved": err == nil}, err)
}

func (a *CommercialAdminController) emailTemplates(c *gin.Context) {
	rows, err := a.service.EmailTemplates()
	commercialJSON(c, rows, err)
}

func (a *CommercialAdminController) saveEmailTemplate(c *gin.Context) {
	var request entity.CommercialEmailTemplateRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		commercialJSON(c, nil, errors.New("请求格式无效"))
		return
	}
	row, err := a.service.SaveEmailTemplate(c.Param("key"), request)
	if err == nil {
		a.audit(c, "email_template.update", "email_template", c.Param("key"), gin.H{"active": request.Active})
	}
	commercialJSON(c, row, err)
}

func (a *CommercialAdminController) sendCustomerEmail(c *gin.Context) {
	var request entity.CommercialEmailSendRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		commercialJSON(c, nil, errors.New("请求格式无效"))
		return
	}
	result, err := a.service.QueueCustomerEmail(request)
	if err == nil {
		a.audit(c, "customer_email.queue", "email_campaign", result.CampaignID, gin.H{
			"audience": request.Audience, "templateKey": request.TemplateKey, "queued": result.Queued,
		})
	}
	commercialJSON(c, result, err)
}

func (a *CommercialAdminController) saveSetting(c *gin.Context) {
	var request entity.CommercialSettingRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		commercialJSON(c, nil, errors.New("请求格式无效"))
		return
	}
	err := a.service.SetSetting(request)
	a.audit(c, "setting.update", "setting", request.Key, gin.H{"encrypted": request.Encrypted})
	commercialJSON(c, gin.H{"saved": err == nil}, err)
}

func (a *CommercialAdminController) auditLogs(c *gin.Context) {
	rows, err := a.service.AuditLogs(queryInt(c, "limit"))
	commercialJSON(c, rows, err)
}

func (a *CommercialAdminController) currentRole(c *gin.Context) {
	user := session.GetLoginUser(c)
	commercialJSON(c, gin.H{"role": a.service.Role(user.Id)}, nil)
}

func (a *CommercialAdminController) adminUsers(c *gin.Context) {
	rows, err := a.service.AdminUsers()
	commercialJSON(c, rows, err)
}

func (a *CommercialAdminController) setRole(c *gin.Context) {
	currentUser := session.GetLoginUser(c)
	if currentUser == nil || a.service.Role(currentUser.Id) != "owner" {
		commercialJSON(c, nil, errors.New("仅所有者可以调整管理员角色"))
		return
	}
	var request entity.CommercialRoleRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		commercialJSON(c, nil, errors.New("请求格式无效"))
		return
	}
	userID, err := strconv.Atoi(c.Param("userId"))
	if err != nil {
		commercialJSON(c, nil, errors.New("管理员 ID 无效"))
		return
	}
	err = a.service.SetRole(userID, request.Role)
	a.audit(c, "role.update", "admin", c.Param("userId"), gin.H{"role": request.Role})
	commercialJSON(c, gin.H{"saved": err == nil}, err)
}

func (a *CommercialAdminController) audit(c *gin.Context, action, targetType, targetID string, metadata any) {
	user := session.GetLoginUser(c)
	if user == nil {
		return
	}
	data := ""
	if metadata != nil {
		encoded, _ := json.Marshal(metadata)
		data = string(encoded)
	}
	role, _ := c.Get("commercial_admin_role")
	ipDigest := sha256.Sum256([]byte(getRemoteIp(c)))
	a.service.Audit(user.Id, strings.TrimSpace(toString(role)), action, targetType, targetID, data, hex.EncodeToString(ipDigest[:]))
}

func queryInt(c *gin.Context, key string) int {
	value, _ := strconv.Atoi(c.Query(key))
	return value
}

func toString(value any) string {
	text, _ := value.(string)
	return text
}

func roleAllows(role, permission string) bool {
	if role == "owner" || role == "administrator" {
		return true
	}
	if permission == "read" {
		return true
	}
	permissions := map[string]map[string]bool{
		"finance":           {"customers.read": true, "plans.read": true, "orders.read": true, "orders.manage": true, "finance.read": true, "finance.manage": true, "marketing.read": true, "marketing.manage": true, "audit.read": true},
		"support":           {"customers.read": true, "customers.manage": true, "plans.read": true, "orders.read": true, "content.read": true, "tickets.read": true, "tickets.manage": true},
		"node_operator":     {"customers.read": true, "plans.read": true, "orders.read": true, "lines.read": true, "lines.manage": true, "audit.read": true},
		"read_only_auditor": {"customers.read": true, "plans.read": true, "orders.read": true, "lines.read": true, "content.read": true, "tickets.read": true, "marketing.read": true, "finance.read": true, "settings.read": true, "audit.read": true},
	}
	return permissions[role][permission]
}
