package app

import (
	"context"
	"errors"
	"net/http"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	d "github.com/refresco/refresco-v2/backend/internal/domain"
	"github.com/refresco/refresco-v2/backend/internal/ports"
)

const MaxSupportAmount = 100_000 * d.MT

var supporterNets = map[string]bool{"x": true, "instagram": true, "tiktok": true, "facebook": true, "youtube": true}

var (
	last4Re   = regexp.MustCompile(`^\d{4}$`)
	cardTokRe = regexp.MustCompile(`^[A-Za-z0-9_\-]{8,128}$`)
	idemRe    = regexp.MustCompile(`^[A-Za-z0-9\-]{8,64}$`)
)

func (a *App) PublicPage(ctx context.Context, handle string) (*d.Account, error) {
	acc, err := a.Store.Accounts().ByHandle(ctx, strings.ToLower(handle))
	if err != nil {
		return nil, err
	}
	if acc.Status == d.AccountSuspended {
		return nil, d.ErrNotFound
	}
	return acc, nil
}

type CreateSupportInput struct {
	Amount       d.Money
	Channel      d.Channel
	Phone        string
	CardToken    string
	CardBrand    string
	CardLast4    string
	Name         string
	Net          string
	Message      string
	IdemKey      string
}

func (a *App) CreateSupport(ctx context.Context, handle string, in CreateSupportInput) (*d.Support, error) {
	acc, err := a.PublicPage(ctx, handle)
	if err != nil {
		return nil, err
	}
	if !in.Channel.Valid() || !acc.Channels.Has(in.Channel) {
		return nil, d.Invalid("channel_unavailable", "Esta forma de pagamento não está disponível.")
	}
	preset := false
	for _, v := range acc.Amounts {
		if v == in.Amount {
			preset = true
		}
	}
	if in.Amount%d.MT != 0 || in.Amount > MaxSupportAmount {
		return nil, d.Invalid("invalid_amount", "Valor inválido.")
	}
	if !preset && (!acc.AllowCustom || in.Amount < acc.MinAmount) {
		return nil, d.Invalid("amount_below_min", "O valor mínimo é "+fmtMT(acc.MinAmount)+".")
	}
	s := &d.Support{ID: newID(), AccountID: acc.ID, Amount: in.Amount, Preset: preset, Channel: in.Channel, Status: d.SupportPending, CreatedAt: a.now()}
	switch {
	case in.Channel.IsWallet():
		p := d.NormalizeMZPhone(in.Phone)
		if p == "" {
			return nil, d.Invalid("invalid_phone", "Número inválido. Deve começar por 84, 85, 86 ou 87.")
		}
		if d.WalletOf(p) != in.Channel {
			return nil, d.Invalid("wallet_mismatch", "Este número não pertence a esta carteira.")
		}
		s.PayerPhone = p
	default:
		brand := strings.ToLower(in.CardBrand)
		if brand != "visa" && brand != "mastercard" {
			return nil, d.Invalid("card_brand", "Só aceitamos Visa e Mastercard.")
		}
		if !last4Re.MatchString(in.CardLast4) || !cardTokRe.MatchString(in.CardToken) {
			return nil, d.Invalid("invalid_card", "Confirma o número, a validade e o CVV.")
		}
		s.CardBrand, s.CardLast4 = brand, in.CardLast4
	}
	name := strings.TrimSpace(in.Name)
	if utf8.RuneCountInString(name) > 40 {
		return nil, d.Invalid("name_too_long", "O nome tem no máximo 40 caracteres.")
	}
	if strings.HasPrefix(name, "@") {
		if !supporterNets[in.Net] {
			return nil, d.Invalid("invalid_network", "Escolhe a rede social.")
		}
		s.SupporterNet = in.Net
	}
	msg := strings.TrimSpace(in.Message)
	if utf8.RuneCountInString(msg) > 280 {
		return nil, d.Invalid("message_too_long", "A mensagem tem no máximo 280 caracteres.")
	}
	s.SupporterName, s.Message = name, msg
	if in.IdemKey != "" {
		if !idemRe.MatchString(in.IdemKey) {
			return nil, d.Invalid("invalid_idempotency_key", "Chave de idempotência inválida.")
		}
		if prev, err := a.Store.Supports().ByIdemKey(ctx, acc.ID, in.IdemKey); err == nil {
			return prev, nil
		}
		s.IdemKey = in.IdemKey
	}
	s.Ref = refPrefix(in.Channel) + "-" + randCode(7, refAlphabet)
	if err := a.Store.Supports().Create(ctx, s); err != nil {
		return nil, err
	}
	init, err := a.Payments.Initiate(ctx, ports.PaymentRequest{Ref: s.Ref, Amount: s.Amount, Channel: s.Channel, Phone: s.PayerPhone, CardToken: in.CardToken, Merchant: "REFRES*" + strings.ToUpper(acc.Name)})
	if err != nil {
		s.Status, s.FailReason = d.SupportFailed, "Pagamento indisponível"
		_ = a.Store.Supports().Update(ctx, s)
		a.Log.Error("payment initiate failed", "support", s.ID, "err", err)
		return nil, d.Unavailable("payment_unavailable", "Não foi possível iniciar o pagamento. Tenta de novo.")
	}
	s.ProviderRef, s.NextAction, s.ExpiresAt = init.ProviderRef, init.NextAction, init.ExpiresAt
	if err := a.Store.Supports().Update(ctx, s); err != nil {
		return nil, err
	}
	return s, nil
}

func refPrefix(c d.Channel) string {
	switch c {
	case d.ChannelMpesa:
		return "MP"
	case d.ChannelEmola:
		return "EM"
	}
	return "CD"
}

// SupportStatus refreshes a pending payment from the gateway.
func (a *App) SupportStatus(ctx context.Context, id string) (*d.Support, *d.Account, error) {
	s, err := a.Store.Supports().ByID(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	if s.Status == d.SupportPending && s.ProviderRef != "" {
		st, err := a.Payments.Status(ctx, s.ProviderRef)
		if err != nil {
			a.Log.Warn("payment status failed", "support", s.ID, "err", err)
		} else if s, err = a.apply(ctx, s.ID, st); err != nil {
			return nil, nil, err
		}
	}
	acc, err := a.Store.Accounts().ByID(ctx, s.AccountID)
	if err != nil {
		return nil, nil, err
	}
	return s, acc, nil
}

func (a *App) SubmitOTP(ctx context.Context, id, code string) (*d.Support, *d.Account, error) {
	s, err := a.Store.Supports().ByID(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	if s.Status != d.SupportPending || s.Channel != d.ChannelCard {
		return nil, nil, d.ErrStaleState
	}
	if len(code) != 6 || strings.Trim(code, "0123456789") != "" {
		return nil, nil, d.Invalid("invalid_code", "Introduz os 6 dígitos.")
	}
	st, err := a.Payments.SubmitOTP(ctx, s.ProviderRef, code)
	if err != nil {
		return nil, nil, d.Unavailable("payment_unavailable", "O banco não respondeu. Tenta de novo.")
	}
	if _, err := a.apply(ctx, s.ID, st); err != nil {
		return nil, nil, err
	}
	return a.SupportStatus(ctx, id)
}

func (a *App) ResendSupport(ctx context.Context, id string) (*d.Support, error) {
	s, err := a.Store.Supports().ByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if !s.Channel.IsWallet() || (s.Status != d.SupportPending && s.Status != d.SupportExpired) {
		return nil, d.ErrStaleState
	}
	acc, err := a.Store.Accounts().ByID(ctx, s.AccountID)
	if err != nil {
		return nil, err
	}
	init, err := a.Payments.Initiate(ctx, ports.PaymentRequest{Ref: s.Ref, Amount: s.Amount, Channel: s.Channel, Phone: s.PayerPhone, Merchant: "REFRES*" + strings.ToUpper(acc.Name)})
	if err != nil {
		return nil, d.Unavailable("payment_unavailable", "Não foi possível reenviar o pedido.")
	}
	var out *d.Support
	err = a.Store.Tx(ctx, func(st ports.Store) error {
		cur, err := st.Supports().ByIDForUpdate(ctx, id)
		if err != nil {
			return err
		}
		if cur.Status != d.SupportPending && cur.Status != d.SupportExpired {
			return d.ErrStaleState
		}
		cur.Status, cur.FailReason = d.SupportPending, ""
		cur.ProviderRef, cur.NextAction, cur.ExpiresAt = init.ProviderRef, init.NextAction, init.ExpiresAt
		out = cur
		return st.Supports().Update(ctx, cur)
	})
	return out, err
}

func (a *App) CancelSupport(ctx context.Context, id string) error {
	_, err := a.apply(ctx, id, ports.PaymentState{Status: d.SupportFailed, FailReason: "Cancelado pelo apoiante"})
	return err
}

func (a *App) SimulatePayment(ctx context.Context, id string, success bool) (*d.Support, *d.Account, error) {
	sim, ok := a.Payments.(ports.PaymentSimulator)
	if !a.AllowSimulate || !ok {
		return nil, nil, d.ErrNotFound
	}
	s, err := a.Store.Supports().ByID(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	if err := sim.Simulate(ctx, s.ProviderRef, success); err != nil {
		return nil, nil, err
	}
	return a.SupportStatus(ctx, id)
}

func (a *App) PaymentWebhook(ctx context.Context, h http.Header, body []byte) error {
	ref, st, err := a.Payments.ParseWebhook(h, body)
	if err != nil {
		return d.ErrUnauthorized
	}
	s, err := a.Store.Supports().ByProviderRef(ctx, ref)
	if errors.Is(err, d.ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	_, err = a.apply(ctx, s.ID, st)
	return err
}

// apply moves a pending support to its final state exactly once.
func (a *App) apply(ctx context.Context, id string, st ports.PaymentState) (*d.Support, error) {
	var out *d.Support
	err := a.Store.Tx(ctx, func(tx ports.Store) error {
		s, err := tx.Supports().ByIDForUpdate(ctx, id)
		if err != nil {
			return err
		}
		out = s
		now := a.now()
		if s.Status == d.SupportExpired && st.Status == d.SupportConfirmed {
			return a.settle(ctx, tx, s, now)
		}
		if s.Status != d.SupportPending {
			return nil
		}
		switch st.Status {
		case d.SupportConfirmed:
			return a.settle(ctx, tx, s, now)
		case d.SupportFailed:
			s.Status, s.FailReason = d.SupportFailed, st.FailReason
		case d.SupportPending:
			if !s.ExpiresAt.IsZero() && now.After(s.ExpiresAt.Add(30*time.Second)) {
				s.Status, s.FailReason = d.SupportExpired, "O pedido expirou"
			} else {
				return nil
			}
		case d.SupportExpired:
			s.Status, s.FailReason = d.SupportExpired, "O pedido expirou"
		default:
			return nil
		}
		return tx.Supports().Update(ctx, s)
	})
	return out, err
}

func (a *App) settle(ctx context.Context, tx ports.Store, s *d.Support, now time.Time) error {
	acc, err := tx.Accounts().ByID(ctx, s.AccountID)
	if err != nil {
		return err
	}
	members, err := tx.Members().LockByAccount(ctx, s.AccountID)
	if err != nil {
		return err
	}
	s.FeeBPS = acc.EffectiveFee(now)
	s.Fee = s.FeeBPS.Of(s.Amount)
	s.Net = s.Amount - s.Fee
	s.Status = d.SupportConfirmed
	s.ConfirmedAt = &now

	adminIdx := 0
	pcts := make([]int, len(members))
	for i, m := range members {
		if m.IsAdmin {
			adminIdx = i
		}
		pcts[i] = m.Pct
	}
	if !acc.AutoSplit {
		for i := range pcts {
			pcts[i] = 0
		}
		pcts[adminIdx] = 100
	}
	shares := d.Split(s.Net, pcts, adminIdx)
	splits := make([]d.SupportSplit, 0, len(members))
	for i := range members {
		if shares[i] == 0 {
			continue
		}
		m := &members[i]
		m.Balance += shares[i]
		if err := tx.Members().Update(ctx, m); err != nil {
			return err
		}
		if err := tx.Ledger().Add(ctx, ports.LedgerEntry{MemberID: m.ID, AccountID: acc.ID, Kind: "support_credit", Amount: shares[i], Ref: s.Ref, CreatedAt: now}); err != nil {
			return err
		}
		splits = append(splits, d.SupportSplit{SupportID: s.ID, MemberID: m.ID, Name: m.Name, Pct: pcts[i], Amount: shares[i]})
	}
	if err := tx.Supports().AddSplits(ctx, splits); err != nil {
		return err
	}
	if err := tx.Stats().RecordSupport(ctx, acc.ID, now.In(a.Location), s.Channel, s.Amount, s.Fee); err != nil {
		return err
	}
	return tx.Supports().Update(ctx, s)
}

func (a *App) ExpireStale(ctx context.Context) {
	if n, err := a.Store.Sessions().Purge(ctx, a.now().Add(-24*time.Hour)); err != nil {
		a.Log.Warn("purge sessions", "err", err)
	} else if n > 0 {
		a.Log.Info("purged sessions", "count", n)
	}
	n, err := a.Store.Supports().ExpireStale(ctx, a.now().Add(-10*time.Minute))
	if err != nil {
		a.Log.Warn("expire stale supports", "err", err)
	} else if n > 0 {
		a.Log.Info("expired stale supports", "count", n)
	}
}
