package app

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base32"
	"encoding/hex"
	"log/slog"
	"math/big"
	"strings"
	"time"

	"github.com/google/uuid"

	d "github.com/refresco/refresco-v2/backend/internal/domain"
	"github.com/refresco/refresco-v2/backend/internal/ports"
)

type Deps struct {
	Store         ports.Store
	Metrics       ports.MetricsReader
	Payments      ports.PaymentGateway
	Notifier      ports.Notifier
	Docs          ports.DocumentRenderer
	Hasher        ports.PasswordHasher
	Tokens        ports.TokenIssuer
	Clock         ports.Clock
	Log           *slog.Logger
	Secret        []byte
	PublicURL     string
	Location      *time.Location
	AllowSimulate bool
	Sandbox       bool
	// ComingSoon is what can't be used yet: payments and payouts by e-Mola, and by
	// card and bank (Card covers both).
	ComingSoon d.ChannelSet
}

type App struct {
	Deps
	dummyHash string
}

func New(deps Deps) *App {
	if deps.Location == nil {
		deps.Location = time.UTC
	}
	a := &App{Deps: deps}
	if deps.Hasher != nil {
		a.dummyHash, _ = deps.Hasher.Hash(randToken(16))
	}
	return a
}

func (a *App) now() time.Time { return a.Clock.Now() }

func newID() string { return uuid.NewString() }

func (a *App) mac(parts ...string) string {
	h := hmac.New(sha256.New, a.Secret)
	h.Write([]byte(strings.Join(parts, "\x00")))
	return hex.EncodeToString(h.Sum(nil))
}

var b32 = base32.StdEncoding.WithPadding(base32.NoPadding)

func randToken(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return strings.ToLower(b32.EncodeToString(b))
}

const refAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"

func randCode(n int, alphabet string) string {
	out := make([]byte, n)
	max := big.NewInt(int64(len(alphabet)))
	for i := range out {
		v, err := rand.Int(rand.Reader, max)
		if err != nil {
			panic(err)
		}
		out[i] = alphabet[v.Int64()]
	}
	return string(out)
}

func randDigits(n int) string { return randCode(n, "0123456789") }

func (a *App) audit(ctx context.Context, s ports.Store, accountID, scope, text string) error {
	return s.Audit().Add(ctx, &d.AuditEntry{ID: newID(), AccountID: accountID, Scope: scope, Text: text, CreatedAt: a.now()})
}

func (a *App) notify(ctx context.Context, m ports.Message) {
	if err := a.Notifier.Send(ctx, m); err != nil {
		a.Log.Warn("notify failed", "kind", m.Kind, "err", err)
	}
}

// notifyMember reaches a member by phone and at their account's email, when they
// have an account: a phone number alone reaches nobody until SMS is connected.
func (a *App) notifyMember(ctx context.Context, m *d.Member, msg ports.Message) {
	msg.To = d.FormatMZPhone(m.WalletNumber)
	a.notify(ctx, msg)
	if m.UserID == nil {
		return
	}
	if u, err := a.Store.Users().ByID(ctx, *m.UserID); err == nil && u.Email != "" {
		msg.To = u.Email
		a.notify(ctx, msg)
	}
}

func pctLabel(b d.BPS) string {
	whole, frac := int(b)/100, int(b)%100
	if frac == 0 {
		return itoa(whole) + "%"
	}
	return itoa(whole) + "," + strings.TrimRight(pad2(frac), "0") + "%"
}

func pad2(n int) string {
	if n < 10 {
		return "0" + itoa(n)
	}
	return itoa(n)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	s := ""
	neg := n < 0
	if neg {
		n = -n
	}
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	if neg {
		s = "-" + s
	}
	return s
}

func fmtMT(m d.Money) string {
	v := int64(m) / 100
	s := ""
	for i, c := range reverse(itoa64(v)) {
		if i > 0 && i%3 == 0 {
			s = "." + s
		}
		s = string(c) + s
	}
	return s + " MT"
}

func itoa64(n int64) string { return itoa(int(n)) }

func reverse(s string) string {
	r := []rune(s)
	for i, j := 0, len(r)-1; i < j; i, j = i+1, j-1 {
		r[i], r[j] = r[j], r[i]
	}
	return string(r)
}
