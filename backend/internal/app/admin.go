package app

import (
	"context"
	"errors"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"

	d "github.com/refresco/refresco-v2/backend/internal/domain"
	"github.com/refresco/refresco-v2/backend/internal/ports"
)

func (a *App) requireAdmin(u *d.User) error {
	if u == nil || u.Role != d.RoleSuperAdmin {
		return d.ErrForbidden
	}
	return nil
}

type AdminDashboard struct {
	Revenue       d.Money             `json:"revenue"`
	Volume        d.Money             `json:"volume"`
	Count         int64               `json:"count"`
	AvgFeeBPS     int64               `json:"avg_fee_bps"`
	Balances      d.Money             `json:"balances"`
	ActiveCount   int                 `json:"active_count"`
	PendingAmount d.Money             `json:"pending_amount"`
	PendingCount  int                 `json:"pending_count"`
	OverdueCount  int                 `json:"overdue_count"`
	NegCount      int                 `json:"negotiated_count"`
	Series        []ports.SeriesPoint `json:"series"`
	Top           []AccountRow        `json:"top"`
}

type AccountRow struct {
	ID         string          `json:"id"`
	Name       string          `json:"name"`
	Handle     string          `json:"handle"`
	Category   string          `json:"category"`
	Owner      string          `json:"owner"`
	Count      int64           `json:"count"`
	Volume     d.Money         `json:"volume"`
	FeeBPS     d.BPS           `json:"fee_bps"`
	Negotiated bool            `json:"negotiated"`
	Revenue    d.Money         `json:"revenue"`
	Balance    d.Money         `json:"balance"`
	Status     d.AccountStatus `json:"status"`
}

func (a *App) last30() (time.Time, time.Time) {
	now := a.now().In(a.Location)
	day := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, a.Location)
	return day.AddDate(0, 0, -29), day.AddDate(0, 0, 1)
}

func (a *App) accountRows(ctx context.Context, f ports.AccountFilter) ([]AccountRow, map[string]ports.AccountAggregate, error) {
	accs, err := a.Store.Accounts().List(ctx, f)
	if err != nil {
		return nil, nil, err
	}
	from, to := a.last30()
	agg, err := a.Metrics.AccountAggregates(ctx, from, to)
	if err != nil {
		return nil, nil, err
	}
	now := a.now()
	rows := make([]AccountRow, 0, len(accs))
	for _, acc := range accs {
		g := agg[acc.ID]
		owner := ""
		if u, err := a.Store.Users().ByID(ctx, acc.OwnerID); err == nil {
			owner = u.Name
		}
		rows = append(rows, AccountRow{ID: acc.ID, Name: acc.Name, Handle: acc.Handle, Category: acc.Category, Owner: owner, Count: g.Count, Volume: g.Volume,
			FeeBPS: acc.EffectiveFee(now), Negotiated: acc.Negotiated(now), Revenue: g.Fee, Balance: g.Balance, Status: acc.Status})
	}
	return rows, agg, nil
}

func (a *App) AdminDashboard(ctx context.Context, u *d.User) (*AdminDashboard, error) {
	if err := a.requireAdmin(u); err != nil {
		return nil, err
	}
	rows, _, err := a.accountRows(ctx, ports.AccountFilter{})
	if err != nil {
		return nil, err
	}
	from, to := a.last30()
	series, err := a.Metrics.Series(ctx, "", from, to, ports.BucketDay, a.Location)
	if err != nil {
		return nil, err
	}
	bal, err := a.Metrics.BalanceTotal(ctx, "")
	if err != nil {
		return nil, err
	}
	out := &AdminDashboard{Series: series, Balances: bal, Top: []AccountRow{}}
	for _, r := range rows {
		out.Revenue += r.Revenue
		out.Volume += r.Volume
		out.Count += r.Count
		if r.Status == d.AccountActive {
			out.ActiveCount++
		}
		if r.Negotiated {
			out.NegCount++
		}
	}
	if out.Volume > 0 {
		out.AvgFeeBPS = int64(out.Revenue) * 10000 / int64(out.Volume)
	}
	ws, err := a.Store.Withdrawals().List(ctx, ports.WithdrawalFilter{Statuses: []d.WithdrawalStatus{d.WithdrawalRequested, d.WithdrawalApproved}})
	if err != nil {
		return nil, err
	}
	now := a.now()
	for _, w := range ws {
		out.PendingAmount += w.Amount
		out.PendingCount++
		if now.After(w.DueAt) {
			out.OverdueCount++
		}
	}
	sort.SliceStable(rows, func(i, j int) bool { return rows[i].Volume > rows[j].Volume })
	if len(rows) > 5 {
		rows = rows[:5]
	}
	out.Top = append(out.Top, rows...)
	return out, nil
}

func (a *App) AdminAccounts(ctx context.Context, u *d.User, f ports.AccountFilter) ([]AccountRow, error) {
	if err := a.requireAdmin(u); err != nil {
		return nil, err
	}
	if len(f.Query) > 60 {
		f.Query = f.Query[:60]
	}
	rows, _, err := a.accountRows(ctx, f)
	return rows, err
}

type AccountDetail struct {
	AccountRow
	TotalVolume  d.Money       `json:"total_volume"`
	TotalRevenue d.Money       `json:"total_revenue"`
	FeeNote      string        `json:"fee_note"`
	FeeUntil     *time.Time    `json:"fee_until"`
	StatusNote   string        `json:"status_note"`
	OwnerPhone   string        `json:"owner_phone"`
	OwnerEmail   string        `json:"owner_email"`
	Members      []MemberView  `json:"members"`
	CreatedAt    time.Time     `json:"created_at"`
	InviteCode   string        `json:"invite_code"`
	Log          []HistoryItem `json:"log"`
}

func (a *App) AdminAccount(ctx context.Context, u *d.User, id string) (*AccountDetail, error) {
	if err := a.requireAdmin(u); err != nil {
		return nil, err
	}
	acc, err := a.Store.Accounts().ByID(ctx, id)
	if err != nil {
		return nil, err
	}
	from, to := a.last30()
	agg, err := a.Metrics.AccountAggregates(ctx, from, to)
	if err != nil {
		return nil, err
	}
	g := agg[acc.ID]
	now := a.now()
	out := &AccountDetail{
		AccountRow: AccountRow{ID: acc.ID, Name: acc.Name, Handle: acc.Handle, Category: acc.Category, Count: g.Count, Volume: g.Volume, FeeBPS: acc.EffectiveFee(now),
			Negotiated: acc.Negotiated(now), Revenue: g.Fee, Balance: g.Balance, Status: acc.Status},
		TotalVolume: g.Total, TotalRevenue: g.TotalFee, FeeNote: acc.FeeNote, FeeUntil: acc.FeeUntil, StatusNote: acc.StatusNote,
		CreatedAt: acc.CreatedAt, InviteCode: acc.InviteCode, Members: []MemberView{}, Log: []HistoryItem{},
	}
	if owner, err := a.Store.Users().ByID(ctx, acc.OwnerID); err == nil {
		out.Owner, out.OwnerPhone, out.OwnerEmail = owner.Name, d.FormatMZPhone(owner.Phone), owner.Email
	}
	members, err := a.Store.Members().ByAccount(ctx, acc.ID)
	if err != nil {
		return nil, err
	}
	for _, m := range members {
		out.Members = append(out.Members, memberView(m))
	}
	logs, err := a.Store.Audit().List(ctx, acc.ID, "account", 30)
	if err != nil {
		return nil, err
	}
	for _, l := range logs {
		out.Log = append(out.Log, HistoryItem{Text: l.Text, When: l.CreatedAt})
	}
	return out, nil
}

type FeeInput struct {
	Enabled bool   `json:"enabled"`
	FeeBPS  d.BPS  `json:"fee_bps"`
	Months  int    `json:"months"`
	Note    string `json:"note"`
}

func (a *App) AdminSetFee(ctx context.Context, u *d.User, id string, in FeeInput) (*AccountDetail, error) {
	if err := a.requireAdmin(u); err != nil {
		return nil, err
	}
	note := strings.TrimSpace(in.Note)
	if utf8.RuneCountInString(note) > 60 {
		return nil, d.Invalid("note_too_long", "O motivo tem no máximo 60 caracteres.")
	}
	if in.Enabled && !in.FeeBPS.Valid() {
		return nil, d.Invalid("invalid_fee", "A taxa deve estar entre 5% e 20%, em passos de 0,5%.")
	}
	if in.Months != 0 && in.Months != 3 && in.Months != 6 && in.Months != 12 {
		return nil, d.Invalid("invalid_period", "Período inválido.")
	}
	err := a.Store.Tx(ctx, func(s ports.Store) error {
		acc, err := s.Accounts().ByIDForUpdate(ctx, id)
		if err != nil {
			return err
		}
		text := "Taxa reposta para 15%"
		if in.Enabled {
			acc.FeeBPS, acc.FeeNote = in.FeeBPS, note
			acc.FeeUntil = nil
			if in.Months > 0 {
				t := a.now().AddDate(0, in.Months, 0)
				acc.FeeUntil = &t
			}
			text = "Taxa negociada para " + pctLabel(in.FeeBPS)
			if note != "" {
				text += " · " + note
			}
		} else {
			acc.FeeBPS, acc.FeeNote, acc.FeeUntil = d.DefaultFeeBPS, "", nil
		}
		if err := s.Accounts().Update(ctx, acc); err != nil {
			return err
		}
		return a.audit(ctx, s, acc.ID, "account", text)
	})
	if err != nil {
		return nil, err
	}
	return a.AdminAccount(ctx, u, id)
}

func (a *App) AdminSetStatus(ctx context.Context, u *d.User, id string, status d.AccountStatus, note string) (*AccountDetail, error) {
	if err := a.requireAdmin(u); err != nil {
		return nil, err
	}
	if status != d.AccountActive && status != d.AccountSuspended {
		return nil, d.Invalid("invalid_status", "Estado inválido.")
	}
	note = strings.TrimSpace(note)
	if utf8.RuneCountInString(note) > 200 {
		return nil, d.Invalid("note_too_long", "A nota é demasiado longa.")
	}
	err := a.Store.Tx(ctx, func(s ports.Store) error {
		acc, err := s.Accounts().ByIDForUpdate(ctx, id)
		if err != nil {
			return err
		}
		if acc.Status == status {
			return nil
		}
		prev := acc.Status
		acc.Status = status
		text := "Conta suspensa"
		if status == d.AccountSuspended {
			acc.StatusNote = "Suspensa pelo super admin."
			if note != "" {
				acc.StatusNote = note
				text += " · " + note
			}
		} else {
			acc.StatusNote = ""
			text = "Conta reativada"
			if prev == d.AccountPending {
				text = "Conta verificada"
			}
		}
		if err := s.Accounts().Update(ctx, acc); err != nil {
			return err
		}
		return a.audit(ctx, s, acc.ID, "account", text)
	})
	if err != nil {
		return nil, err
	}
	return a.AdminAccount(ctx, u, id)
}

type AdminWithdrawal struct {
	WithdrawalView
	AccountID     string          `json:"account_id"`
	AccountName   string          `json:"account_name"`
	AccountStatus d.AccountStatus `json:"account_status"`
	Balance       d.Money         `json:"balance"`
}

type AdminWithdrawals struct {
	Items   []AdminWithdrawal                `json:"items"`
	Summary map[string]WithdrawalSummaryItem `json:"summary"`
}

type WithdrawalSummaryItem struct {
	Count  int     `json:"count"`
	Amount d.Money `json:"amount"`
}

func (a *App) AdminWithdrawals(ctx context.Context, u *d.User) (*AdminWithdrawals, error) {
	if err := a.requireAdmin(u); err != nil {
		return nil, err
	}
	ws, err := a.Store.Withdrawals().List(ctx, ports.WithdrawalFilter{})
	if err != nil {
		return nil, err
	}
	out := &AdminWithdrawals{Items: []AdminWithdrawal{}, Summary: map[string]WithdrawalSummaryItem{}}
	accs := map[string]*d.Account{}
	mems := map[string]*d.Member{}
	for _, w := range ws {
		acc, ok := accs[w.AccountID]
		if !ok {
			if acc, err = a.Store.Accounts().ByID(ctx, w.AccountID); err != nil {
				return nil, err
			}
			accs[w.AccountID] = acc
		}
		m, ok := mems[w.MemberID]
		if !ok {
			if m, err = a.Store.Members().ByID(ctx, w.MemberID); err != nil {
				m = &d.Member{Name: "Membro removido"}
			}
			mems[w.MemberID] = m
		}
		out.Items = append(out.Items, AdminWithdrawal{WithdrawalView: withdrawalView(w, m.Name, true), AccountID: acc.ID, AccountName: acc.Name, AccountStatus: acc.Status, Balance: m.Balance})
		key := string(w.Status)
		it := out.Summary[key]
		it.Count++
		it.Amount += w.Amount
		out.Summary[key] = it
		if w.Status.Open() {
			o := out.Summary["open"]
			o.Count++
			o.Amount += w.Amount
			out.Summary["open"] = o
		}
	}
	return out, nil
}

var transferRefRe = regexp.MustCompile(`^[A-Z0-9\-]{4,40}$`)

var RejectReasons = []string{"NIB inválido", "Conta por verificar", "Conta suspensa", "Suspeita de fraude"}

type WithdrawalAction struct {
	Action string `json:"action"`
	Ref    string `json:"ref"`
	Reason string `json:"reason"`
	Note   string `json:"note"`
}

func (a *App) AdminActOnWithdrawal(ctx context.Context, u *d.User, id string, in WithdrawalAction) error {
	if err := a.requireAdmin(u); err != nil {
		return err
	}
	var notice *ports.Message
	err := a.Store.Tx(ctx, func(s ports.Store) error {
		w, err := s.Withdrawals().ByIDForUpdate(ctx, id)
		if err != nil {
			return err
		}
		acc, err := s.Accounts().ByID(ctx, w.AccountID)
		if err != nil {
			return err
		}
		now := a.now()
		switch in.Action {
		case "approve":
			if w.Status != d.WithdrawalRequested {
				return d.ErrStaleState
			}
			if acc.Status != d.AccountActive {
				return d.Forbidden("account_not_active", "A conta não está ativa. Verifica ou reativa antes de aprovar.")
			}
			w.Status, w.DecidedAt, w.DecidedBy = d.WithdrawalApproved, &now, u.ID
		case "complete":
			if w.Status != d.WithdrawalApproved {
				return d.ErrStaleState
			}
			ref := strings.ToUpper(strings.TrimSpace(in.Ref))
			if !transferRefRe.MatchString(ref) {
				return d.Invalid("invalid_ref", "Referência da transferência inválida.")
			}
			w.Status, w.TransferRef, w.PaidAt = d.WithdrawalPaid, ref, &now
			notice = &ports.Message{Kind: "withdrawal_paid", Body: "Levantamento " + w.Code() + " pago: " + fmtMT(w.Amount) + ". Ref. " + ref}
		case "reject":
			if !w.Status.Open() {
				return d.ErrStaleState
			}
			valid := false
			for _, r := range RejectReasons {
				if r == in.Reason {
					valid = true
				}
			}
			if !valid {
				return d.Invalid("invalid_reason", "Escolhe o motivo.")
			}
			note := strings.TrimSpace(in.Note)
			if utf8.RuneCountInString(note) > 280 {
				return d.Invalid("note_too_long", "A mensagem é demasiado longa.")
			}
			members, err := s.Members().LockByAccount(ctx, w.AccountID)
			if err != nil {
				return err
			}
			var m *d.Member
			for i := range members {
				if members[i].ID == w.MemberID {
					m = &members[i]
				}
			}
			if m == nil {
				return d.Conflict("member_missing", "O membro já não existe.")
			}
			m.Balance += w.Amount
			if err := s.Members().Update(ctx, m); err != nil {
				return err
			}
			if err := s.Ledger().Add(ctx, ports.LedgerEntry{MemberID: m.ID, AccountID: w.AccountID, Kind: "withdrawal_refund", Amount: w.Amount, Ref: w.ID, CreatedAt: now}); err != nil {
				return err
			}
			w.Status, w.Reason, w.Note, w.DecidedAt, w.DecidedBy = d.WithdrawalRejected, in.Reason, note, &now, u.ID
			notice = &ports.Message{Kind: "withdrawal_rejected", Body: "Levantamento " + w.Code() + " rejeitado: " + in.Reason + ". O valor voltou ao teu saldo."}
		default:
			return d.Invalid("invalid_action", "Ação inválida.")
		}
		if notice != nil {
			if m, err := s.Members().ByID(ctx, w.MemberID); err == nil {
				notice.To = d.FormatMZPhone(m.WalletNumber)
			}
		}
		return s.Withdrawals().Update(ctx, w)
	})
	if err == nil && notice != nil && notice.To != "" {
		a.notify(ctx, *notice)
	}
	return err
}

type InviteRow struct {
	Code      string         `json:"code"`
	Note      string         `json:"note"`
	MaxUses   int            `json:"max_uses"`
	Uses      int            `json:"uses"`
	UsedBy    []string       `json:"used_by"`
	FeeBPS    d.BPS          `json:"fee_bps"`
	Status    d.InviteStatus `json:"status"`
	CreatedAt time.Time      `json:"created_at"`
}

func (a *App) AdminInvites(ctx context.Context, u *d.User) ([]InviteRow, error) {
	if err := a.requireAdmin(u); err != nil {
		return nil, err
	}
	invs, err := a.Store.Invites().List(ctx)
	if err != nil {
		return nil, err
	}
	codes := make([]string, len(invs))
	for i, inv := range invs {
		codes[i] = inv.Code
	}
	used, err := a.Store.Accounts().ByInviteCodes(ctx, codes)
	if err != nil {
		return nil, err
	}
	out := make([]InviteRow, 0, len(invs))
	for _, inv := range invs {
		by := used[inv.Code]
		if by == nil {
			by = []string{}
		}
		out = append(out, InviteRow{Code: inv.Code, Note: inv.Note, MaxUses: inv.MaxUses, Uses: inv.Uses, UsedBy: by, FeeBPS: inv.FeeBPS, Status: inv.Status, CreatedAt: inv.CreatedAt})
	}
	return out, nil
}

type InviteInput struct {
	Note    string `json:"note"`
	MaxUses int    `json:"max_uses"`
	FeeBPS  d.BPS  `json:"fee_bps"`
}

func codeBase(note string) string {
	t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	s, _, _ := transform.String(t, note)
	var b strings.Builder
	for _, r := range strings.ToUpper(s) {
		if r >= 'A' && r <= 'Z' && b.Len() < 7 {
			b.WriteRune(r)
		}
	}
	if b.Len() == 0 {
		return "REFRES"
	}
	return b.String()
}

func (a *App) AdminCreateInvite(ctx context.Context, u *d.User, in InviteInput) (*InviteRow, error) {
	if err := a.requireAdmin(u); err != nil {
		return nil, err
	}
	note := strings.TrimSpace(in.Note)
	if utf8.RuneCountInString(note) > 40 {
		return nil, d.Invalid("note_too_long", "A nota tem no máximo 40 caracteres.")
	}
	if in.MaxUses != 1 && in.MaxUses != 5 && in.MaxUses != 10 && in.MaxUses != 25 {
		return nil, d.Invalid("invalid_uses", "Número de utilizações inválido.")
	}
	if !in.FeeBPS.Valid() {
		return nil, d.Invalid("invalid_fee", "Taxa inválida.")
	}
	if note == "" {
		note = "Sem nota"
	}
	base := codeBase(in.Note)
	for i := 0; i < 5; i++ {
		inv := &d.Invite{Code: base + "-" + randCode(10, "23456789ABCDEFGHJKMNPQRSTUVWXYZ"), Note: note, MaxUses: in.MaxUses, FeeBPS: in.FeeBPS, Status: d.InviteActive, CreatedAt: a.now()}
		err := a.Store.Invites().Create(ctx, inv)
		if err == nil {
			return &InviteRow{Code: inv.Code, Note: inv.Note, MaxUses: inv.MaxUses, UsedBy: []string{}, FeeBPS: inv.FeeBPS, Status: inv.Status, CreatedAt: inv.CreatedAt}, nil
		}
		var de *d.Error
		if !errors.As(err, &de) || de.Kind != d.KindConflict {
			return nil, err
		}
	}
	return nil, d.Conflict("code_collision", "Tenta de novo.")
}

func (a *App) AdminRevokeInvite(ctx context.Context, u *d.User, code string) error {
	if err := a.requireAdmin(u); err != nil {
		return err
	}
	return a.Store.Tx(ctx, func(s ports.Store) error {
		inv, err := s.Invites().ByCodeForUpdate(ctx, normCode(code))
		if err != nil {
			return err
		}
		inv.Status = d.InviteRevoked
		return s.Invites().Update(ctx, inv)
	})
}
