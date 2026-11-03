package app_test

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"math/rand"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/refresco/refresco-v2/backend/internal/adapters/notify"
	"github.com/refresco/refresco-v2/backend/internal/adapters/payments/mock"
	"github.com/refresco/refresco-v2/backend/internal/adapters/postgres"
	"github.com/refresco/refresco-v2/backend/internal/adapters/security"
	"github.com/refresco/refresco-v2/backend/internal/app"
	d "github.com/refresco/refresco-v2/backend/internal/domain"
	"github.com/refresco/refresco-v2/backend/internal/ports"
)

func setup(t *testing.T) (*app.App, *d.User) {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	schema := fmt.Sprintf("t_%d_%d", time.Now().UnixNano(), rand.Intn(1000))
	admin, err := postgres.Open(dsn, 2, false)
	if err != nil {
		t.Fatal(err)
	}
	if err := admin.Exec("CREATE SCHEMA " + schema).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { admin.Exec("DROP SCHEMA " + schema + " CASCADE") })
	sep := "?"
	for _, c := range dsn {
		if c == '?' {
			sep = "&"
		}
	}
	db, err := postgres.Open(dsn+sep+"search_path="+schema, 10, false)
	if err != nil {
		t.Fatal(err)
	}
	if err := postgres.Migrate(db); err != nil {
		t.Fatal(err)
	}
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	a := app.New(app.Deps{
		Store: postgres.New(db), Metrics: postgres.NewMetrics(db), Payments: mock.New([]byte("secret"), 0),
		Notifier: notify.Log{L: log}, Hasher: security.Bcrypt{Cost: 10}, Tokens: security.JWT{Secret: []byte("k"), TTL: time.Hour, Issuer: "t"},
		Clock: security.SystemClock{}, Log: log, Secret: []byte("s"), PublicURL: "http://x", Location: time.UTC, AllowSimulate: true,
	})
	ctx := context.Background()
	if err := a.Store.Invites().Create(ctx, &d.Invite{Code: "TEST-1", Note: "t", MaxUses: 1, FeeBPS: 1500, Status: d.InviteActive, CreatedAt: time.Now()}); err != nil {
		t.Fatal(err)
	}
	if err := a.EnsureSuperAdmin(ctx, "root@x.co", "root-password-1"); err != nil {
		t.Fatal(err)
	}
	root, _ := a.Store.Users().ByEmail(ctx, "root@x.co")
	return a, root
}

func TestDonationLifecycle(t *testing.T) {
	a, root := setup(t)
	ctx := context.Background()
	sess, err := a.Register(ctx, app.RegisterInput{Code: "test-1", Name: "Banda Teste", Handle: "bandateste", Email: "b@x.co", Phone: "845551234", Password: "password-123", Terms: true})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := a.Register(ctx, app.RegisterInput{Code: "TEST-1", Name: "Outra", Handle: "outra", Email: "o@x.co", Phone: "845551235", Password: "password-123", Terms: true}); err == nil {
		t.Fatal("single-use invite was reused")
	}
	cr, _ := a.CreatorContext(ctx, sess.User)
	m2, err := a.InviteMember(ctx, cr, app.InviteMemberInput{Name: "Baixo", Wallet: d.ChannelEmola, Phone: "861112222"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := a.ProposeSplit(ctx, cr, []d.MemberPct{{MemberID: cr.Member.ID, Pct: 70}, {MemberID: m2.ID, Pct: 30}}); err != nil {
		t.Fatal(err)
	}
	members, _ := a.Store.Members().ByAccount(ctx, cr.Account.ID)
	if members[0].Pct != 100 {
		t.Fatal("split applied before approval")
	}

	if _, err := a.CreateSupport(ctx, "bandateste", app.CreateSupportInput{Amount: 300 * d.MT, Channel: d.ChannelEmola, Phone: "845123398"}); err == nil {
		t.Fatal("wallet mismatch accepted")
	}
	if _, err := a.CreateSupport(ctx, "bandateste", app.CreateSupportInput{Amount: 20 * d.MT, Channel: d.ChannelMpesa, Phone: "845123398"}); err == nil {
		t.Fatal("amount below minimum accepted")
	}
	s, err := a.CreateSupport(ctx, "bandateste", app.CreateSupportInput{Amount: 1000 * d.MT, Channel: d.ChannelMpesa, Phone: "845123398", IdemKey: "idem-key-0001"})
	if err != nil {
		t.Fatal(err)
	}
	again, _ := a.CreateSupport(ctx, "bandateste", app.CreateSupportInput{Amount: 1000 * d.MT, Channel: d.ChannelMpesa, Phone: "845123398", IdemKey: "idem-key-0001"})
	if again.ID != s.ID {
		t.Fatal("idempotency key not honoured")
	}

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); _, _, _ = a.SimulatePayment(ctx, s.ID, true) }()
	}
	wg.Wait()
	got, _, _ := a.SupportStatus(ctx, s.ID)
	if got.Status != d.SupportConfirmed || got.Fee != 150*d.MT || got.Net != 850*d.MT {
		t.Fatalf("unexpected settlement %+v", got)
	}
	members, _ = a.Store.Members().ByAccount(ctx, cr.Account.ID)
	var total d.Money
	for _, m := range members {
		total += m.Balance
	}
	if total != 850*d.MT {
		t.Fatalf("balances %d, want %d (double settlement?)", total, 850*d.MT)
	}

	o, err := a.Overview(ctx, cr, "7d")
	if err != nil {
		t.Fatal(err)
	}
	if o.Gross != 1000*d.MT || o.Count != 1 || o.ByChannel.Mpesa != 1000*d.MT {
		t.Fatalf("aggregates wrong: %+v", o)
	}

	cr, _ = a.CreatorContext(ctx, sess.User)
	w, _ := a.Wallet(ctx, cr)
	if _, err := a.RequestWithdrawal(ctx, cr, 400*d.MT, w.Accounts[0].ID); err == nil {
		t.Fatal("below minimum withdrawal accepted")
	}
	if _, err := a.RequestWithdrawal(ctx, cr, 900*d.MT, w.Accounts[0].ID); err == nil {
		t.Fatal("overdraft accepted")
	}
	wd, err := a.RequestWithdrawal(ctx, cr, 600*d.MT, w.Accounts[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := a.AdminActOnWithdrawal(ctx, root, wd.ID, app.WithdrawalAction{Action: "approve"}); err == nil {
		t.Fatal("approved withdrawal for unverified account")
	}
	if err := a.AdminActOnWithdrawal(ctx, root, wd.ID, app.WithdrawalAction{Action: "reject", Reason: "Conta por verificar"}); err != nil {
		t.Fatal(err)
	}
	me, _ := a.Store.Members().ByID(ctx, cr.Member.ID)
	if me.Balance != 850*d.MT {
		t.Fatalf("refund missing: %d", me.Balance)
	}
	if err := a.AdminActOnWithdrawal(ctx, sess.User, wd.ID, app.WithdrawalAction{Action: "approve"}); err == nil {
		t.Fatal("creator acted as admin")
	}

	if _, err := a.AdminSetStatus(ctx, root, cr.Account.ID, d.AccountActive, ""); err != nil {
		t.Fatal(err)
	}
	wd2, _ := a.RequestWithdrawal(ctx, cr, 500*d.MT, w.Accounts[0].ID)
	for _, act := range []app.WithdrawalAction{{Action: "approve"}, {Action: "complete", Ref: "mp-pay-1"}} {
		if err := a.AdminActOnWithdrawal(ctx, root, wd2.ID, act); err != nil {
			t.Fatal(act.Action, err)
		}
	}
	if err := a.AdminActOnWithdrawal(ctx, root, wd2.ID, app.WithdrawalAction{Action: "reject", Reason: "NIB inválido"}); err == nil {
		t.Fatal("rejected a paid withdrawal")
	}
	ws, _ := a.Store.Withdrawals().List(ctx, ports.WithdrawalFilter{AccountID: cr.Account.ID, Statuses: []d.WithdrawalStatus{d.WithdrawalPaid}})
	if len(ws) != 1 || ws[0].TransferRef != "MP-PAY-1" {
		t.Fatalf("paid withdrawal not recorded: %+v", ws)
	}
}
