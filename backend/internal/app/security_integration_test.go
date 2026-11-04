package app_test

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/refresco/refresco-v2/backend/internal/adapters/security"
	"github.com/refresco/refresco-v2/backend/internal/app"
	d "github.com/refresco/refresco-v2/backend/internal/domain"
	"github.com/refresco/refresco-v2/backend/internal/ports"
)

type stepClock struct{ t time.Time }

func (c *stepClock) Now() time.Time { return c.t }

type capture struct {
	mu   sync.Mutex
	msgs []ports.Message
}

func (c *capture) Send(_ context.Context, m ports.Message) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.msgs = append(c.msgs, m)
	return nil
}

func (c *capture) last(kind string) *ports.Message {
	c.mu.Lock()
	defer c.mu.Unlock()
	for i := len(c.msgs) - 1; i >= 0; i-- {
		if c.msgs[i].Kind == kind {
			return &c.msgs[i]
		}
	}
	return nil
}

func code(t *testing.T, err error) string {
	t.Helper()
	var de *d.Error
	if !errors.As(err, &de) {
		t.Fatalf("expected domain error, got %v", err)
	}
	return de.Code
}

func TestSecurityControls(t *testing.T) {
	a, _ := setup(t)
	box := &capture{}
	a.Notifier = box
	ctx := context.Background()

	if r, _ := a.CheckInvite(ctx, "NOPE-123"); r.Valid || r.Reason != "not_found" {
		t.Fatalf("unknown invite: %+v", r)
	}
	sess, err := a.Register(ctx, app.RegisterInput{Code: "TEST-1", Name: "Seg Teste", Handle: "segteste", Email: "s@x.co", Phone: "845550000", Password: "password-123", Terms: true})
	if err != nil {
		t.Fatal(err)
	}
	if r, _ := a.CheckInvite(ctx, "TEST-1"); r.Valid || r.Reason != "not_found" {
		t.Fatalf("used invite must look like any invalid code: %+v", r)
	}

	t.Run("logout revokes the session", func(t *testing.T) {
		s, err := a.Login(ctx, "s@x.co", "password-123")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := a.Authenticate(ctx, s.Token); err != nil {
			t.Fatal(err)
		}
		a.Logout(ctx, s.Token)
		if _, err := a.Authenticate(ctx, s.Token); err == nil {
			t.Fatal("token still valid after logout")
		}
		if _, err := a.Authenticate(ctx, sess.Token); err != nil {
			t.Fatal("other sessions must survive a logout")
		}
	})

	t.Run("account locks after repeated failures", func(t *testing.T) {
		for i := 0; i < 10; i++ {
			if _, err := a.Login(ctx, "s@x.co", "wrong-password"); code(t, err) != "invalid_credentials" {
				t.Fatalf("attempt %d: %v", i, err)
			}
		}
		if _, err := a.Login(ctx, "s@x.co", "password-123"); code(t, err) != "account_locked" {
			t.Fatalf("correct password accepted while locked: %v", err)
		}
	})

	t.Run("password reset is capped per day", func(t *testing.T) {
		clk := &stepClock{t: time.Now().UTC()}
		a.Clock = clk
		defer func() { a.Clock = security.SystemClock{} }()
		sent := func() int { return len(box.msgs) }
		for round := 0; round < 2; round++ {
			before := sent()
			if err := a.ForgotPassword(ctx, "s@x.co"); err != nil || sent() != before+1 {
				t.Fatalf("round %d: code not sent (%v)", round, err)
			}
			for i := 0; i < 5; i++ {
				_, _ = a.VerifyResetCode(ctx, "s@x.co", "000000")
			}
			clk.t = clk.t.Add(time.Minute)
		}
		before := sent()
		_ = a.ForgotPassword(ctx, "s@x.co")
		if sent() != before {
			t.Fatal("a new code was sent after 10 wrong guesses in one day")
		}
		clk.t = clk.t.Add(25 * time.Hour)
		if _ = a.ForgotPassword(ctx, "s@x.co"); sent() != before+1 {
			t.Fatal("the cap must reset after 24 hours")
		}
	})

	t.Run("unpaid supports hide name and message", func(t *testing.T) {
		s, err := a.CreateSupport(ctx, "segteste", app.CreateSupportInput{Amount: 300 * d.MT, Channel: d.ChannelMpesa, Phone: "841110000", Name: "Troll", Message: "abuse"})
		if err != nil {
			t.Fatal(err)
		}
		_ = a.CancelSupport(ctx, s.ID)
		cr, _ := a.CreatorContext(ctx, sess.User)
		l, err := a.ListSupports(ctx, cr, ports.SupportFilter{})
		if err != nil {
			t.Fatal(err)
		}
		for _, it := range l.Items {
			if it.ID == s.ID && (it.Message != "" || it.Name != "" || !it.Hidden) {
				t.Fatalf("unpaid support leaked content: %+v", it)
			}
		}
	})

	t.Run("payment prompts are capped per phone", func(t *testing.T) {
		var last *d.Support
		for i := 0; i < 5; i++ {
			s, err := a.CreateSupport(ctx, "segteste", app.CreateSupportInput{Amount: 100 * d.MT, Channel: d.ChannelMpesa, Phone: "842220000"})
			if err != nil {
				t.Fatalf("push %d: %v", i, err)
			}
			last = s
		}
		if _, err := a.CreateSupport(ctx, "segteste", app.CreateSupportInput{Amount: 100 * d.MT, Channel: d.ChannelMpesa, Phone: "842220000"}); code(t, err) != "push_limit" {
			t.Fatalf("sixth prompt to the same number allowed: %v", err)
		}
		if _, err := a.ResendSupport(ctx, last.ID); code(t, err) != "push_limit" {
			t.Fatalf("resend bypassed the phone cap: %v", err)
		}
	})

	t.Run("adding a payout account needs the password and notifies", func(t *testing.T) {
		_ = a.Store.Users().ClearLoginFailures(ctx, sess.User.ID)
		cr, _ := a.CreatorContext(ctx, sess.User)
		in := app.PayoutAccountInput{Kind: d.PayoutEmola, Number: "861234567"}
		if _, err := a.AddPayoutAccount(ctx, cr, in); code(t, err) != "wrong_password" {
			t.Fatalf("payout account added without password: %v", err)
		}
		in.Password = "password-123"
		if _, err := a.AddPayoutAccount(ctx, cr, in); err != nil {
			t.Fatal(err)
		}
		var sms, mail bool
		for _, m := range box.msgs {
			if m.Kind == "payout_account_added" && strings.Contains(m.Body, "86 123 4567") {
				sms = sms || m.To == "+258 84 555 0000"
				mail = mail || m.To == "s@x.co"
			}
		}
		if !sms || !mail {
			t.Fatalf("owner not notified by SMS (%v) and email (%v)", sms, mail)
		}
	})

	t.Run("social handles reject links", func(t *testing.T) {
		cr, _ := a.CreatorContext(ctx, sess.User)
		p := app.PageInput{Name: "Seg Teste", Handle: "segteste", Category: "Podcast", Amounts: []d.Money{100 * d.MT, 300 * d.MT, 500 * d.MT, 1000 * d.MT},
			MinAmount: 50 * d.MT, AllowCustom: true, Channels: d.ChannelSet{Mpesa: true}, Socials: map[string]string{"spotify": "https://evil.example/login"}}
		if _, err := a.UpdatePage(ctx, cr, p); code(t, err) != "invalid_social" {
			t.Fatalf("URL accepted as a social handle: %v", err)
		}
	})
}
