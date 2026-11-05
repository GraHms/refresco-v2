// Command seed loads demo data (invites, the Tu Pra Tu Podcast page and 60 days of supports).
// It refuses to run when APP_ENV=production.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand"
	"os"
	"time"

	"github.com/refresco/refresco-v2/backend/internal/app"
	"github.com/refresco/refresco-v2/backend/internal/bootstrap"
	"github.com/refresco/refresco-v2/backend/internal/config"
	d "github.com/refresco/refresco-v2/backend/internal/domain"
)

type clock struct{ t time.Time }

func (c *clock) Now() time.Time { return c.t }

func main() {
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))
	if err := run(log); err != nil {
		log.Error("seed failed", "err", err)
		os.Exit(1)
	}
}

func run(log *slog.Logger) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if cfg.Production() {
		return fmt.Errorf("refusing to seed a production database")
	}
	cfg.AllowSimulate = true
	a, db, err := bootstrap.Build(cfg, log)
	if err != nil {
		return err
	}
	ctx := context.Background()
	clk := &clock{t: time.Now().UTC().AddDate(0, 0, -75)}
	a.Clock = clk
	a.AllowSimulate = true
	st := a.Store

	if _, err := st.Accounts().ByHandle(ctx, "tupratu"); err == nil {
		log.Info("demo data already present")
		return nil
	}
	var users int64
	if err := db.Table("users").Count(&users).Error; err != nil {
		return err
	}
	if users > 0 && os.Getenv("SEED_FORCE") != "true" {
		return fmt.Errorf("database already has %d users; demo data is only loaded into an empty database (SEED_FORCE=true overrides)", users)
	}
	for _, inv := range []d.Invite{
		{Code: "REFRES-BETA", Note: "Beta aberta", MaxUses: 10, FeeBPS: 1500},
		{Code: "TPT-2026", Note: "Tu Pra Tu Podcast", MaxUses: 1, FeeBPS: 1200},
		{Code: "MAPUTO-01", Note: "Evento Maputo Fast Forward", MaxUses: 5, FeeBPS: 1500},
	} {
		inv.Status, inv.CreatedAt = d.InviteActive, clk.t
		if err := st.Invites().Create(ctx, &inv); err != nil {
			return err
		}
	}
	if err := a.EnsureSuperAdmin(ctx, envOr("ADMIN_EMAIL", "admin@refres.co"), envOr("ADMIN_PASSWORD", "admin-refres-2026")); err != nil {
		return err
	}
	sess, err := a.Register(ctx, app.RegisterInput{Code: "TPT-2026", Name: "Tu Pra Tu Podcast", Handle: "tupratu", Email: "cardo@tupratu.co.mz",
		Phone: "843120457", Password: "tupratu-2026", Terms: true})
	if err != nil {
		return err
	}
	cr, err := a.CreatorContext(ctx, sess.User)
	if err != nil {
		return err
	}
	cr.Member.Name, cr.Member.Role = "Cardo", "Apresentador · Admin"
	if err := st.Members().Update(ctx, cr.Member); err != nil {
		return err
	}
	if _, err := a.UpdatePage(ctx, cr, app.PageInput{Name: "Tu Pra Tu Podcast", Handle: "tupratu", Category: "Podcast",
		Bio:     "Conversas descontraídas de Tu Pra Tu com o Cardo, da música à sociedade. Episódios completos no YouTube.",
		Thanks:  "Kanimambo! O teu apoio ajuda a trazer mais conversas de Tu Pra Tu.",
		Amounts: []d.Money{100 * d.MT, 300 * d.MT, 500 * d.MT, 1000 * d.MT}, DefaultIdx: 1, AllowCustom: true, MinAmount: 50 * d.MT,
		Socials:  map[string]string{"youtube": "@tupratupodcast", "spotify": "tupratu", "instagram": "tupratupodcast", "tiktok": "tupratupod", "x": "Tupratupodcast"},
		Channels: d.ChannelSet{Mpesa: true, Emola: true, Bim: true, Card: true}}); err != nil {
		return err
	}
	admin := &d.User{}
	if admin, err = st.Users().ByEmail(ctx, envOr("ADMIN_EMAIL", "admin@refres.co")); err != nil {
		return err
	}
	if _, err := a.AdminSetStatus(ctx, admin, cr.Account.ID, d.AccountActive, ""); err != nil {
		return err
	}
	if _, err := a.AdminSetFee(ctx, admin, cr.Account.ID, app.FeeInput{Enabled: true, FeeBPS: 1200, Months: 12, Note: "Parceiro de lançamento"}); err != nil {
		return err
	}
	p1, err := a.InviteMember(ctx, cr, app.InviteMemberInput{Name: "Produção", Role: "Equipa de produção", Wallet: d.ChannelEmola, Phone: "867742210"})
	if err != nil {
		return err
	}
	p2, err := a.InviteMember(ctx, cr, app.InviteMemberInput{Name: "Edição", Role: "Edição e cortes", Wallet: d.ChannelMpesa, Phone: "856401182"})
	if err != nil {
		return err
	}
	members, _ := st.Members().ByAccount(ctx, cr.Account.ID)
	for i := range members {
		m := &members[i]
		switch m.ID {
		case cr.Member.ID:
			m.Pct = 70
		case p1.ID:
			m.Pct, m.Status = 20, d.MemberActive
		case p2.ID:
			m.Pct, m.Status = 10, d.MemberActive
		}
		if err := st.Members().Update(ctx, m); err != nil {
			return err
		}
	}

	fans := []struct{ name, net, phone, msg string }{
		{"@helio.tembe", "x", "845123398", "Episódio brutal. Continuem com as conversas de Tu Pra Tu!"},
		{"Ivânia Cuamba", "", "862307714", "Para o microfone novo do estúdio."},
		{"@arsenio.mz", "instagram", "859910042", "Vejo todos os episódios no chapa a caminho da Baixa."},
		{"Edson Nhantumbo", "", "871185530", "Kanimambo pelo episódio. Precisávamos disto."},
		{"@yolanda.sitoe", "tiktok", "846602187", "Ri tanto que o cobrador olhou para mim."},
		{"Amílcar Chongo", "", "852047761", ""},
		{"Shelsia Manhiça", "", "848773402", "Para a equipa de edição: os cortes no TikTok estão demais."},
		{"@nadiaz", "x", "849931206", "Primeira vez que apoio um podcast. Valeu a pena."},
		{"Lurdes Novela", "", "", "Da turma de Comunicação da UEM. Obrigado pela palestra!"},
		{"Hugo Mabjaia", "", "", "Força, malta! Quando é o episódio ao vivo?"},
	}
	amounts := []d.Money{100, 300, 300, 300, 500, 500, 1000, 200, 250}
	rnd := rand.New(rand.NewSource(7))
	start := time.Now().UTC().AddDate(0, 0, -60)
	for day := 0; day <= 60; day++ {
		n := 1 + rnd.Intn(4)
		if start.AddDate(0, 0, day).Weekday() == time.Friday {
			n += 3
		}
		for k := 0; k < n; k++ {
			clk.t = start.AddDate(0, 0, day).Add(time.Duration(8+rnd.Intn(14))*time.Hour + time.Duration(rnd.Intn(60))*time.Minute)
			if clk.t.After(time.Now().UTC()) {
				continue
			}
			f := fans[rnd.Intn(len(fans))]
			in := app.CreateSupportInput{Amount: amounts[rnd.Intn(len(amounts))] * d.MT, Name: f.name, Net: f.net, Message: f.msg}
			if f.phone == "" {
				in.Channel, in.CardToken, in.CardBrand, in.CardLast4 = d.ChannelCard, "mock_tok_seed_card", []string{"visa", "mastercard"}[rnd.Intn(2)], fmt.Sprintf("%04d", rnd.Intn(10000))
			} else {
				in.Channel, in.Phone = d.WalletOf(f.phone), f.phone
			}
			s, err := a.CreateSupport(ctx, "tupratu", in)
			if err != nil {
				return err
			}
			if _, _, err := a.SimulatePayment(ctx, s.ID, rnd.Intn(20) != 0); err != nil {
				return err
			}
			if day < 55 {
				now := clk.t
				_ = st.Supports().SetRead(ctx, s.ID, &now)
			}
		}
	}
	clk.t = time.Now().UTC().Add(-50 * time.Hour)
	cr, _ = a.CreatorContext(ctx, sess.User)
	wallet, err := a.Wallet(ctx, cr)
	if err != nil {
		return err
	}
	if len(wallet.Accounts) > 0 && wallet.Balance > 2000*d.MT {
		if _, err := a.RequestWithdrawal(ctx, cr, 1500*d.MT, wallet.Accounts[0].ID); err != nil {
			return err
		}
	}
	log.Info("seeded", "creator", "cardo@tupratu.co.mz / tupratu-2026", "admin", envOr("ADMIN_EMAIL", "admin@refres.co"))
	return nil
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
