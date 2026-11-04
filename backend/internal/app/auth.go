package app

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	d "github.com/refresco/refresco-v2/backend/internal/domain"
	"github.com/refresco/refresco-v2/backend/internal/ports"
)

const (
	resetCodeTTL      = 10 * time.Minute
	resetTokenTTL     = 15 * time.Minute
	resetResendAfter  = 45 * time.Second
	resetMaxAttempts  = 5
	resetDailyCodes   = 5
	resetDailyGuesses = 10
	loginMaxFailures  = 10
	loginLockFor      = 15 * time.Minute
)

type Session struct {
	User      *d.User
	Token     string
	ExpiresAt time.Time
}

func validPassword(pw string) error {
	if utf8.RuneCountInString(pw) < 8 {
		return d.Invalid("weak_password", "A palavra-passe tem de ter pelo menos 8 caracteres.")
	}
	if len(pw) > 72 {
		return d.Invalid("long_password", "A palavra-passe é demasiado longa.")
	}
	return nil
}

func (a *App) findUserByIdentifier(ctx context.Context, s ports.Store, id string) (*d.User, error) {
	id = strings.TrimSpace(id)
	if strings.Contains(id, "@") {
		e := d.NormalizeEmail(id)
		if e == "" {
			return nil, d.ErrNotFound
		}
		return s.Users().ByEmail(ctx, e)
	}
	p := d.NormalizeMZPhone(id)
	if p == "" {
		return nil, d.ErrNotFound
	}
	return s.Users().ByPhone(ctx, p)
}

func (a *App) Login(ctx context.Context, identifier, password string) (*Session, error) {
	u, err := a.findUserByIdentifier(ctx, a.Store, identifier)
	if err != nil {
		if errors.Is(err, d.ErrNotFound) {
			a.Hasher.Compare(a.dummyHash, password)
			return nil, d.ErrInvalidCredentials
		}
		return nil, err
	}
	if err := a.checkPassword(ctx, u, password); err != nil {
		return nil, err
	}
	return a.issue(ctx, u)
}

// checkPassword verifies a password and applies the per-account lockout.
func (a *App) checkPassword(ctx context.Context, u *d.User, password string) error {
	now := a.now()
	if u.LockedUntil != nil && now.Before(*u.LockedUntil) {
		a.Hasher.Compare(a.dummyHash, password)
		return d.ErrAccountLocked
	}
	if !a.Hasher.Compare(u.PasswordHash, password) {
		if err := a.Store.Users().RecordLoginFailure(ctx, u.ID, loginMaxFailures, now.Add(loginLockFor)); err != nil {
			a.Log.Warn("record login failure", "err", err)
		}
		return d.ErrInvalidCredentials
	}
	if u.FailedLogins != 0 || u.LockedUntil != nil {
		_ = a.Store.Users().ClearLoginFailures(ctx, u.ID)
	}
	return nil
}

// ConfirmPassword re-authenticates the signed-in user before a sensitive action.
func (a *App) ConfirmPassword(ctx context.Context, userID, password string) error {
	u, err := a.Store.Users().ByID(ctx, userID)
	if err != nil {
		return d.ErrUnauthorized
	}
	if err := a.checkPassword(ctx, u, password); err != nil {
		if errors.Is(err, d.ErrInvalidCredentials) {
			return d.Invalid("wrong_password", "Palavra-passe incorreta.")
		}
		return err
	}
	return nil
}

func (a *App) issue(ctx context.Context, u *d.User) (*Session, error) {
	now := a.now()
	sid := newID()
	tok, exp, err := a.Tokens.Issue(u, sid)
	if err != nil {
		return nil, err
	}
	if err := a.Store.Sessions().Create(ctx, &d.Session{ID: sid, UserID: u.ID, ExpiresAt: exp, CreatedAt: now}); err != nil {
		return nil, err
	}
	return &Session{User: u, Token: tok, ExpiresAt: exp}, nil
}

// Logout revokes the session behind a token, even if it has otherwise expired.
func (a *App) Logout(ctx context.Context, token string) {
	c, err := a.Tokens.Parse(token)
	if err != nil {
		return
	}
	if err := a.Store.Sessions().Revoke(ctx, c.SessionID, a.now()); err != nil {
		a.Log.Warn("revoke session", "err", err)
	}
}

func (a *App) Authenticate(ctx context.Context, token string) (*d.User, error) {
	c, err := a.Tokens.Parse(token)
	if err != nil {
		return nil, d.ErrUnauthorized
	}
	s, err := a.Store.Sessions().ByID(ctx, c.SessionID)
	if err != nil || s.UserID != c.UserID || s.RevokedAt != nil || a.now().After(s.ExpiresAt) {
		return nil, d.ErrUnauthorized
	}
	u, err := a.Store.Users().ByID(ctx, c.UserID)
	if err != nil || u.TokenVersion != c.TokenVersion || u.Role != c.Role {
		return nil, d.ErrUnauthorized
	}
	return u, nil
}

type InviteCheck struct {
	Valid   bool   `json:"valid"`
	Reason  string `json:"reason,omitempty"`
	Inviter string `json:"inviter,omitempty"`
}

func normCode(c string) string {
	c = strings.ToUpper(strings.TrimSpace(c))
	if len(c) > 24 {
		return ""
	}
	for _, r := range c {
		if !(r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-') {
			return ""
		}
	}
	return c
}

func (a *App) CheckInvite(ctx context.Context, code string) (InviteCheck, error) {
	c := normCode(code)
	if c == "" {
		return InviteCheck{Reason: "not_found"}, nil
	}
	inv, err := a.Store.Invites().ByCode(ctx, c)
	if errors.Is(err, d.ErrNotFound) {
		return InviteCheck{Reason: "not_found"}, nil
	}
	if err != nil {
		return InviteCheck{}, err
	}
	if !inv.Usable() {
		return InviteCheck{Reason: "not_found"}, nil
	}
	return InviteCheck{Valid: true, Inviter: "Equipa refres.co"}, nil
}

func (a *App) HandleAvailable(ctx context.Context, handle, exceptAccountID string) (bool, string, error) {
	if !d.ValidHandle(handle) {
		return false, "invalid", nil
	}
	if d.ReservedHandle(handle) {
		return false, "taken", nil
	}
	taken, err := a.Store.Accounts().HandleTaken(ctx, handle, exceptAccountID)
	if err != nil {
		return false, "", err
	}
	if taken {
		return false, "taken", nil
	}
	return true, "", nil
}

type RegisterInput struct {
	Code     string
	Name     string
	Handle   string
	Email    string
	Phone    string
	Password string
	Terms    bool
}

func (a *App) Register(ctx context.Context, in RegisterInput) (*Session, error) {
	code := normCode(in.Code)
	email := d.NormalizeEmail(in.Email)
	phone := d.NormalizeMZPhone(in.Phone)
	name := strings.TrimSpace(in.Name)
	handle := strings.ToLower(strings.TrimSpace(in.Handle))
	switch {
	case code == "":
		return nil, d.Invalid("invite_required", "Escreve o código que recebeste.")
	case utf8.RuneCountInString(name) < 2 || utf8.RuneCountInString(name) > 40:
		return nil, d.Invalid("invalid_name", "O nome da página deve ter entre 2 e 40 caracteres.")
	case email == "":
		return nil, d.Invalid("invalid_email", "Email inválido.")
	case phone == "" || d.WalletOf(phone) == "":
		return nil, d.Invalid("invalid_phone", "Número inválido. Deve começar por 84, 85, 86 ou 87.")
	case !in.Terms:
		return nil, d.Invalid("terms_required", "Aceita os termos para continuar.")
	}
	if err := validPassword(in.Password); err != nil {
		return nil, err
	}
	ok, why, err := a.HandleAvailable(ctx, handle, "")
	if err != nil {
		return nil, err
	}
	if !ok {
		if why == "taken" {
			return nil, d.Conflict("handle_taken", "Este link já está ocupado.")
		}
		return nil, d.Invalid("invalid_handle", "Usa 3 a 24 letras minúsculas, números ou pontos.")
	}
	hash, err := a.Hasher.Hash(in.Password)
	if err != nil {
		return nil, err
	}
	now := a.now()
	u := &d.User{ID: newID(), Email: email, Phone: phone, Name: name, PasswordHash: hash, Role: d.RoleCreator, CreatedAt: now}
	err = a.Store.Tx(ctx, func(s ports.Store) error {
		inv, err := s.Invites().ByCodeForUpdate(ctx, code)
		if errors.Is(err, d.ErrNotFound) {
			return errInviteInvalid
		}
		if err != nil {
			return err
		}
		if !inv.Usable() {
			return errInviteInvalid
		}
		if _, err := s.Users().ByEmail(ctx, email); err == nil {
			return d.Conflict("email_taken", "Já existe uma conta com este email.")
		}
		if _, err := s.Users().ByPhone(ctx, phone); err == nil {
			return d.Conflict("phone_taken", "Já existe uma conta com este telemóvel.")
		}
		if err := s.Users().Create(ctx, u); err != nil {
			return err
		}
		acc := &d.Account{
			ID: newID(), OwnerID: u.ID, Handle: handle, Name: name, Category: d.Categories[0],
			Thanks:  "Kanimambo! O teu apoio faz toda a diferença.",
			Amounts: []d.Money{100 * d.MT, 300 * d.MT, 500 * d.MT, 1000 * d.MT}, DefaultIdx: 1,
			AllowCustom: true, MinAmount: d.MinCustomFloor, Socials: map[string]string{},
			Channels: d.ChannelSet{Mpesa: true, Emola: true, Card: true},
			FeeBPS:   inv.FeeBPS, Status: d.AccountPending, StatusNote: "Falta validar o BI antes do primeiro levantamento.",
			InviteCode: inv.Code, AutoSplit: true, CreatedAt: now,
		}
		if acc.FeeBPS != d.DefaultFeeBPS {
			acc.FeeNote = "Convite " + inv.Code
		}
		if err := s.Accounts().Create(ctx, acc); err != nil {
			return err
		}
		uid := u.ID
		wallet := d.WalletOf(phone)
		m := &d.Member{ID: newID(), AccountID: acc.ID, UserID: &uid, Name: firstName(name), Role: "Admin", Pct: 100,
			Wallet: wallet, WalletNumber: phone, IsAdmin: true, Status: d.MemberActive, CreatedAt: now}
		if err := s.Members().Create(ctx, m); err != nil {
			return err
		}
		if err := s.PayoutAccounts().Create(ctx, &d.PayoutAccount{ID: newID(), MemberID: m.ID, Kind: d.PayoutKind(wallet), Number: phone, IsMain: true, CreatedAt: now}); err != nil {
			return err
		}
		inv.Uses++
		if err := s.Invites().Update(ctx, inv); err != nil {
			return err
		}
		return a.audit(ctx, s, acc.ID, "account", "Conta criada com o convite "+inv.Code)
	})
	if err != nil {
		return nil, err
	}
	return a.issue(ctx, u)
}

var errInviteInvalid = d.Invalid("invite_invalid", "Este código não é válido ou já foi usado.")

func firstName(n string) string {
	f := strings.Fields(n)
	if len(f) == 0 {
		return n
	}
	return f[0]
}

func (a *App) JoinWaitlist(ctx context.Context, email string) (int64, error) {
	e := d.NormalizeEmail(email)
	if e == "" {
		return 0, d.Invalid("invalid_email", "Email inválido.")
	}
	return a.Store.Waitlist().Join(ctx, e)
}

func (a *App) ForgotPassword(ctx context.Context, identifier string) error {
	u, err := a.findUserByIdentifier(ctx, a.Store, identifier)
	if errors.Is(err, d.ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	now := a.now()
	if last, err := a.Store.Resets().Latest(ctx, u.ID); err == nil && now.Sub(last.CreatedAt) < resetResendAfter {
		return nil
	}
	codes, guesses, err := a.Store.Resets().Since(ctx, u.ID, now.Add(-24*time.Hour))
	if err != nil {
		return err
	}
	if codes >= resetDailyCodes || guesses >= resetDailyGuesses {
		a.Log.Warn("password reset capped", "user", u.ID, "codes", codes, "guesses", guesses)
		return nil
	}
	code := randDigits(6)
	r := &d.PasswordReset{ID: newID(), UserID: u.ID, ExpiresAt: now.Add(resetCodeTTL), CreatedAt: now}
	r.CodeHash = a.mac("reset", r.ID, code)
	if err := a.Store.Resets().Create(ctx, r); err != nil {
		return err
	}
	msg := ports.Message{To: u.Email, Kind: "password_reset_code", Body: "O teu código refres.co: " + code + ". Expira em 10 minutos.", Code: code}
	if !strings.Contains(identifier, "@") {
		phone := msg
		phone.To = d.FormatMZPhone(u.Phone)
		a.notify(ctx, phone)
	}
	a.notify(ctx, msg)
	return nil
}

func (a *App) VerifyResetCode(ctx context.Context, identifier, code string) (string, error) {
	bad := d.Invalid("invalid_code", "Código inválido ou expirado.")
	u, err := a.findUserByIdentifier(ctx, a.Store, identifier)
	if err != nil {
		return "", bad
	}
	r, err := a.Store.Resets().Latest(ctx, u.ID)
	now := a.now()
	if err != nil || r.UsedAt != nil || r.VerifiedAt != nil || now.After(r.ExpiresAt) {
		return "", bad
	}
	if r.Attempts >= resetMaxAttempts {
		return "", d.ErrTooManyAttempts
	}
	if _, guesses, err := a.Store.Resets().Since(ctx, u.ID, now.Add(-24*time.Hour)); err != nil {
		return "", err
	} else if guesses >= resetDailyGuesses {
		return "", d.ErrTooManyAttempts
	}
	if !hmacEqual(r.CodeHash, a.mac("reset", r.ID, strings.TrimSpace(code))) {
		r.Attempts++
		_ = a.Store.Resets().Update(ctx, r)
		return "", bad
	}
	tok := randToken(32)
	r.ResetTokenHash = a.mac("reset-token", tok)
	r.VerifiedAt = &now
	r.ExpiresAt = now.Add(resetTokenTTL)
	if err := a.Store.Resets().Update(ctx, r); err != nil {
		return "", err
	}
	return tok, nil
}

func (a *App) ResetPassword(ctx context.Context, token, password string) error {
	if err := validPassword(password); err != nil {
		return err
	}
	hash, err := a.Hasher.Hash(password)
	if err != nil {
		return err
	}
	bad := d.Invalid("invalid_token", "O pedido expirou. Pede um novo código.")
	return a.Store.Tx(ctx, func(s ports.Store) error {
		r, err := s.Resets().ByTokenHash(ctx, a.mac("reset-token", token))
		now := a.now()
		if err != nil || r.UsedAt != nil || now.After(r.ExpiresAt) {
			return bad
		}
		u, err := s.Users().ByID(ctx, r.UserID)
		if err != nil {
			return err
		}
		u.PasswordHash = hash
		u.TokenVersion++
		if err := s.Sessions().RevokeUser(ctx, u.ID, now); err != nil {
			return err
		}
		if err := s.Users().ClearLoginFailures(ctx, u.ID); err != nil {
			return err
		}
		r.UsedAt = &now
		if err := s.Users().Update(ctx, u); err != nil {
			return err
		}
		return s.Resets().Update(ctx, r)
	})
}

func hmacEqual(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	var v byte
	for i := 0; i < len(a); i++ {
		v |= a[i] ^ b[i]
	}
	return v == 0
}

func (a *App) EnsureSuperAdmin(ctx context.Context, email, password string) error {
	e := d.NormalizeEmail(email)
	if e == "" || password == "" {
		return nil
	}
	if _, err := a.Store.Users().ByEmail(ctx, e); err == nil {
		return nil
	} else if !errors.Is(err, d.ErrNotFound) {
		return err
	}
	if err := validPassword(password); err != nil {
		return err
	}
	hash, err := a.Hasher.Hash(password)
	if err != nil {
		return err
	}
	return a.Store.Users().Create(ctx, &d.User{ID: newID(), Email: e, Name: "Super admin", PasswordHash: hash, Role: d.RoleSuperAdmin, CreatedAt: a.now()})
}
