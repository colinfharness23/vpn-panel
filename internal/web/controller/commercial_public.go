package controller

import (
	"encoding/base64"
	"errors"
	"mime"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/mhsanaei/3x-ui/v3/internal/sub"
	"github.com/mhsanaei/3x-ui/v3/internal/web/entity"
	"github.com/mhsanaei/3x-ui/v3/internal/web/middleware"
	"github.com/mhsanaei/3x-ui/v3/internal/web/service/commercial"
	"github.com/mhsanaei/3x-ui/v3/internal/web/session"

	"github.com/gin-gonic/gin"
	"github.com/skip2/go-qrcode"
	"gorm.io/gorm"
)

const customerSessionCookie = "nova-session"

type CommercialPublicController struct {
	BaseController
	auth   *commercial.AuthService
	orders *commercial.OrderService
	portal *commercial.PortalService
	config *commercial.ConfigStore
	relays *commercial.ResidentialRelayService
}

func NewCommercialPublicController(g *gin.RouterGroup, mailer commercial.VerificationMailer) *CommercialPublicController {
	controller := &CommercialPublicController{auth: commercial.NewAuthService(mailer), orders: commercial.NewOrderService(), portal: commercial.NewPortalService(), config: commercial.NewConfigStore(), relays: commercial.NewResidentialRelayService()}
	controller.initRouter(g)
	return controller
}

func (a *CommercialPublicController) initRouter(g *gin.RouterGroup) {
	g.GET("/", a.siteHostGuard, a.portalSPA)
	for _, route := range []string{"/subscription", "/plans", "/guides", "/tickets", "/orders", "/account"} {
		g.GET(route, a.siteHostGuard, a.portalSPA)
	}
	g.GET("/portal", a.siteHostGuard, func(c *gin.Context) {
		c.Redirect(http.StatusPermanentRedirect, strings.TrimRight(c.GetString("base_path"), "/")+"/")
	})
	g.GET("/portal/*path", a.siteHostGuard, func(c *gin.Context) {
		c.Redirect(http.StatusPermanentRedirect, strings.TrimRight(c.GetString("base_path"), "/")+"/")
	})

	guest := g.Group("/api/v1/guest")
	guest.Use(a.siteHostGuard)
	guest.GET("/auth-config", a.authConfig)
	guest.POST("/payments/alipay/notify", a.alipayNotify)
	guest.GET("/payments/epay/notify", a.epayNotify)
	guest.POST("/payments/epay/notify", a.epayNotify)
	guest.GET("/payments/codepay/notify", a.codepayNotify)
	guest.POST("/payments/codepay/notify", a.codepayNotify)

	passport := g.Group("/api/v1/passport")
	passport.Use(a.siteHostGuard)
	passport.Use(middleware.CSRFMiddleware())
	passport.GET("/csrf-token", a.csrfToken)
	passport.POST("/send-code", a.sendCode)
	passport.POST("/register", a.register)
	passport.POST("/login", a.loginCustomer)
	passport.POST("/reset-password", a.resetPassword)
	passport.POST("/logout", a.customerAuth, a.logoutCustomer)

	user := g.Group("/api/v1/user")
	user.Use(a.siteHostGuard)
	user.Use(a.customerAuth)
	user.Use(middleware.CSRFMiddleware())
	user.GET("/bootstrap", a.bootstrap)
	user.GET("/applications/:id/download", a.downloadApplication)
	user.GET("/dashboard", a.dashboard)
	user.GET("/orders", a.ordersList)
	user.POST("/orders", a.createOrder)
	user.POST("/orders/:id/pay", a.precreatePayment)
	user.POST("/orders/:id/cancel", a.cancelOrder)
	user.POST("/orders/:id/demo-pay", a.demoPay)
	user.POST("/subscription/rotate", a.rotateSubscription)
	user.GET("/subscription/qr", a.subscriptionQR)
	user.GET("/residential-relays", a.residentialRelays)
	user.POST("/residential-relays", a.createResidentialRelay)
	user.PUT("/residential-relays/:id", a.updateResidentialRelay)
	user.DELETE("/residential-relays/:id", a.deleteResidentialRelay)
	user.GET("/residential-relays/:id/qr", a.residentialRelayQR)
	user.GET("/sessions", a.sessions)
	user.DELETE("/sessions/:id", a.revokeSession)
	user.POST("/account/password", a.changePassword)
	user.GET("/tickets", a.tickets)
	user.POST("/tickets", a.createTicket)
	user.GET("/tickets/:id/messages", a.ticketMessages)
	user.POST("/tickets/:id/reply", a.replyTicket)
	user.POST("/gift-cards/redeem", a.redeemGiftCard)
}

func (a *CommercialPublicController) siteHostGuard(c *gin.Context) {
	if !a.config.SecurityPolicy().SafeMode {
		c.Next()
		return
	}
	parsed, err := url.Parse(strings.TrimSpace(a.config.GetDefault("site.url", "")))
	allowedHost := strings.ToLower(strings.TrimSuffix(parsed.Hostname(), "."))
	requestHost := strings.TrimSpace(c.Request.Host)
	if host, _, splitErr := net.SplitHostPort(requestHost); splitErr == nil {
		requestHost = host
	}
	requestHost = strings.ToLower(strings.TrimSuffix(strings.Trim(requestHost, "[]"), "."))
	if err != nil || allowedHost == "" || requestHost != allowedHost {
		c.AbortWithStatusJSON(http.StatusForbidden, entity.Msg{Success: false, Msg: "当前域名未被站点安全模式允许"})
		return
	}
	c.Next()
}

func (a *CommercialPublicController) portalSPA(c *gin.Context) {
	serveDistPage(c, "portal.html")
}

func (a *CommercialPublicController) csrfToken(c *gin.Context) {
	token, err := session.EnsureCSRFToken(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, entity.Msg{Success: false, Msg: err.Error()})
		return
	}
	c.JSON(http.StatusOK, entity.Msg{Success: true, Obj: token})
}

func (a *CommercialPublicController) bootstrap(c *gin.Context) {
	locale := requestLocale(c)
	data, err := a.portal.Bootstrap(locale)
	commercialJSON(c, data, err)
}

func (a *CommercialPublicController) authConfig(c *gin.Context) {
	commercialJSON(c, a.portal.AuthConfig(), nil)
}

func (a *CommercialPublicController) downloadApplication(c *gin.Context) {
	row, file, info, err := a.portal.OpenApplicationPackage(c.Param("id"))
	if err != nil {
		c.AbortWithStatusJSON(http.StatusNotFound, entity.Msg{Success: false, Msg: err.Error()})
		return
	}
	defer file.Close()

	disposition := mime.FormatMediaType("attachment", map[string]string{"filename": row.PackageFileName})
	if disposition != "" {
		c.Header("Content-Disposition", disposition)
	}
	c.Header("Content-Type", "application/octet-stream")
	c.Header("X-Content-Type-Options", "nosniff")
	c.Header("Cache-Control", "private, max-age=0, must-revalidate")
	if row.PackageSHA256 != "" {
		c.Header("ETag", `"`+row.PackageSHA256+`"`)
	}
	http.ServeContent(c.Writer, c.Request, row.PackageFileName, info.ModTime(), file)
}

func (a *CommercialPublicController) sendCode(c *gin.Context) {
	var request entity.SendVerificationCodeRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		commercialJSON(c, nil, errors.New("请求格式无效"))
		return
	}
	debugCode, err := a.auth.SendCode(c.Request.Context(), request.Email, request.Purpose, getRemoteIp(c), request.TurnstileToken)
	obj := gin.H{"sent": err == nil}
	if debugCode != "" {
		obj["debugCode"] = debugCode
	}
	commercialJSON(c, obj, err)
}

func (a *CommercialPublicController) register(c *gin.Context) {
	var request entity.CustomerRegisterRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		commercialJSON(c, nil, errors.New("请求格式无效"))
		return
	}
	customer, err := a.auth.Register(c.Request.Context(), request.Email, request.Password, request.Code, request.InviteCode, request.Locale, getRemoteIp(c), request.TurnstileToken, request.AcceptedTerms, request.TermsVersion)
	if err != nil {
		commercialJSON(c, nil, err)
		return
	}
	if err := a.establishCustomerSession(c, customer.ID); err != nil {
		commercialJSON(c, nil, err)
		return
	}
	commercialJSON(c, customer, nil)
}

func (a *CommercialPublicController) loginCustomer(c *gin.Context) {
	var request entity.CustomerLoginRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		commercialJSON(c, nil, errors.New("请求格式无效"))
		return
	}
	customer, err := a.auth.Login(request.Email, request.Password)
	if err != nil {
		commercialJSON(c, nil, err)
		return
	}
	if err := a.establishCustomerSession(c, customer.ID); err != nil {
		commercialJSON(c, nil, err)
		return
	}
	commercialJSON(c, customer, nil)
}

func (a *CommercialPublicController) resetPassword(c *gin.Context) {
	var request entity.CustomerResetPasswordRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		commercialJSON(c, nil, errors.New("请求格式无效"))
		return
	}
	commercialJSON(c, gin.H{"reset": true}, a.auth.ResetPassword(request.Email, request.Code, request.Password))
}

func (a *CommercialPublicController) logoutCustomer(c *gin.Context) {
	identity := customerIdentity(c)
	if identity != nil {
		_ = a.auth.RevokeSession(identity.Customer.ID, identity.Session.ID)
	}
	a.clearCustomerCookie(c)
	commercialJSON(c, gin.H{"loggedOut": true}, nil)
}

func (a *CommercialPublicController) dashboard(c *gin.Context) {
	identity := customerIdentity(c)
	data, err := a.portal.Dashboard(&identity.Customer, requestOrigin(c))
	commercialJSON(c, data, err)
}

func (a *CommercialPublicController) ordersList(c *gin.Context) {
	identity := customerIdentity(c)
	orders, err := a.orders.List(identity.Customer.ID)
	commercialJSON(c, orders, err)
}

func (a *CommercialPublicController) createOrder(c *gin.Context) {
	var request entity.CreateCommercialOrderRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		commercialJSON(c, nil, errors.New("请求格式无效"))
		return
	}
	identity := customerIdentity(c)
	order, err := a.orders.CreateFor(identity.Customer.ID, request.PlanPriceID, request.OrderKind, request.EntitlementID, request.CouponCode, request.UseBalance)
	commercialJSON(c, order, err)
}

func (a *CommercialPublicController) precreatePayment(c *gin.Context) {
	var request entity.CreatePaymentRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		commercialJSON(c, nil, errors.New("请选择支付方式"))
		return
	}
	identity := customerIdentity(c)
	intent, err := a.orders.Precreate(c.Request.Context(), identity.Customer.ID, c.Param("id"), request.Provider)
	if err != nil {
		commercialJSON(c, nil, err)
		return
	}
	png, err := qrcode.Encode(intent.QRCode, qrcode.Medium, 320)
	if err != nil {
		commercialJSON(c, nil, err)
		return
	}
	commercialJSON(c, gin.H{"intent": intent, "qrImage": "data:image/png;base64," + base64.StdEncoding.EncodeToString(png)}, nil)
}

func (a *CommercialPublicController) cancelOrder(c *gin.Context) {
	identity := customerIdentity(c)
	err := a.orders.Cancel(identity.Customer.ID, c.Param("id"))
	commercialJSON(c, gin.H{"cancelled": err == nil}, err)
}

func (a *CommercialPublicController) demoPay(c *gin.Context) {
	identity := customerIdentity(c)
	err := a.orders.DemoPay(c.Request.Context(), identity.Customer.ID, c.Param("id"))
	commercialJSON(c, gin.H{"paid": err == nil}, err)
}

func (a *CommercialPublicController) rotateSubscription(c *gin.Context) {
	identity := customerIdentity(c)
	links, err := a.portal.RotateSubscription(identity.Customer.ID, requestOrigin(c))
	commercialJSON(c, links, err)
}

func (a *CommercialPublicController) subscriptionQR(c *gin.Context) {
	identity := customerIdentity(c)
	dashboard, err := a.portal.Dashboard(&identity.Customer, requestOrigin(c))
	if err != nil || dashboard.Subscription == nil {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}
	format := c.DefaultQuery("format", "raw")
	value := dashboard.Subscription.Links.Raw
	switch format {
	case "clash":
		value = dashboard.Subscription.Links.Clash
	case "json":
		value = dashboard.Subscription.Links.JSON
	}
	png, err := qrcode.Encode(value, qrcode.Medium, 320)
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.Data(http.StatusOK, "image/png", png)
}

func (a *CommercialPublicController) residentialRelays(c *gin.Context) {
	identity := customerIdentity(c)
	overview, err := a.relays.Overview(identity.Customer.ID)
	if err == nil {
		a.attachResidentialRelayLinks(c, overview)
	}
	commercialJSON(c, overview, err)
}

func (a *CommercialPublicController) createResidentialRelay(c *gin.Context) {
	var request entity.ResidentialRelayRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		commercialJSON(c, nil, errors.New("请检查住宅中转的填写内容"))
		return
	}
	identity := customerIdentity(c)
	overview, err := a.relays.Create(c.Request.Context(), identity.Customer.ID, request)
	if err == nil {
		a.attachResidentialRelayLinks(c, overview)
	}
	commercialJSON(c, overview, err)
}

func (a *CommercialPublicController) updateResidentialRelay(c *gin.Context) {
	var request entity.ResidentialRelayRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		commercialJSON(c, nil, errors.New("请检查住宅中转的填写内容"))
		return
	}
	identity := customerIdentity(c)
	overview, err := a.relays.Update(c.Request.Context(), identity.Customer.ID, c.Param("id"), request)
	if err == nil {
		a.attachResidentialRelayLinks(c, overview)
	}
	commercialJSON(c, overview, err)
}

func (a *CommercialPublicController) deleteResidentialRelay(c *gin.Context) {
	identity := customerIdentity(c)
	overview, err := a.relays.Delete(identity.Customer.ID, c.Param("id"))
	if err == nil {
		a.attachResidentialRelayLinks(c, overview)
	}
	commercialJSON(c, overview, err)
}

func (a *CommercialPublicController) residentialRelayQR(c *gin.Context) {
	identity := customerIdentity(c)
	relay, err := a.relays.Relay(identity.Customer.ID, c.Param("id"))
	if err != nil {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}
	links := sub.NewLinkProvider().LinksForClient(requestOrigin(c), relay.Inbound, relay.ClientID)
	if len(links) == 0 {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}
	png, err := qrcode.Encode(links[0], qrcode.Medium, 360)
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.Data(http.StatusOK, "image/png", png)
}

func (a *CommercialPublicController) attachResidentialRelayLinks(c *gin.Context, overview *commercial.ResidentialRelayOverview) {
	if overview == nil {
		return
	}
	provider := sub.NewLinkProvider()
	for i := range overview.Relays {
		overview.Relays[i].Links = provider.LinksForClient(requestOrigin(c), overview.Relays[i].Inbound, overview.Relays[i].ClientID)
	}
}

func (a *CommercialPublicController) sessions(c *gin.Context) {
	identity := customerIdentity(c)
	rows, err := a.auth.Sessions(identity.Customer.ID)
	commercialJSON(c, gin.H{"currentSessionId": identity.Session.ID, "sessions": rows}, err)
}

func (a *CommercialPublicController) revokeSession(c *gin.Context) {
	identity := customerIdentity(c)
	err := a.auth.RevokeSession(identity.Customer.ID, c.Param("id"))
	if identity.Session.ID == c.Param("id") {
		a.clearCustomerCookie(c)
	}
	commercialJSON(c, gin.H{"revoked": err == nil}, err)
}

func (a *CommercialPublicController) changePassword(c *gin.Context) {
	var request entity.CustomerChangePasswordRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		commercialJSON(c, nil, errors.New("请求格式无效"))
		return
	}
	identity := customerIdentity(c)
	token, _, err := a.auth.ChangePassword(identity.Customer.ID, request.CurrentPassword, request.NewPassword, getRemoteIp(c), c.GetHeader("User-Agent"))
	if err == nil {
		a.setCustomerCookie(c, token)
	}
	commercialJSON(c, gin.H{"changed": err == nil}, err)
}

func (a *CommercialPublicController) tickets(c *gin.Context) {
	identity := customerIdentity(c)
	rows, err := a.portal.Tickets(identity.Customer.ID)
	commercialJSON(c, rows, err)
}

func (a *CommercialPublicController) createTicket(c *gin.Context) {
	var request entity.CreateTicketRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		commercialJSON(c, nil, errors.New("请求格式无效"))
		return
	}
	identity := customerIdentity(c)
	ticket, err := a.portal.CreateTicket(identity.Customer.ID, request.EntitlementID, request.Subject, request.Body)
	commercialJSON(c, ticket, err)
}

func (a *CommercialPublicController) ticketMessages(c *gin.Context) {
	identity := customerIdentity(c)
	rows, err := a.portal.TicketMessages(identity.Customer.ID, c.Param("id"))
	commercialJSON(c, rows, err)
}

func (a *CommercialPublicController) replyTicket(c *gin.Context) {
	var request entity.ReplyTicketRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		commercialJSON(c, nil, errors.New("请求格式无效"))
		return
	}
	identity := customerIdentity(c)
	err := a.portal.ReplyTicket(identity.Customer.ID, c.Param("id"), request.Body)
	commercialJSON(c, gin.H{"replied": err == nil}, err)
}

func (a *CommercialPublicController) redeemGiftCard(c *gin.Context) {
	var request entity.RedeemGiftCardRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		commercialJSON(c, nil, errors.New("请求格式无效"))
		return
	}
	identity := customerIdentity(c)
	value, err := a.portal.RedeemGiftCard(identity.Customer.ID, request.Code)
	commercialJSON(c, gin.H{"valueFen": value}, err)
}

func (a *CommercialPublicController) alipayNotify(c *gin.Context) {
	if err := c.Request.ParseForm(); err != nil {
		c.String(http.StatusOK, "failure")
		return
	}
	if err := a.orders.HandleNotificationForProvider(c.Request.Context(), "alipay", c.Request.PostForm); err != nil {
		c.String(http.StatusOK, "failure")
		return
	}
	c.String(http.StatusOK, "success")
}

func (a *CommercialPublicController) epayNotify(c *gin.Context) {
	if err := c.Request.ParseForm(); err != nil {
		c.String(http.StatusOK, "fail")
		return
	}
	if err := a.orders.HandleNotificationForProvider(c.Request.Context(), "epay", c.Request.Form); err != nil {
		c.String(http.StatusOK, "fail")
		return
	}
	c.String(http.StatusOK, "success")
}

func (a *CommercialPublicController) codepayNotify(c *gin.Context) {
	if err := c.Request.ParseForm(); err != nil {
		c.String(http.StatusOK, "fail")
		return
	}
	if err := a.orders.HandleNotificationForProvider(c.Request.Context(), "codepay", c.Request.Form); err != nil {
		c.String(http.StatusOK, "fail")
		return
	}
	c.String(http.StatusOK, "success")
}

func (a *CommercialPublicController) customerAuth(c *gin.Context) {
	token, err := c.Cookie(customerSessionCookie)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, entity.Msg{Success: false, Msg: "请先登录"})
		return
	}
	identity, err := a.auth.Authenticate(token)
	if err != nil {
		a.clearCustomerCookie(c)
		c.AbortWithStatusJSON(http.StatusUnauthorized, entity.Msg{Success: false, Msg: "登录已失效"})
		return
	}
	c.Set("commercial_customer", identity)
	c.Next()
}

func (a *CommercialPublicController) establishCustomerSession(c *gin.Context, customerID string) error {
	token, _, err := a.auth.CreateSession(customerID, getRemoteIp(c), c.GetHeader("User-Agent"))
	if err != nil {
		return err
	}
	a.setCustomerCookie(c, token)
	return nil
}

func (a *CommercialPublicController) setCustomerCookie(c *gin.Context, token string) {
	basePath := c.GetString("base_path")
	if basePath == "" {
		basePath = "/"
	}
	http.SetCookie(c.Writer, &http.Cookie{Name: customerSessionCookie, Value: token, Path: basePath, MaxAge: 30 * 24 * 60 * 60, Expires: time.Now().UTC().Add(30 * 24 * time.Hour), HttpOnly: true, Secure: commercialCookieSecure(c), SameSite: http.SameSiteLaxMode})
}

func (a *CommercialPublicController) clearCustomerCookie(c *gin.Context) {
	basePath := c.GetString("base_path")
	if basePath == "" {
		basePath = "/"
	}
	http.SetCookie(c.Writer, &http.Cookie{Name: customerSessionCookie, Value: "", Path: basePath, MaxAge: -1, Expires: time.Unix(0, 0), HttpOnly: true, Secure: commercialCookieSecure(c), SameSite: http.SameSiteLaxMode})
}

func commercialCookieSecure(c *gin.Context) bool {
	if c.Request.TLS != nil {
		return true
	}
	if !isTrustedForwardedRequest(c) {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(strings.Split(c.GetHeader("X-Forwarded-Proto"), ",")[0]), "https")
}

func customerIdentity(c *gin.Context) *commercial.SessionIdentity {
	value, _ := c.Get("commercial_customer")
	identity, _ := value.(*commercial.SessionIdentity)
	return identity
}

func requestLocale(c *gin.Context) string {
	if locale := strings.TrimSpace(c.Query("locale")); locale != "" {
		return locale
	}
	value := c.GetHeader("Accept-Language")
	if comma := strings.Index(value, ","); comma >= 0 {
		value = value[:comma]
	}
	if semicolon := strings.Index(value, ";"); semicolon >= 0 {
		value = value[:semicolon]
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return "zh-CN"
	}
	return value
}

func requestOrigin(c *gin.Context) string {
	scheme := "http"
	host := c.Request.Host
	if c.Request.TLS != nil {
		scheme = "https"
	}
	if isTrustedForwardedRequest(c) {
		if forwarded := strings.TrimSpace(strings.Split(c.GetHeader("X-Forwarded-Proto"), ",")[0]); forwarded == "http" || forwarded == "https" {
			scheme = forwarded
		}
		if forwarded := strings.TrimSpace(strings.Split(c.GetHeader("X-Forwarded-Host"), ",")[0]); forwarded != "" {
			host = forwarded
		}
	}
	return scheme + "://" + host
}

func commercialJSON(c *gin.Context, obj any, err error) {
	status := http.StatusOK
	message := ""
	if err != nil {
		message = err.Error()
		status = http.StatusBadRequest
		if errors.Is(err, gorm.ErrRecordNotFound) {
			status = http.StatusNotFound
		}
	}
	c.JSON(status, entity.Msg{Success: err == nil, Msg: message, Obj: obj})
}
