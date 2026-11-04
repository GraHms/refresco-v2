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

type MemberView struct {
	ID       string    `json:"id"`
	Name     string    `json:"name"`
	Role     string    `json:"role"`
	Pct      int       `json:"pct"`
	Wallet   d.Channel `json:"wallet"`
	Number   string    `json:"number"`
	IsAdmin  bool      `json:"is_admin"`
	Invited  bool      `json:"invited"`
	Balance  d.Money   `json:"balance"`
	Approved *bool     `json:"approved,omitempty"`
}

type ProposalView struct {
	ID        string        `json:"id"`
	Splits    []d.MemberPct `json:"splits"`
	Waiting   []string      `json:"waiting"`
	CreatedAt time.Time     `json:"created_at"`
}

type HistoryItem struct {
	Text string    `json:"text"`
	When time.Time `json:"when"`
}

type GroupView struct {
	AutoSplit bool          `json:"auto_split"`
	FeeBPS    d.BPS         `json:"fee_bps"`
	Members   []MemberView  `json:"members"`
	Proposal  *ProposalView `json:"proposal"`
	History   []HistoryItem `json:"history"`
	CanEdit   bool          `json:"can_edit"`
}

func memberView(m d.Member) MemberView {
	return MemberView{ID: m.ID, Name: m.Name, Role: m.Role, Pct: m.Pct, Wallet: m.Wallet, Number: d.FormatMZPhone(m.WalletNumber),
		IsAdmin: m.IsAdmin, Invited: m.Status == d.MemberInvited, Balance: m.Balance}
}

func (a *App) Group(ctx context.Context, c *Creator) (*GroupView, error) {
	members, err := a.Store.Members().ByAccount(ctx, c.Account.ID)
	if err != nil {
		return nil, err
	}
	g := &GroupView{AutoSplit: c.Account.AutoSplit, FeeBPS: c.Account.EffectiveFee(a.now()), CanEdit: c.Member.IsAdmin, Members: []MemberView{}, History: []HistoryItem{}}
	names := map[string]string{}
	for _, m := range members {
		g.Members = append(g.Members, memberView(m))
		names[m.ID] = m.Name
	}
	p, err := a.Store.Proposals().PendingByAccount(ctx, c.Account.ID)
	if err == nil {
		apps, err := a.Store.Proposals().Approvals(ctx, p.ID)
		if err != nil {
			return nil, err
		}
		pv := &ProposalView{ID: p.ID, Splits: p.Splits, CreatedAt: p.CreatedAt, Waiting: []string{}}
		for _, ap := range apps {
			if ap.ApprovedAt == nil {
				pv.Waiting = append(pv.Waiting, names[ap.MemberID])
			}
		}
		g.Proposal = pv
	} else if !errors.Is(err, d.ErrNotFound) {
		return nil, err
	}
	hist, err := a.Store.Audit().List(ctx, c.Account.ID, "group", 20)
	if err != nil {
		return nil, err
	}
	for _, h := range hist {
		g.History = append(g.History, HistoryItem{Text: h.Text, When: h.CreatedAt})
	}
	return g, nil
}

func (a *App) SetAutoSplit(ctx context.Context, c *Creator, on bool) error {
	if err := a.adminOnly(c); err != nil {
		return err
	}
	return a.Store.Tx(ctx, func(s ports.Store) error {
		acc, err := s.Accounts().ByIDForUpdate(ctx, c.Account.ID)
		if err != nil {
			return err
		}
		if acc.AutoSplit == on {
			return nil
		}
		acc.AutoSplit = on
		if err := s.Accounts().Update(ctx, acc); err != nil {
			return err
		}
		text := c.Member.Name + " pausou a divisão automática."
		if on {
			text = c.Member.Name + " reativou a divisão automática."
		}
		return a.audit(ctx, s, acc.ID, "group", text)
	})
}

type InviteMemberInput struct {
	Name   string    `json:"name"`
	Role   string    `json:"role"`
	Wallet d.Channel `json:"wallet"`
	Phone  string    `json:"phone"`
}

func (a *App) InviteMember(ctx context.Context, c *Creator, in InviteMemberInput) (*MemberView, error) {
	if err := a.adminOnly(c); err != nil {
		return nil, err
	}
	name, role := strings.TrimSpace(in.Name), strings.TrimSpace(in.Role)
	if utf8.RuneCountInString(name) < 2 || utf8.RuneCountInString(name) > 40 || utf8.RuneCountInString(role) > 40 {
		return nil, d.Invalid("invalid_member", "Confirma o nome e a função.")
	}
	if role == "" {
		role = "Membro"
	}
	phone := d.NormalizeMZPhone(in.Phone)
	if !in.Wallet.IsWallet() || phone == "" || d.WalletOf(phone) != in.Wallet {
		return nil, d.Invalid("wallet_mismatch", "O número não corresponde à carteira escolhida.")
	}
	m := &d.Member{ID: newID(), AccountID: c.Account.ID, Name: name, Role: role, Wallet: in.Wallet, WalletNumber: phone, Status: d.MemberInvited, CreatedAt: a.now()}
	err := a.Store.Tx(ctx, func(s ports.Store) error {
		existing, err := s.Members().LockByAccount(ctx, c.Account.ID)
		if err != nil {
			return err
		}
		if len(existing) >= 12 {
			return d.Invalid("too_many_members", "Um grupo tem no máximo 12 membros.")
		}
		if err := s.Members().Create(ctx, m); err != nil {
			return err
		}
		if err := s.PayoutAccounts().Create(ctx, &d.PayoutAccount{ID: newID(), MemberID: m.ID, Kind: d.PayoutKind(in.Wallet), Number: phone, IsMain: true, CreatedAt: a.now()}); err != nil {
			return err
		}
		return a.audit(ctx, s, c.Account.ID, "group", name+" foi convidado para o grupo.")
	})
	if err != nil {
		return nil, err
	}
	a.notify(ctx, ports.Message{To: d.FormatMZPhone(phone), Kind: "group_invite", Body: c.Account.Name + " convidou-te para o grupo na refres.co. Os teus ganhos ficam no teu saldo."})
	v := memberView(*m)
	return &v, nil
}

func (a *App) RemoveMember(ctx context.Context, c *Creator, memberID string) error {
	if err := a.adminOnly(c); err != nil {
		return err
	}
	return a.Store.Tx(ctx, func(s ports.Store) error {
		members, err := s.Members().LockByAccount(ctx, c.Account.ID)
		if err != nil {
			return err
		}
		var target *d.Member
		for i := range members {
			if members[i].ID == memberID {
				target = &members[i]
			}
		}
		if target == nil {
			return d.ErrNotFound
		}
		if target.IsAdmin {
			return d.Invalid("cannot_remove_admin", "O admin não pode ser removido.")
		}
		if target.Pct != 0 {
			return d.Invalid("member_has_share", "Põe a percentagem deste membro a 0% e aprova a nova divisão antes de o remover.")
		}
		if target.Balance != 0 {
			return d.Invalid("member_has_balance", "Este membro ainda tem saldo por levantar.")
		}
		if _, err := s.Proposals().PendingByAccount(ctx, c.Account.ID); err == nil {
			return d.Conflict("proposal_pending", "Há uma divisão à espera de aprovação.")
		}
		open, err := s.Withdrawals().List(ctx, ports.WithdrawalFilter{MemberID: memberID, Statuses: []d.WithdrawalStatus{d.WithdrawalRequested, d.WithdrawalApproved}})
		if err != nil {
			return err
		}
		if len(open) > 0 {
			return d.Conflict("member_has_withdrawals", "Este membro tem levantamentos em curso.")
		}
		if err := s.Members().Delete(ctx, memberID); err != nil {
			return err
		}
		return a.audit(ctx, s, c.Account.ID, "group", target.Name+" saiu do grupo.")
	})
}

func splitText(members []d.Member, splits []d.MemberPct) string {
	by := map[string]int{}
	for _, s := range splits {
		by[s.MemberID] = s.Pct
	}
	parts := []string{}
	for _, m := range members {
		parts = append(parts, itoa(by[m.ID]))
	}
	return strings.Join(parts, " / ")
}

func (a *App) ProposeSplit(ctx context.Context, c *Creator, splits []d.MemberPct) (*GroupView, error) {
	if err := a.adminOnly(c); err != nil {
		return nil, err
	}
	type pending struct {
		phone, token string
	}
	var notes []pending
	err := a.Store.Tx(ctx, func(s ports.Store) error {
		members, err := s.Members().LockByAccount(ctx, c.Account.ID)
		if err != nil {
			return err
		}
		if err := d.ValidateSplits(members, splits); err != nil {
			return err
		}
		changed := false
		by := map[string]int{}
		for _, sp := range splits {
			by[sp.MemberID] = sp.Pct
		}
		for _, m := range members {
			if by[m.ID] != m.Pct {
				changed = true
			}
		}
		if !changed {
			return d.Invalid("split_unchanged", "A divisão não mudou.")
		}
		if prev, err := s.Proposals().PendingByAccount(ctx, c.Account.ID); err == nil {
			prev.Status = d.ProposalCancelled
			if err := s.Proposals().Update(ctx, prev); err != nil {
				return err
			}
		}
		p := &d.SplitProposal{ID: newID(), AccountID: c.Account.ID, Splits: splits, Status: d.ProposalPending, CreatedBy: c.Member.ID, CreatedAt: a.now()}
		var approvals []d.SplitApproval
		for _, m := range members {
			if m.IsAdmin {
				continue
			}
			tok := randToken(24)
			approvals = append(approvals, d.SplitApproval{ProposalID: p.ID, MemberID: m.ID, TokenHash: a.mac("split-approval", tok)})
			notes = append(notes, pending{phone: m.WalletNumber, token: tok})
		}
		if len(approvals) == 0 {
			p.Status = d.ProposalApplied
			if err := s.Proposals().Create(ctx, p, nil); err != nil {
				return err
			}
			return a.applySplit(ctx, s, c.Account.ID, members, p)
		}
		if err := s.Proposals().Create(ctx, p, approvals); err != nil {
			return err
		}
		return a.audit(ctx, s, c.Account.ID, "group", c.Member.Name+" propôs a divisão "+splitText(members, splits)+".")
	})
	if err != nil {
		return nil, err
	}
	for _, n := range notes {
		a.notify(ctx, ports.Message{To: d.FormatMZPhone(n.phone), Kind: "split_approval", Body: c.Account.Name + " propôs uma nova divisão. Aprova em " + a.PublicURL + "/aprovar#" + n.token})
	}
	return a.Group(ctx, c)
}

func (a *App) applySplit(ctx context.Context, s ports.Store, accountID string, members []d.Member, p *d.SplitProposal) error {
	by := map[string]int{}
	for _, sp := range p.Splits {
		by[sp.MemberID] = sp.Pct
	}
	for i := range members {
		members[i].Pct = by[members[i].ID]
		if err := s.Members().Update(ctx, &members[i]); err != nil {
			return err
		}
	}
	return a.audit(ctx, s, accountID, "group", "Divisão alterada para "+splitText(members, p.Splits)+". Aprovado por todos.")
}

type ApprovalView struct {
	Account  string       `json:"account"`
	Member   string       `json:"member"`
	Splits   []MemberView `json:"splits"`
	Approved bool         `json:"approved"`
	Applied  bool         `json:"applied"`
}

func (a *App) approvalByToken(ctx context.Context, s ports.Store, token string) (*d.SplitApproval, *d.SplitProposal, error) {
	ap, err := s.Proposals().ApprovalByToken(ctx, a.mac("split-approval", token))
	if err != nil {
		return nil, nil, d.ErrNotFound
	}
	p, err := s.Proposals().ByID(ctx, ap.ProposalID)
	if err != nil || p.Status != d.ProposalPending {
		return nil, nil, d.NotFound("proposal_closed", "Esta proposta já não está ativa.")
	}
	return ap, p, nil
}

func (a *App) SplitApproval(ctx context.Context, token string) (*ApprovalView, error) {
	ap, p, err := a.approvalByToken(ctx, a.Store, token)
	if err != nil {
		return nil, err
	}
	acc, err := a.Store.Accounts().ByID(ctx, p.AccountID)
	if err != nil {
		return nil, err
	}
	members, err := a.Store.Members().ByAccount(ctx, p.AccountID)
	if err != nil {
		return nil, err
	}
	by := map[string]int{}
	for _, sp := range p.Splits {
		by[sp.MemberID] = sp.Pct
	}
	v := &ApprovalView{Account: acc.Name, Approved: ap.ApprovedAt != nil}
	for _, m := range members {
		if m.ID == ap.MemberID {
			v.Member = m.Name
		}
		mv := memberView(m)
		mv.Pct, mv.Number, mv.Balance = by[m.ID], "", 0
		v.Splits = append(v.Splits, mv)
	}
	return v, nil
}

func (a *App) ApproveSplit(ctx context.Context, token string) (*ApprovalView, error) {
	applied := false
	err := a.Store.Tx(ctx, func(s ports.Store) error {
		ap, p, err := a.approvalByToken(ctx, s, token)
		if err != nil {
			return err
		}
		members, err := s.Members().LockByAccount(ctx, p.AccountID)
		if err != nil {
			return err
		}
		if ap.ApprovedAt == nil {
			if err := s.Proposals().Approve(ctx, p.ID, ap.MemberID, a.now()); err != nil {
				return err
			}
		}
		apps, err := s.Proposals().Approvals(ctx, p.ID)
		if err != nil {
			return err
		}
		for _, x := range apps {
			if x.ApprovedAt == nil && x.MemberID != ap.MemberID {
				return nil
			}
		}
		if err := d.ValidateSplits(members, p.Splits); err != nil {
			p.Status = d.ProposalCancelled
			return s.Proposals().Update(ctx, p)
		}
		p.Status = d.ProposalApplied
		if err := s.Proposals().Update(ctx, p); err != nil {
			return err
		}
		applied = true
		return a.applySplit(ctx, s, p.AccountID, members, p)
	})
	if err != nil {
		return nil, err
	}
	v := &ApprovalView{Approved: true, Applied: applied}
	return v, nil
}

func (a *App) CancelProposal(ctx context.Context, c *Creator) error {
	if err := a.adminOnly(c); err != nil {
		return err
	}
	return a.Store.Tx(ctx, func(s ports.Store) error {
		p, err := s.Proposals().PendingByAccount(ctx, c.Account.ID)
		if err != nil {
			return err
		}
		p.Status = d.ProposalCancelled
		if err := s.Proposals().Update(ctx, p); err != nil {
			return err
		}
		return a.audit(ctx, s, c.Account.ID, "group", c.Member.Name+" cancelou a proposta de divisão.")
	})
}
