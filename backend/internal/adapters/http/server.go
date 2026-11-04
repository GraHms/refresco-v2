package http

import (
	"encoding/csv"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/refresco/refresco-v2/backend/internal/app"
	d "github.com/refresco/refresco-v2/backend/internal/domain"
	"github.com/refresco/refresco-v2/backend/internal/ports"
)

type Options struct {
	PublicURL      string
	AllowedOrigins []string
	TrustedProxies []string
	CookieSecure   bool
	Production     bool
	Ping           func() error
}

type Server struct {
	cards cardCache
	app   *app.App
	log   *slog.Logger
	opts  Options
}

func asMaxBytes(err error, t **http.MaxBytesError) bool { return errors.As(err, t) }

func New(a *app.App, l *slog.Logger, o Options) (*gin.Engine, error) {
	if o.Production {
		gin.SetMode(gin.ReleaseMode)
	}
	s := &Server{app: a, log: l, opts: o}
	r := gin.New()
	r.HandleMethodNotAllowed = true
	if err := r.SetTrustedProxies(o.TrustedProxies); err != nil {
		return nil, err
	}
	allowed := map[string]bool{}
	for _, x := range o.AllowedOrigins {
		allowed[x] = true
	}
	if u, err := url.Parse(o.PublicURL); err == nil && u.Scheme != "" && u.Host != "" {
		allowed[u.Scheme+"://"+u.Host] = true
	}
	r.Use(requestID(), recovery(l), accessLog(l), securityHeaders(), bodyLimit(64<<10), cors(allowed))

	r.GET("/healthz", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) })
	r.GET("/_meta/*path", s.metaHead)
	r.GET("/_body/*path", s.metaBody)
	r.GET("/robots.txt", s.robots)
	r.GET("/sitemap.xml", s.sitemap)
	r.GET("/og/*rest", newLimiter(600, 100).mw("og"), s.ogImage)
	r.GET("/readyz", func(c *gin.Context) {
		if o.Ping != nil && o.Ping() != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"ok": false})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	authLimit := newLimiter(20, 10)
	payLimit := newLimiter(30, 10)
	pollLimit := newLimiter(240, 40)
	apiLimit := newLimiter(600, 120)

	r.POST("/api/v1/webhooks/payments", newLimiter(600, 100).mw("webhook"), s.webhook)

	v1 := r.Group("/api/v1", csrfGuard(allowed), apiLimit.mw("api"))
	{
		au := v1.Group("/auth", authLimit.mw("auth"))
		au.POST("/login", s.login)
		au.POST("/logout", s.logout)
		au.POST("/register", s.register)
		au.POST("/invites/check", s.checkInvite)
		au.POST("/waitlist", s.waitlist)
		au.POST("/password/forgot", s.forgot)
		au.POST("/password/verify", s.verifyCode)
		au.POST("/password/reset", s.resetPassword)
		v1.GET("/handles/:handle", s.handleAvailable)

		v1.GET("/pages/:handle", s.publicPage)
		v1.POST("/pages/:handle/supports", payLimit.mw("pay"), s.createSupport)
		v1.GET("/supports/:id", pollLimit.mw("poll"), s.supportStatus)
		v1.POST("/supports/:id/otp", payLimit.mw("otp"), s.submitOTP)
		v1.POST("/supports/:id/resend", payLimit.mw("pay"), s.resend)
		v1.POST("/supports/:id/cancel", s.cancel)
		v1.POST("/supports/:id/simulate", payLimit.mw("pay"), s.simulate)

		v1.POST("/split-approvals/lookup", authLimit.mw("approval"), s.splitApproval)
		v1.POST("/split-approvals/approve", authLimit.mw("approval"), s.approveSplit)

		me := v1.Group("", s.auth)
		me.GET("/me", s.me)

		bo := me.Group("/backoffice", s.creator)
		bo.GET("/overview", s.overview)
		bo.GET("/supports", s.listSupports)
		bo.GET("/supports.csv", s.exportSupports)
		bo.GET("/supports/:id", s.getSupport)
		bo.POST("/supports/:id/unread", s.markUnread)
		bo.GET("/page", s.getPage)
		bo.PUT("/page", s.updatePage)
		bo.GET("/group", s.group)
		bo.PUT("/group/auto-split", s.autoSplit)
		bo.POST("/group/members", s.inviteMember)
		bo.DELETE("/group/members/:id", s.removeMember)
		bo.POST("/group/proposals", s.propose)
		bo.DELETE("/group/proposals/current", s.cancelProposal)
		bo.GET("/wallet", s.wallet)
		bo.POST("/wallet/accounts", s.addPayoutAccount)
		bo.DELETE("/wallet/accounts/:id", s.deletePayoutAccount)
		bo.POST("/wallet/withdrawals", s.requestWithdrawal)

		ad := me.Group("/admin", s.superAdmin)
		ad.GET("/dashboard", s.adminDashboard)
		ad.GET("/accounts", s.adminAccounts)
		ad.GET("/accounts/:id", s.adminAccount)
		ad.PUT("/accounts/:id/fee", s.adminFee)
		ad.PUT("/accounts/:id/status", s.adminStatus)
		ad.GET("/withdrawals", s.adminWithdrawals)
		ad.GET("/withdrawals.pdf", s.adminWithdrawalsPDF)
		ad.GET("/withdrawals/:id", s.adminWithdrawal)
		ad.GET("/withdrawals/:id/pdf", s.adminWithdrawalPDF)
		ad.POST("/withdrawals/:id/actions", s.adminWithdrawalAction)
		ad.GET("/invites", s.adminInvites)
		ad.POST("/invites", s.adminCreateInvite)
		ad.POST("/invites/:code/revoke", s.adminRevokeInvite)
	}
	r.NoRoute(func(c *gin.Context) { abort(c, http.StatusNotFound, "not_found", "Não encontrado.") })
	return r, nil
}

func (s *Server) bind(c *gin.Context, v any) bool {
	if err := c.ShouldBindJSON(v); err != nil {
		var mbe *http.MaxBytesError
		if errors.As(err, &mbe) {
			abort(c, http.StatusRequestEntityTooLarge, "too_large", "Pedido demasiado grande.")
		} else {
			abort(c, http.StatusBadRequest, "bad_request", "Pedido inválido.")
		}
		return false
	}
	return true
}

func (s *Server) setCookie(c *gin.Context, tok string, exp time.Time) {
	http.SetCookie(c.Writer, &http.Cookie{Name: cookieName, Value: tok, Path: "/api", Expires: exp, MaxAge: int(time.Until(exp).Seconds()),
		HttpOnly: true, Secure: s.opts.CookieSecure, SameSite: http.SameSiteStrictMode})
}

func (s *Server) clearCookie(c *gin.Context) {
	http.SetCookie(c.Writer, &http.Cookie{Name: cookieName, Value: "", Path: "/api", MaxAge: -1, HttpOnly: true, Secure: s.opts.CookieSecure, SameSite: http.SameSiteStrictMode})
}

type meView struct {
	ID         string          `json:"id"`
	Name       string          `json:"name"`
	Email      string          `json:"email"`
	Role       d.Role          `json:"role"`
	Account    *pageView       `json:"account,omitempty"`
	Member     *app.MemberView `json:"member,omitempty"`
	Status     d.AccountStatus `json:"account_status,omitempty"`
	Sandbox    bool            `json:"sandbox"`
	ComingSoon []d.Channel     `json:"coming_soon"`
}

func (s *Server) meOf(c *gin.Context, u *d.User) meView {
	v := meView{ID: u.ID, Name: u.Name, Email: u.Email, Role: u.Role, Sandbox: s.app.Sandbox, ComingSoon: s.comingSoon()}
	if u.Role == d.RoleCreator {
		if cr, err := s.app.CreatorContext(c.Request.Context(), u); err == nil {
			p := toPage(cr.Account)
			mv := app.MemberView{ID: cr.Member.ID, Name: cr.Member.Name, Role: cr.Member.Role, IsAdmin: cr.Member.IsAdmin, Pct: cr.Member.Pct}
			v.Account, v.Member, v.Status = &p, &mv, cr.Account.Status
		}
	}
	return v
}

func (s *Server) login(c *gin.Context) {
	var in struct {
		Identifier string `json:"identifier"`
		Password   string `json:"password"`
	}
	if !s.bind(c, &in) {
		return
	}
	sess, err := s.app.Login(c.Request.Context(), in.Identifier, in.Password)
	if err != nil {
		s.fail(c, err)
		return
	}
	s.setCookie(c, sess.Token, sess.ExpiresAt)
	c.JSON(http.StatusOK, s.meOf(c, sess.User))
}

func (s *Server) logout(c *gin.Context) {
	if tok, err := c.Cookie(cookieName); err == nil && tok != "" {
		s.app.Logout(c.Request.Context(), tok)
	}
	s.clearCookie(c)
	c.Status(http.StatusNoContent)
}

func (s *Server) me(c *gin.Context) { c.JSON(http.StatusOK, s.meOf(c, user(c))) }

func (s *Server) register(c *gin.Context) {
	var in struct {
		Code     string `json:"code"`
		Name     string `json:"name"`
		Handle   string `json:"handle"`
		Email    string `json:"email"`
		Phone    string `json:"phone"`
		Password string `json:"password"`
		Terms    bool   `json:"terms"`
	}
	if !s.bind(c, &in) {
		return
	}
	sess, err := s.app.Register(c.Request.Context(), app.RegisterInput(in))
	if err != nil {
		s.fail(c, err)
		return
	}
	s.setCookie(c, sess.Token, sess.ExpiresAt)
	c.JSON(http.StatusCreated, s.meOf(c, sess.User))
}

func (s *Server) checkInvite(c *gin.Context) {
	var in struct {
		Code string `json:"code"`
	}
	if !s.bind(c, &in) {
		return
	}
	r, err := s.app.CheckInvite(c.Request.Context(), in.Code)
	if err != nil {
		s.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, r)
}

func (s *Server) waitlist(c *gin.Context) {
	var in struct {
		Email string `json:"email"`
	}
	if !s.bind(c, &in) {
		return
	}
	pos, err := s.app.JoinWaitlist(c.Request.Context(), in.Email)
	if err != nil {
		s.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"position": pos})
}

func (s *Server) forgot(c *gin.Context) {
	var in struct {
		Identifier string `json:"identifier"`
	}
	if !s.bind(c, &in) {
		return
	}
	if err := s.app.ForgotPassword(c.Request.Context(), in.Identifier); err != nil {
		s.fail(c, err)
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"ok": true})
}

func (s *Server) verifyCode(c *gin.Context) {
	var in struct {
		Identifier string `json:"identifier"`
		Code       string `json:"code"`
	}
	if !s.bind(c, &in) {
		return
	}
	tok, err := s.app.VerifyResetCode(c.Request.Context(), in.Identifier, in.Code)
	if err != nil {
		s.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"reset_token": tok})
}

func (s *Server) resetPassword(c *gin.Context) {
	var in struct {
		Token    string `json:"reset_token"`
		Password string `json:"password"`
	}
	if !s.bind(c, &in) {
		return
	}
	if err := s.app.ResetPassword(c.Request.Context(), in.Token, in.Password); err != nil {
		s.fail(c, err)
		return
	}
	s.clearCookie(c)
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (s *Server) handleAvailable(c *gin.Context) {
	except := ""
	if tok, err := c.Cookie(cookieName); err == nil {
		if u, err := s.app.Authenticate(c.Request.Context(), tok); err == nil {
			if cr, err := s.app.CreatorContext(c.Request.Context(), u); err == nil {
				except = cr.Account.ID
			}
		}
	}
	ok, why, err := s.app.HandleAvailable(c.Request.Context(), strings.ToLower(c.Param("handle")), except)
	if err != nil {
		s.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"available": ok, "reason": why})
}

type pageView struct {
	ID          string            `json:"id"`
	Sandbox     bool              `json:"sandbox,omitempty"`
	Name        string            `json:"name"`
	Handle      string            `json:"handle"`
	Category    string            `json:"category"`
	Bio         string            `json:"bio"`
	Thanks      string            `json:"thanks"`
	Amounts     []d.Money         `json:"amounts"`
	DefaultIdx  int               `json:"default_idx"`
	AllowCustom bool              `json:"allow_custom"`
	MinAmount   d.Money           `json:"min_amount"`
	Socials     map[string]string `json:"socials"`
	Channels    d.ChannelSet      `json:"channels"`
	ComingSoon  []d.Channel       `json:"coming_soon,omitempty"`
}

func (s *Server) comingSoon() []d.Channel {
	out := []d.Channel{}
	for _, c := range []d.Channel{d.ChannelMpesa, d.ChannelEmola, d.ChannelCard} {
		if s.app.ComingSoon.Has(c) {
			out = append(out, c)
		}
	}
	return out
}

func toPage(a *d.Account) pageView {
	return pageView{ID: a.ID, Name: a.Name, Handle: a.Handle, Category: a.Category, Bio: a.Bio, Thanks: a.Thanks, Amounts: a.Amounts, DefaultIdx: a.DefaultIdx,
		AllowCustom: a.AllowCustom, MinAmount: a.MinAmount, Socials: a.Socials, Channels: a.Channels}
}

func (s *Server) publicPage(c *gin.Context) {
	a, err := s.app.PublicPage(c.Request.Context(), c.Param("handle"))
	if err != nil {
		s.fail(c, err)
		return
	}
	p := toPage(a)
	p.ID, p.Sandbox = "", s.app.Sandbox
	p.ComingSoon = s.comingSoon()
	soon := s.app.ComingSoon
	p.Channels = d.ChannelSet{Mpesa: p.Channels.Mpesa && !soon.Mpesa, Emola: p.Channels.Emola && !soon.Emola, Card: p.Channels.Card && !soon.Card}
	c.Header("Cache-Control", "public, max-age=30")
	c.JSON(http.StatusOK, p)
}

type supportStatusView struct {
	ID          string          `json:"id"`
	Ref         string          `json:"ref"`
	Status      d.SupportStatus `json:"status"`
	NextAction  string          `json:"next_action"`
	ExpiresAt   time.Time       `json:"expires_at"`
	Amount      d.Money         `json:"amount"`
	Channel     d.Channel       `json:"channel"`
	Phone       string          `json:"phone,omitempty"`
	Card        string          `json:"card,omitempty"`
	FailReason  string          `json:"fail_reason,omitempty"`
	Name        string          `json:"name"`
	ConfirmedAt *time.Time      `json:"confirmed_at,omitempty"`
	CreatedAt   time.Time       `json:"created_at"`
	Page        string          `json:"page"`
	Thanks      string          `json:"thanks"`
	Simulate    bool            `json:"simulate"`
}

func (s *Server) supportOut(sp *d.Support, acc *d.Account) supportStatusView {
	v := supportStatusView{ID: sp.ID, Ref: sp.Ref, Status: sp.Status, NextAction: sp.NextAction, ExpiresAt: sp.ExpiresAt, Amount: sp.Amount, Channel: sp.Channel,
		FailReason: sp.FailReason, Name: sp.SupporterName, ConfirmedAt: sp.ConfirmedAt, CreatedAt: sp.CreatedAt, Simulate: s.app.AllowSimulate}
	if sp.Channel.IsWallet() {
		v.Phone = maskPhone(sp.PayerPhone)
	} else {
		v.Card = sp.CardBrand + ":" + sp.CardLast4
	}
	if acc != nil {
		v.Page, v.Thanks = acc.Name, acc.Thanks
	}
	return v
}

func (s *Server) createSupport(c *gin.Context) {
	var in struct {
		Amount    d.Money   `json:"amount"`
		Channel   d.Channel `json:"channel"`
		Phone     string    `json:"phone"`
		CardToken string    `json:"card_token"`
		CardBrand string    `json:"card_brand"`
		CardLast4 string    `json:"card_last4"`
		Name      string    `json:"name"`
		Net       string    `json:"net"`
		Message   string    `json:"message"`
	}
	if !s.bind(c, &in) {
		return
	}
	sp, err := s.app.CreateSupport(c.Request.Context(), c.Param("handle"), app.CreateSupportInput{
		Amount: in.Amount, Channel: in.Channel, Phone: in.Phone, CardToken: in.CardToken, CardBrand: in.CardBrand, CardLast4: in.CardLast4,
		Name: in.Name, Net: in.Net, Message: in.Message, IdemKey: c.GetHeader("Idempotency-Key"),
	})
	if err != nil {
		s.fail(c, err)
		return
	}
	c.JSON(http.StatusCreated, s.supportOut(sp, nil))
}

func validID(id string) bool {
	if len(id) != 36 {
		return false
	}
	for i, r := range id {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			if r != '-' {
				return false
			}
		} else if !(r >= '0' && r <= '9' || r >= 'a' && r <= 'f') {
			return false
		}
	}
	return true
}

func (s *Server) idParam(c *gin.Context) (string, bool) {
	id := c.Param("id")
	if !validID(id) {
		abort(c, http.StatusNotFound, "not_found", "Não encontrado.")
		return "", false
	}
	return id, true
}

func (s *Server) supportStatus(c *gin.Context) {
	id, ok := s.idParam(c)
	if !ok {
		return
	}
	sp, acc, err := s.app.SupportStatus(c.Request.Context(), id)
	if err != nil {
		s.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, s.supportOut(sp, acc))
}

func (s *Server) submitOTP(c *gin.Context) {
	id, ok := s.idParam(c)
	if !ok {
		return
	}
	var in struct {
		Code string `json:"code"`
	}
	if !s.bind(c, &in) {
		return
	}
	sp, acc, err := s.app.SubmitOTP(c.Request.Context(), id, in.Code)
	if err != nil {
		s.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, s.supportOut(sp, acc))
}

func (s *Server) resend(c *gin.Context) {
	id, ok := s.idParam(c)
	if !ok {
		return
	}
	sp, err := s.app.ResendSupport(c.Request.Context(), id)
	if err != nil {
		s.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, s.supportOut(sp, nil))
}

func (s *Server) cancel(c *gin.Context) {
	id, ok := s.idParam(c)
	if !ok {
		return
	}
	if err := s.app.CancelSupport(c.Request.Context(), id); err != nil {
		s.fail(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (s *Server) simulate(c *gin.Context) {
	id, ok := s.idParam(c)
	if !ok {
		return
	}
	var in struct {
		Success bool `json:"success"`
	}
	if !s.bind(c, &in) {
		return
	}
	sp, acc, err := s.app.SimulatePayment(c.Request.Context(), id, in.Success)
	if err != nil {
		s.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, s.supportOut(sp, acc))
}

func (s *Server) webhook(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		abort(c, http.StatusBadRequest, "bad_request", "Pedido inválido.")
		return
	}
	if err := s.app.PaymentWebhook(c.Request.Context(), c.Request.Header, body); err != nil {
		s.fail(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (s *Server) splitApproval(c *gin.Context) {
	var in struct {
		Token string `json:"token"`
	}
	if !s.bind(c, &in) {
		return
	}
	v, err := s.app.SplitApproval(c.Request.Context(), in.Token)
	if err != nil {
		s.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, v)
}

func (s *Server) approveSplit(c *gin.Context) {
	var in struct {
		Token string `json:"token"`
	}
	if !s.bind(c, &in) {
		return
	}
	v, err := s.app.ApproveSplit(c.Request.Context(), in.Token)
	if err != nil {
		s.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, v)
}

func (s *Server) creator(c *gin.Context) {
	cr, err := s.app.CreatorContext(c.Request.Context(), user(c))
	if err != nil {
		s.fail(c, err)
		c.Abort()
		return
	}
	c.Set("creator", cr)
	c.Next()
}

func creator(c *gin.Context) *app.Creator {
	v, _ := c.Get("creator")
	return v.(*app.Creator)
}

func (s *Server) overview(c *gin.Context) {
	o, err := s.app.Overview(c.Request.Context(), creator(c), c.Query("range"))
	if err != nil {
		s.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, o)
}

func supportFilter(c *gin.Context) ports.SupportFilter {
	f := ports.SupportFilter{Query: c.Query("q")}
	if len(f.Query) > 80 {
		f.Query = f.Query[:80]
	}
	if ch := d.Channel(c.Query("channel")); ch.Valid() {
		f.Channel = ch
	}
	switch st := d.SupportStatus(c.Query("status")); st {
	case d.SupportConfirmed, d.SupportPending, d.SupportFailed:
		f.Status = st
	}
	f.Limit, _ = strconv.Atoi(c.DefaultQuery("limit", "50"))
	f.Offset, _ = strconv.Atoi(c.DefaultQuery("offset", "0"))
	return f
}

func (s *Server) listSupports(c *gin.Context) {
	l, err := s.app.ListSupports(c.Request.Context(), creator(c), supportFilter(c))
	if err != nil {
		s.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, l)
}

func csvSafe(v string) string {
	if v != "" && strings.ContainsRune("=+-@\t\r", rune(v[0])) {
		return "'" + v
	}
	return v
}

func (s *Server) exportSupports(c *gin.Context) {
	items, err := s.app.ExportSupports(c.Request.Context(), creator(c), supportFilter(c))
	if err != nil {
		s.fail(c, err)
		return
	}
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", `attachment; filename="apoios.csv"`)
	_, _ = c.Writer.Write([]byte("\xef\xbb\xbf"))
	w := csv.NewWriter(c.Writer)
	w.Comma = ';'
	_ = w.Write([]string{"Data", "Nome", "Mensagem", "Valor (MT)", "Canal", "Contacto", "Estado", "Referência"})
	chName := map[d.Channel]string{d.ChannelMpesa: "M-Pesa", d.ChannelEmola: "e-Mola", d.ChannelCard: "Cartão"}
	stName := map[d.SupportStatus]string{d.SupportConfirmed: "Confirmado", d.SupportPending: "Pendente", d.SupportFailed: "Falhou", d.SupportExpired: "Expirou"}
	for _, x := range items {
		if x.Status != d.SupportConfirmed {
			x.SupporterName, x.Message = "", ""
		}
		contact := d.FormatMZPhone(x.PayerPhone)
		if x.Channel == d.ChannelCard {
			contact = x.CardBrand + " " + x.CardLast4
		}
		_ = w.Write([]string{x.CreatedAt.Format("2006-01-02 15:04"), csvSafe(x.SupporterName), csvSafe(x.Message), strconv.FormatInt(int64(x.Amount)/100, 10),
			chName[x.Channel], contact, stName[x.Status], x.Ref})
	}
	w.Flush()
}

func (s *Server) getSupport(c *gin.Context) {
	id, ok := s.idParam(c)
	if !ok {
		return
	}
	v, err := s.app.GetSupport(c.Request.Context(), creator(c), id)
	if err != nil {
		s.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, v)
}

func (s *Server) markUnread(c *gin.Context) {
	id, ok := s.idParam(c)
	if !ok {
		return
	}
	if err := s.app.MarkUnread(c.Request.Context(), creator(c), id); err != nil {
		s.fail(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (s *Server) getPage(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"page": toPage(creator(c).Account), "categories": d.Categories, "can_edit": creator(c).Member.IsAdmin})
}

func (s *Server) updatePage(c *gin.Context) {
	var in app.PageInput
	if !s.bind(c, &in) {
		return
	}
	a, err := s.app.UpdatePage(c.Request.Context(), creator(c), in)
	if err != nil {
		s.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"page": toPage(a), "categories": d.Categories, "can_edit": true})
}

func (s *Server) group(c *gin.Context) {
	g, err := s.app.Group(c.Request.Context(), creator(c))
	if err != nil {
		s.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, g)
}

func (s *Server) autoSplit(c *gin.Context) {
	var in struct {
		On bool `json:"on"`
	}
	if !s.bind(c, &in) {
		return
	}
	if err := s.app.SetAutoSplit(c.Request.Context(), creator(c), in.On); err != nil {
		s.fail(c, err)
		return
	}
	s.group(c)
}

func (s *Server) inviteMember(c *gin.Context) {
	var in app.InviteMemberInput
	if !s.bind(c, &in) {
		return
	}
	m, err := s.app.InviteMember(c.Request.Context(), creator(c), in)
	if err != nil {
		s.fail(c, err)
		return
	}
	c.JSON(http.StatusCreated, m)
}

func (s *Server) removeMember(c *gin.Context) {
	id, ok := s.idParam(c)
	if !ok {
		return
	}
	if err := s.app.RemoveMember(c.Request.Context(), creator(c), id); err != nil {
		s.fail(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (s *Server) propose(c *gin.Context) {
	var in struct {
		Splits []d.MemberPct `json:"splits"`
	}
	if !s.bind(c, &in) {
		return
	}
	g, err := s.app.ProposeSplit(c.Request.Context(), creator(c), in.Splits)
	if err != nil {
		s.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, g)
}

func (s *Server) cancelProposal(c *gin.Context) {
	if err := s.app.CancelProposal(c.Request.Context(), creator(c)); err != nil {
		s.fail(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (s *Server) wallet(c *gin.Context) {
	w, err := s.app.Wallet(c.Request.Context(), creator(c))
	if err != nil {
		s.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, w)
}

func (s *Server) addPayoutAccount(c *gin.Context) {
	var in app.PayoutAccountInput
	if !s.bind(c, &in) {
		return
	}
	p, err := s.app.AddPayoutAccount(c.Request.Context(), creator(c), in)
	if err != nil {
		s.fail(c, err)
		return
	}
	c.JSON(http.StatusCreated, p)
}

func (s *Server) deletePayoutAccount(c *gin.Context) {
	id, ok := s.idParam(c)
	if !ok {
		return
	}
	if err := s.app.DeletePayoutAccount(c.Request.Context(), creator(c), id); err != nil {
		s.fail(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (s *Server) requestWithdrawal(c *gin.Context) {
	var in struct {
		Amount          d.Money `json:"amount"`
		PayoutAccountID string  `json:"payout_account_id"`
	}
	if !s.bind(c, &in) {
		return
	}
	w, err := s.app.RequestWithdrawal(c.Request.Context(), creator(c), in.Amount, in.PayoutAccountID)
	if err != nil {
		s.fail(c, err)
		return
	}
	c.JSON(http.StatusCreated, w)
}

func (s *Server) adminDashboard(c *gin.Context) {
	v, err := s.app.AdminDashboard(c.Request.Context(), user(c))
	if err != nil {
		s.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, v)
}

func (s *Server) adminAccounts(c *gin.Context) {
	f := ports.AccountFilter{Query: c.Query("q")}
	switch c.Query("status") {
	case "active", "pending", "suspended":
		f.Status = d.AccountStatus(c.Query("status"))
	case "negotiated":
		f.Negotiated = true
	}
	v, err := s.app.AdminAccounts(c.Request.Context(), user(c), f)
	if err != nil {
		s.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": v})
}

func (s *Server) adminAccount(c *gin.Context) {
	id, ok := s.idParam(c)
	if !ok {
		return
	}
	v, err := s.app.AdminAccount(c.Request.Context(), user(c), id)
	if err != nil {
		s.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, v)
}

func (s *Server) adminFee(c *gin.Context) {
	id, ok := s.idParam(c)
	if !ok {
		return
	}
	var in app.FeeInput
	if !s.bind(c, &in) {
		return
	}
	v, err := s.app.AdminSetFee(c.Request.Context(), user(c), id, in)
	if err != nil {
		s.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, v)
}

func (s *Server) adminStatus(c *gin.Context) {
	id, ok := s.idParam(c)
	if !ok {
		return
	}
	var in struct {
		Status d.AccountStatus `json:"status"`
		Note   string          `json:"note"`
	}
	if !s.bind(c, &in) {
		return
	}
	v, err := s.app.AdminSetStatus(c.Request.Context(), user(c), id, in.Status, in.Note)
	if err != nil {
		s.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, v)
}

func (s *Server) adminWithdrawals(c *gin.Context) {
	v, err := s.app.AdminWithdrawals(c.Request.Context(), user(c))
	if err != nil {
		s.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, v)
}

func (s *Server) adminWithdrawalAction(c *gin.Context) {
	id, ok := s.idParam(c)
	if !ok {
		return
	}
	var in app.WithdrawalAction
	if !s.bind(c, &in) {
		return
	}
	if err := s.app.AdminActOnWithdrawal(c.Request.Context(), user(c), id, in); err != nil {
		s.fail(c, err)
		return
	}
	s.adminWithdrawals(c)
}

func (s *Server) adminInvites(c *gin.Context) {
	v, err := s.app.AdminInvites(c.Request.Context(), user(c))
	if err != nil {
		s.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": v})
}

func (s *Server) adminCreateInvite(c *gin.Context) {
	var in app.InviteInput
	if !s.bind(c, &in) {
		return
	}
	v, err := s.app.AdminCreateInvite(c.Request.Context(), user(c), in)
	if err != nil {
		s.fail(c, err)
		return
	}
	c.JSON(http.StatusCreated, v)
}

func (s *Server) adminRevokeInvite(c *gin.Context) {
	if err := s.app.AdminRevokeInvite(c.Request.Context(), user(c), c.Param("code")); err != nil {
		s.fail(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (s *Server) adminWithdrawal(c *gin.Context) {
	id, ok := s.idParam(c)
	if !ok {
		return
	}
	v, err := s.app.AdminWithdrawal(c.Request.Context(), user(c), id)
	if err != nil {
		s.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, v)
}

func (s *Server) sendPDF(c *gin.Context, b []byte, name string, err error) {
	if err != nil {
		s.fail(c, err)
		return
	}
	c.Header("Content-Disposition", `attachment; filename="`+name+`"`)
	c.Header("Cache-Control", "no-store")
	c.Data(http.StatusOK, "application/pdf", b)
}

func (s *Server) adminWithdrawalPDF(c *gin.Context) {
	id, ok := s.idParam(c)
	if !ok {
		return
	}
	b, name, err := s.app.AdminPaymentOrdersPDF(c.Request.Context(), user(c), id, "")
	s.sendPDF(c, b, name, err)
}

func (s *Server) adminWithdrawalsPDF(c *gin.Context) {
	b, name, err := s.app.AdminPaymentOrdersPDF(c.Request.Context(), user(c), "", d.WithdrawalStatus(c.DefaultQuery("status", "approved")))
	s.sendPDF(c, b, name, err)
}

func maskPhone(n string) string {
	if len(n) != 9 {
		return ""
	}
	return n[:2] + " ••• " + n[5:]
}
