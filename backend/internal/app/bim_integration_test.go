package app_test

import (
	"context"
	"testing"

	"github.com/refresco/refresco-v2/backend/internal/app"
	d "github.com/refresco/refresco-v2/backend/internal/domain"
	"github.com/refresco/refresco-v2/backend/internal/ports"
)

func TestBimIzi(t *testing.T) {
	a, _ := setup(t)
	ctx := context.Background()
	sess, err := a.Register(ctx, app.RegisterInput{Code: "TEST-1", Name: "Izi Teste", Handle: "iziteste", Email: "i@x.co", Phone: "845559000", Password: "password-123", Terms: true})
	if err != nil {
		t.Fatal(err)
	}
	g := &gate{state: ports.PaymentState{Status: d.SupportPending}, inFlight: true}
	a.Payments = g
	a.ComingSoon = d.ChannelSet{Card: true}
	pay := func(channel d.Channel, phone string) (*d.Support, error) {
		return a.CreateSupport(ctx, "iziteste", app.CreateSupportInput{Amount: 10000, Channel: channel, Phone: phone})
	}

	t.Run("a new page offers it", func(t *testing.T) {
		acc, err := a.PublicPage(ctx, "iziteste")
		if err != nil || !acc.Channels.Bim {
			t.Fatalf("%+v %v", acc, err)
		}
	})

	t.Run("it takes numbers from any operator", func(t *testing.T) {
		for _, phone := range []string{"821234560", "831234560", "841234560", "861234560"} {
			s, err := pay(d.ChannelBim, phone)
			if err != nil {
				t.Fatalf("%s: %v", phone, err)
			}
			g.state = ports.PaymentState{Status: d.SupportFailed, FailReason: "test"}
			if _, _, err := a.SupportStatus(ctx, s.ID); err != nil {
				t.Fatal(err)
			}
			g.state = ports.PaymentState{Status: d.SupportPending}
		}
	})

	t.Run("the others keep their rules", func(t *testing.T) {
		if _, err := pay(d.ChannelBim, "123"); code(t, err) != "invalid_phone" {
			t.Fatal(err)
		}
		if _, err := pay(d.ChannelMpesa, "821234561"); code(t, err) != "wallet_mismatch" {
			t.Fatal(err)
		}
		if _, err := pay(d.ChannelEmola, "841234561"); code(t, err) != "wallet_mismatch" {
			t.Fatal(err)
		}
	})

	t.Run("cards stay refused", func(t *testing.T) {
		if _, err := a.CreateSupport(ctx, "iziteste", app.CreateSupportInput{Amount: 10000, Channel: d.ChannelCard, CardBrand: "visa", CardLast4: "4242", CardToken: "x"}); code(t, err) != "channel_unavailable" {
			t.Fatal(err)
		}
	})

	// IZI waits for the customer to dial, so a late payment must still be found.
	t.Run("the reconcile job credits one nobody is watching", func(t *testing.T) {
		s, err := pay(d.ChannelBim, "821234562")
		if err != nil {
			t.Fatal(err)
		}
		g.state = ports.PaymentState{Status: d.SupportConfirmed}
		a.ReconcilePayments(ctx)
		got, err := a.Store.Supports().ByID(ctx, s.ID)
		if err != nil || got.Status != d.SupportConfirmed {
			t.Fatalf("%+v %v", got, err)
		}
		g.state = ports.PaymentState{Status: d.SupportPending}
	})

	t.Run("reports count it", func(t *testing.T) {
		c, err := a.CreatorContext(ctx, sess.User)
		if err != nil {
			t.Fatal(err)
		}
		o, err := a.Overview(ctx, c, "7d")
		if err != nil {
			t.Fatal(err)
		}
		if o.ByChannel.Bim != 10000 || o.ByChannel.Sum() != 10000 {
			t.Fatalf("%+v", o.ByChannel)
		}
	})

	t.Run("a creator can turn it off", func(t *testing.T) {
		c, err := a.CreatorContext(ctx, sess.User)
		if err != nil {
			t.Fatal(err)
		}
		p := app.PageInput{Name: "Izi Teste", Handle: "iziteste", Category: "Podcast", Amounts: []d.Money{100 * d.MT, 300 * d.MT, 500 * d.MT, 1000 * d.MT}, DefaultIdx: 0, AllowCustom: true, MinAmount: 50 * d.MT, Channels: d.ChannelSet{Mpesa: true}, Socials: map[string]string{}}
		if _, err := a.UpdatePage(ctx, c, p); err != nil {
			t.Fatal(err)
		}
		acc, err := a.PublicPage(ctx, "iziteste")
		if err != nil || acc.Channels.Bim {
			t.Fatalf("still on: %+v %v", acc.Channels, err)
		}
		if _, err := pay(d.ChannelBim, "821234563"); code(t, err) != "channel_unavailable" {
			t.Fatal(err)
		}
	})
}
