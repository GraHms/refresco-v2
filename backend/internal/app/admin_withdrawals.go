package app

import (
	"context"
	"sort"
	"time"

	d "github.com/refresco/refresco-v2/backend/internal/domain"
	"github.com/refresco/refresco-v2/backend/internal/ports"
)

type AdminWithdrawalDetail struct {
	AdminWithdrawal
	MemberRole    string           `json:"member_role"`
	MemberPhone   string           `json:"member_phone"`
	MemberPct     int              `json:"member_pct"`
	AccountHandle string           `json:"account_handle"`
	OwnerName     string           `json:"owner_name"`
	OwnerEmail    string           `json:"owner_email"`
	OwnerPhone    string           `json:"owner_phone"`
	Recent        []WithdrawalView `json:"recent"`
	PaidTotal     d.Money          `json:"paid_total"`
	DestAddedAt   *time.Time       `json:"dest_added_at,omitempty"`
	DestFirstUse  bool             `json:"dest_first_use"`
}

func (a *App) memberOrGhost(ctx context.Context, id string) *d.Member {
	m, err := a.Store.Members().ByID(ctx, id)
	if err != nil {
		return &d.Member{Name: "Membro removido"}
	}
	return m
}

func (a *App) AdminWithdrawal(ctx context.Context, u *d.User, id string) (*AdminWithdrawalDetail, error) {
	if err := a.requireAdmin(u); err != nil {
		return nil, err
	}
	w, err := a.Store.Withdrawals().ByID(ctx, id)
	if err != nil {
		return nil, err
	}
	acc, err := a.Store.Accounts().ByID(ctx, w.AccountID)
	if err != nil {
		return nil, err
	}
	m := a.memberOrGhost(ctx, w.MemberID)
	out := &AdminWithdrawalDetail{
		AdminWithdrawal: AdminWithdrawal{WithdrawalView: withdrawalView(*w, m.Name, true), AccountID: acc.ID, AccountName: acc.Name, AccountStatus: acc.Status, Balance: m.Balance},
		MemberRole:      m.Role, MemberPhone: d.FormatMZPhone(m.WalletNumber), MemberPct: m.Pct, AccountHandle: acc.Handle, Recent: []WithdrawalView{},
	}
	if owner, err := a.Store.Users().ByID(ctx, acc.OwnerID); err == nil {
		out.OwnerName, out.OwnerEmail, out.OwnerPhone = owner.Name, owner.Email, d.FormatMZPhone(owner.Phone)
	}
	prev, err := a.Store.Withdrawals().List(ctx, ports.WithdrawalFilter{MemberID: w.MemberID})
	if err != nil {
		return nil, err
	}
	if accts, err := a.Store.PayoutAccounts().ByMember(ctx, w.MemberID); err == nil {
		for _, p := range accts {
			if p.Kind == w.DestKind && p.Number == w.DestNumber {
				t := p.CreatedAt
				out.DestAddedAt = &t
			}
		}
	}
	out.DestFirstUse = true
	for _, x := range prev {
		if x.Status == d.WithdrawalPaid {
			out.PaidTotal += x.Amount
			if x.DestKind == w.DestKind && x.DestNumber == w.DestNumber {
				out.DestFirstUse = false
			}
		}
		if x.ID != w.ID && len(out.Recent) < 5 {
			out.Recent = append(out.Recent, withdrawalView(x, m.Name, false))
		}
	}
	return out, nil
}

func (a *App) paymentOrder(ctx context.Context, w d.Withdrawal) (ports.PaymentOrder, error) {
	acc, err := a.Store.Accounts().ByID(ctx, w.AccountID)
	if err != nil {
		return ports.PaymentOrder{}, err
	}
	m := a.memberOrGhost(ctx, w.MemberID)
	o := ports.PaymentOrder{Code: w.Code(), Status: w.Status, Amount: w.Amount, Beneficiary: w.DestHolder, MemberName: m.Name, MemberRole: m.Role,
		MemberPhone: m.WalletNumber, AccountName: acc.Name, AccountHandle: acc.Handle, AccountStatus: acc.Status, DestKind: w.DestKind,
		DestNumber: w.DestNumber, DestBank: w.DestBank, RequestedAt: w.CreatedAt, DueAt: w.DueAt, ApprovedAt: w.DecidedAt, PaidAt: w.PaidAt,
		TransferRef: w.TransferRef, Reason: w.Reason}
	if w.DestKind != d.PayoutNIB && o.Beneficiary == "" {
		o.Beneficiary = m.Name
	}
	return o, nil
}

// AdminPaymentOrdersPDF renders one withdrawal (id) or every withdrawal in a status as a batch.
func (a *App) AdminPaymentOrdersPDF(ctx context.Context, u *d.User, id string, status d.WithdrawalStatus) ([]byte, string, error) {
	if err := a.requireAdmin(u); err != nil {
		return nil, "", err
	}
	if a.Docs == nil {
		return nil, "", d.Unavailable("pdf_unavailable", "Geração de PDF indisponível.")
	}
	var ws []d.Withdrawal
	name := ""
	if id != "" {
		w, err := a.Store.Withdrawals().ByID(ctx, id)
		if err != nil {
			return nil, "", err
		}
		ws, name = []d.Withdrawal{*w}, w.Code()
	} else {
		if status != d.WithdrawalApproved && status != d.WithdrawalRequested {
			return nil, "", d.Invalid("invalid_status", "Só há lotes de pedidos ou de aprovados.")
		}
		list, err := a.Store.Withdrawals().List(ctx, ports.WithdrawalFilter{Statuses: []d.WithdrawalStatus{status}})
		if err != nil {
			return nil, "", err
		}
		if len(list) == 0 {
			return nil, "", d.NotFound("empty_batch", "Não há levantamentos neste estado.")
		}
		sort.SliceStable(list, func(i, j int) bool { return list[i].DueAt.Before(list[j].DueAt) })
		ws, name = list, "lote-"+string(status)+"-"+a.now().In(a.Location).Format("2006-01-02")
	}
	orders := make([]ports.PaymentOrder, 0, len(ws))
	for _, w := range ws {
		o, err := a.paymentOrder(ctx, w)
		if err != nil {
			return nil, "", err
		}
		orders = append(orders, o)
	}
	b, err := a.Docs.PaymentOrders(orders, u.Email, a.now())
	if err != nil {
		return nil, "", err
	}
	a.Log.Info("payment orders exported", "admin", u.ID, "count", len(orders), "file", name)
	return b, name + ".pdf", nil
}
