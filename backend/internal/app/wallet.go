package app

import (
	"context"
	"strings"
	"time"
	"unicode/utf8"

	d "github.com/refresco/refresco-v2/backend/internal/domain"
	"github.com/refresco/refresco-v2/backend/internal/ports"
)

type PayoutAccountView struct {
	ID     string       `json:"id"`
	Kind   d.PayoutKind `json:"kind"`
	Number string       `json:"number"`
	Bank   string       `json:"bank,omitempty"`
	Holder string       `json:"holder,omitempty"`
	IsMain bool         `json:"is_main"`
}

type WithdrawalView struct {
	ID          string             `json:"id"`
	Code        string             `json:"code"`
	MemberID    string             `json:"member_id"`
	MemberName  string             `json:"member_name"`
	Amount      d.Money            `json:"amount"`
	DestKind    d.PayoutKind       `json:"dest_kind"`
	DestNumber  string             `json:"dest_number"`
	DestBank    string             `json:"dest_bank,omitempty"`
	DestHolder  string             `json:"dest_holder,omitempty"`
	Status      d.WithdrawalStatus `json:"status"`
	Reason      string             `json:"reason,omitempty"`
	Note        string             `json:"note,omitempty"`
	TransferRef string             `json:"transfer_ref,omitempty"`
	DueAt       time.Time          `json:"due_at"`
	DecidedAt   *time.Time         `json:"decided_at,omitempty"`
	PaidAt      *time.Time         `json:"paid_at,omitempty"`
	CreatedAt   time.Time          `json:"created_at"`
}

func payoutView(p d.PayoutAccount) PayoutAccountView {
	return PayoutAccountView{ID: p.ID, Kind: p.Kind, Number: maskDest(p.Kind, p.Number, false), Bank: p.BankName, Holder: p.Holder, IsMain: p.IsMain}
}

func maskDest(k d.PayoutKind, num string, full bool) string {
	if k == d.PayoutNIB {
		if full || len(num) < 9 {
			return groupNIB(num)
		}
		return "NIB " + num[:4] + " ···· " + num[len(num)-5:len(num)-1] + " " + num[len(num)-1:]
	}
	return d.FormatMZPhone(num)
}

func groupNIB(n string) string {
	var b strings.Builder
	for i, r := range n {
		if i > 0 && i%4 == 0 {
			b.WriteByte(' ')
		}
		b.WriteRune(r)
	}
	return b.String()
}

func withdrawalView(w d.Withdrawal, name string, full bool) WithdrawalView {
	return WithdrawalView{ID: w.ID, Code: w.Code(), MemberID: w.MemberID, MemberName: name, Amount: w.Amount, DestKind: w.DestKind,
		DestNumber: maskDest(w.DestKind, w.DestNumber, full), DestBank: w.DestBank, DestHolder: w.DestHolder, Status: w.Status, Reason: w.Reason, Note: w.Note,
		TransferRef: w.TransferRef, DueAt: w.DueAt, DecidedAt: w.DecidedAt, PaidAt: w.PaidAt, CreatedAt: w.CreatedAt}
}

type WalletView struct {
	MemberID        string              `json:"member_id"`
	Balance         d.Money             `json:"balance"`
	Processing      d.Money             `json:"processing"`
	ProcessingCount int                 `json:"processing_count"`
	PaidMonth       d.Money             `json:"paid_month"`
	PaidMonthCount  int                 `json:"paid_month_count"`
	MinWithdrawal   d.Money             `json:"min_withdrawal"`
	Accounts        []PayoutAccountView `json:"accounts"`
	Members         []MemberBalance     `json:"members"`
	Withdrawals     []WithdrawalView    `json:"withdrawals"`
	AccountStatus   d.AccountStatus     `json:"account_status"`
}

type MemberBalance struct {
	ID       string     `json:"id"`
	Name     string     `json:"name"`
	Pct      int        `json:"pct"`
	Balance  d.Money    `json:"balance"`
	LastPaid *d.Money   `json:"last_paid,omitempty"`
	LastWhen *time.Time `json:"last_when,omitempty"`
}

func (a *App) Wallet(ctx context.Context, c *Creator) (*WalletView, error) {
	members, err := a.Store.Members().ByAccount(ctx, c.Account.ID)
	if err != nil {
		return nil, err
	}
	ws, err := a.Store.Withdrawals().List(ctx, ports.WithdrawalFilter{AccountID: c.Account.ID})
	if err != nil {
		return nil, err
	}
	accts, err := a.Store.PayoutAccounts().ByMember(ctx, c.Member.ID)
	if err != nil {
		return nil, err
	}
	names := map[string]string{}
	v := &WalletView{MemberID: c.Member.ID, MinWithdrawal: d.MinWithdrawal, Accounts: []PayoutAccountView{}, Withdrawals: []WithdrawalView{}, AccountStatus: c.Account.Status}
	for _, m := range members {
		names[m.ID] = m.Name
		if m.ID == c.Member.ID {
			v.Balance = m.Balance
		}
	}
	now := a.now().In(a.Location)
	month := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, a.Location)
	last := map[string]d.Withdrawal{}
	for _, w := range ws {
		v.Withdrawals = append(v.Withdrawals, withdrawalView(w, names[w.MemberID], false))
		if w.Status == d.WithdrawalPaid {
			if p, ok := last[w.MemberID]; !ok || w.PaidAt.After(*p.PaidAt) {
				last[w.MemberID] = w
			}
		}
		if w.MemberID != c.Member.ID {
			continue
		}
		if w.Status.Open() {
			v.Processing += w.Amount
			v.ProcessingCount++
		}
		if w.Status == d.WithdrawalPaid && w.PaidAt != nil && !w.PaidAt.Before(month) {
			v.PaidMonth += w.Amount
			v.PaidMonthCount++
		}
	}
	for _, m := range members {
		mb := MemberBalance{ID: m.ID, Name: m.Name, Pct: m.Pct, Balance: m.Balance}
		if l, ok := last[m.ID]; ok {
			amt := l.Amount
			mb.LastPaid, mb.LastWhen = &amt, l.PaidAt
		}
		v.Members = append(v.Members, mb)
	}
	for _, p := range accts {
		v.Accounts = append(v.Accounts, payoutView(p))
	}
	return v, nil
}

type PayoutAccountInput struct {
	Kind     d.PayoutKind `json:"kind"`
	Number   string       `json:"number"`
	Bank     string       `json:"bank"`
	Holder   string       `json:"holder"`
	IsMain   bool         `json:"is_main"`
	Password string       `json:"password"`
}

var banks = map[string]bool{"BCI": true, "Millennium bim": true, "Standard Bank": true, "Absa": true, "Moza Banco": true, "FNB": true, "Nedbank": true, "Access Bank": true, "Ecobank": true, "Letshego": true}

var errPayoutSoon = d.Invalid("payout_coming_soon", "Por agora os levantamentos são só por M-Pesa. O e-Mola e o banco chegam em breve.")

func (a *App) payoutComingSoon(k d.PayoutKind) bool {
	switch k {
	case d.PayoutEmola:
		return a.ComingSoon.Emola
	case d.PayoutNIB:
		return a.ComingSoon.Card
	}
	return false
}

func (a *App) AddPayoutAccount(ctx context.Context, c *Creator, in PayoutAccountInput) (*PayoutAccountView, error) {
	if err := a.ConfirmPassword(ctx, c.User.ID, in.Password); err != nil {
		return nil, err
	}
	if a.payoutComingSoon(in.Kind) {
		return nil, errPayoutSoon
	}
	p := &d.PayoutAccount{ID: newID(), MemberID: c.Member.ID, Kind: in.Kind, IsMain: in.IsMain, CreatedAt: a.now()}
	switch in.Kind {
	case d.PayoutMpesa, d.PayoutEmola:
		n := d.NormalizeMZPhone(in.Number)
		if n == "" || string(d.WalletOf(n)) != string(in.Kind) {
			return nil, d.Invalid("wallet_mismatch", "O número não corresponde à carteira escolhida.")
		}
		p.Number = n
	case d.PayoutNIB:
		n := d.ValidateNIB(in.Number)
		if n == "" {
			return nil, d.Invalid("invalid_nib", "O NIB tem 21 dígitos.")
		}
		if !banks[in.Bank] {
			return nil, d.Invalid("invalid_bank", "Escolhe o banco.")
		}
		holder := strings.Join(strings.Fields(in.Holder), " ")
		if l := utf8.RuneCountInString(holder); l < 3 || l > 80 {
			return nil, d.Invalid("invalid_holder", "Escreve o nome do titular da conta.")
		}
		p.Number, p.BankName, p.Holder = n, in.Bank, holder
	default:
		return nil, d.Invalid("invalid_kind", "Tipo de conta inválido.")
	}
	err := a.Store.Tx(ctx, func(s ports.Store) error {
		existing, err := s.PayoutAccounts().ByMember(ctx, c.Member.ID)
		if err != nil {
			return err
		}
		if len(existing) >= 5 {
			return d.Invalid("too_many_accounts", "Podes ter no máximo 5 contas.")
		}
		for _, e := range existing {
			if e.Kind == p.Kind && e.Number == p.Number {
				return d.Conflict("duplicate_account", "Esta conta já existe.")
			}
		}
		if len(existing) == 0 {
			p.IsMain = true
		}
		if p.IsMain {
			if err := s.PayoutAccounts().ClearMain(ctx, c.Member.ID); err != nil {
				return err
			}
		}
		return s.PayoutAccounts().Create(ctx, p)
	})
	if err != nil {
		return nil, err
	}
	label := "M-Pesa " + d.FormatMZPhone(p.Number)
	switch p.Kind {
	case d.PayoutEmola:
		label = "e-Mola " + d.FormatMZPhone(p.Number)
	case d.PayoutNIB:
		label = p.BankName + " NIB terminado em " + p.Number[len(p.Number)-5:]
	}
	notice := "refres.co: foi adicionada a conta de levantamento " + label + " a " + c.Account.Name + ". Se não foste tu, muda a palavra-passe e contacta-nos."
	a.notify(ctx, ports.Message{To: d.FormatMZPhone(c.Member.WalletNumber), Kind: "payout_account_added", Body: notice})
	if c.User.Email != "" {
		a.notify(ctx, ports.Message{To: c.User.Email, Kind: "payout_account_added", Body: notice})
	}
	_ = a.audit(ctx, a.Store, c.Account.ID, "account", c.Member.Name+" adicionou a conta de levantamento "+label)
	v := payoutView(*p)
	return &v, nil
}

func (a *App) DeletePayoutAccount(ctx context.Context, c *Creator, id string) error {
	p, err := a.Store.PayoutAccounts().ByID(ctx, id)
	if err != nil || p.MemberID != c.Member.ID {
		return d.ErrNotFound
	}
	if p.IsMain {
		return d.Invalid("main_account", "Escolhe outra conta principal antes de remover esta.")
	}
	return a.Store.PayoutAccounts().Delete(ctx, id)
}

func (a *App) RequestWithdrawal(ctx context.Context, c *Creator, amount d.Money, payoutID string) (*WithdrawalView, error) {
	if amount < d.MinWithdrawal {
		return nil, d.Invalid("below_min", "O mínimo é "+fmtMT(d.MinWithdrawal)+".")
	}
	if amount%d.MT != 0 {
		return nil, d.Invalid("invalid_amount", "Valor inválido.")
	}
	p, err := a.Store.PayoutAccounts().ByID(ctx, payoutID)
	if err != nil || p.MemberID != c.Member.ID {
		return nil, d.Invalid("invalid_destination", "Escolhe a conta de destino.")
	}
	if a.payoutComingSoon(p.Kind) {
		return nil, errPayoutSoon
	}
	now := a.now()
	w := &d.Withdrawal{ID: newID(), AccountID: c.Account.ID, MemberID: c.Member.ID, Amount: amount, DestKind: p.Kind, DestNumber: p.Number, DestBank: p.BankName, DestHolder: p.Holder,
		Status: d.WithdrawalRequested, DueAt: now.Add(d.WithdrawalSLA), CreatedAt: now}
	err = a.Store.Tx(ctx, func(s ports.Store) error {
		acc, err := s.Accounts().ByID(ctx, c.Account.ID)
		if err != nil {
			return err
		}
		if acc.Status == d.AccountSuspended {
			return d.Forbidden("account_suspended", "A conta está suspensa.")
		}
		members, err := s.Members().LockByAccount(ctx, c.Account.ID)
		if err != nil {
			return err
		}
		var me *d.Member
		for i := range members {
			if members[i].ID == c.Member.ID {
				me = &members[i]
			}
		}
		if me == nil {
			return d.ErrForbidden
		}
		if me.Balance < amount {
			return d.Invalid("insufficient_balance", "O valor é maior do que o teu saldo.")
		}
		me.Balance -= amount
		if err := s.Members().Update(ctx, me); err != nil {
			return err
		}
		if err := s.Withdrawals().Create(ctx, w); err != nil {
			return err
		}
		return s.Ledger().Add(ctx, ports.LedgerEntry{MemberID: me.ID, AccountID: acc.ID, Kind: "withdrawal_hold", Amount: -amount, Ref: w.ID, CreatedAt: now})
	})
	if err != nil {
		return nil, err
	}
	v := withdrawalView(*w, c.Member.Name, false)
	v.Code = w.Code()
	return &v, nil
}
