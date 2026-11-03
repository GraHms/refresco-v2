package ports

import (
	"context"
	"net/http"
	"time"

	d "github.com/refresco/refresco-v2/backend/internal/domain"
)

type Store interface {
	Tx(ctx context.Context, fn func(Store) error) error
	Users() UserRepo
	Accounts() AccountRepo
	Members() MemberRepo
	PayoutAccounts() PayoutAccountRepo
	Supports() SupportRepo
	Withdrawals() WithdrawalRepo
	Invites() InviteRepo
	Proposals() ProposalRepo
	Audit() AuditRepo
	Resets() ResetRepo
	Waitlist() WaitlistRepo
	Ledger() LedgerRepo
	Stats() StatsWriter
}

type UserRepo interface {
	Create(ctx context.Context, u *d.User) error
	ByID(ctx context.Context, id string) (*d.User, error)
	ByEmail(ctx context.Context, email string) (*d.User, error)
	ByPhone(ctx context.Context, phone string) (*d.User, error)
	Update(ctx context.Context, u *d.User) error
}

type AccountFilter struct {
	Query      string
	Status     d.AccountStatus
	Negotiated bool
}

type AccountRepo interface {
	Create(ctx context.Context, a *d.Account) error
	ByID(ctx context.Context, id string) (*d.Account, error)
	ByIDForUpdate(ctx context.Context, id string) (*d.Account, error)
	ByHandle(ctx context.Context, handle string) (*d.Account, error)
	HandleTaken(ctx context.Context, handle, exceptID string) (bool, error)
	Update(ctx context.Context, a *d.Account) error
	List(ctx context.Context, f AccountFilter) ([]d.Account, error)
	ByInviteCodes(ctx context.Context, codes []string) (map[string][]string, error)
}

type MemberRepo interface {
	Create(ctx context.Context, m *d.Member) error
	ByID(ctx context.Context, id string) (*d.Member, error)
	ByAccount(ctx context.Context, accountID string) ([]d.Member, error)
	LockByAccount(ctx context.Context, accountID string) ([]d.Member, error)
	ByUser(ctx context.Context, userID string) (*d.Member, error)
	Update(ctx context.Context, m *d.Member) error
	Delete(ctx context.Context, id string) error
}

type PayoutAccountRepo interface {
	Create(ctx context.Context, p *d.PayoutAccount) error
	ByID(ctx context.Context, id string) (*d.PayoutAccount, error)
	ByMember(ctx context.Context, memberID string) ([]d.PayoutAccount, error)
	Delete(ctx context.Context, id string) error
	ClearMain(ctx context.Context, memberID string) error
}

type SupportFilter struct {
	Query   string
	Channel d.Channel
	Status  d.SupportStatus
	Limit   int
	Offset  int
}

type SupportRepo interface {
	Create(ctx context.Context, s *d.Support) error
	ByID(ctx context.Context, id string) (*d.Support, error)
	ByIDForUpdate(ctx context.Context, id string) (*d.Support, error)
	ByProviderRef(ctx context.Context, ref string) (*d.Support, error)
	ByIdemKey(ctx context.Context, accountID, key string) (*d.Support, error)
	Update(ctx context.Context, s *d.Support) error
	List(ctx context.Context, accountID string, f SupportFilter) ([]d.Support, int64, error)
	ChannelCounts(ctx context.Context, accountID string) (map[d.Channel]int64, error)
	CountUnread(ctx context.Context, accountID string) (int64, error)
	SetRead(ctx context.Context, id string, at *time.Time) error
	AddSplits(ctx context.Context, splits []d.SupportSplit) error
	Splits(ctx context.Context, supportID string) ([]d.SupportSplit, error)
	ExpireStale(ctx context.Context, before time.Time) (int64, error)
}

type WithdrawalFilter struct {
	AccountID string
	MemberID  string
	Statuses  []d.WithdrawalStatus
}

type WithdrawalRepo interface {
	Create(ctx context.Context, w *d.Withdrawal) error
	ByID(ctx context.Context, id string) (*d.Withdrawal, error)
	ByIDForUpdate(ctx context.Context, id string) (*d.Withdrawal, error)
	Update(ctx context.Context, w *d.Withdrawal) error
	List(ctx context.Context, f WithdrawalFilter) ([]d.Withdrawal, error)
}

type InviteRepo interface {
	Create(ctx context.Context, i *d.Invite) error
	ByCode(ctx context.Context, code string) (*d.Invite, error)
	ByCodeForUpdate(ctx context.Context, code string) (*d.Invite, error)
	Update(ctx context.Context, i *d.Invite) error
	List(ctx context.Context) ([]d.Invite, error)
}

type ProposalRepo interface {
	Create(ctx context.Context, p *d.SplitProposal, approvals []d.SplitApproval) error
	PendingByAccount(ctx context.Context, accountID string) (*d.SplitProposal, error)
	ByID(ctx context.Context, id string) (*d.SplitProposal, error)
	Update(ctx context.Context, p *d.SplitProposal) error
	Approvals(ctx context.Context, proposalID string) ([]d.SplitApproval, error)
	ApprovalByToken(ctx context.Context, tokenHash string) (*d.SplitApproval, error)
	Approve(ctx context.Context, proposalID, memberID string, at time.Time) error
}

type AuditRepo interface {
	Add(ctx context.Context, e *d.AuditEntry) error
	List(ctx context.Context, accountID, scope string, limit int) ([]d.AuditEntry, error)
}

type ResetRepo interface {
	Create(ctx context.Context, r *d.PasswordReset) error
	Latest(ctx context.Context, userID string) (*d.PasswordReset, error)
	ByTokenHash(ctx context.Context, hash string) (*d.PasswordReset, error)
	Update(ctx context.Context, r *d.PasswordReset) error
}

type WaitlistRepo interface {
	Join(ctx context.Context, email string) (int64, error)
}

type LedgerEntry struct {
	MemberID  string
	AccountID string
	Kind      string
	Amount    d.Money
	Ref       string
	CreatedAt time.Time
}

type LedgerRepo interface {
	Add(ctx context.Context, e LedgerEntry) error
}

// StatsWriter maintains the pre-aggregated daily rollups that every metric reads from.
type StatsWriter interface {
	RecordSupport(ctx context.Context, accountID string, at time.Time, ch d.Channel, amount, fee d.Money) error
}

type Bucket string

const (
	BucketDay   Bucket = "day"
	BucketMonth Bucket = "month"
)

type ChannelTotals struct {
	Mpesa d.Money `json:"mpesa"`
	Emola d.Money `json:"emola"`
	Card  d.Money `json:"card"`
}

func (c ChannelTotals) Sum() d.Money { return c.Mpesa + c.Emola + c.Card }

type SeriesPoint struct {
	Start  time.Time     `json:"start"`
	Total  d.Money       `json:"total"`
	Fee    d.Money       `json:"fee"`
	ByChan ChannelTotals `json:"by_channel"`
	Count  int64         `json:"count"`
}

type PeriodTotals struct {
	Gross     d.Money
	Fee       d.Money
	Count     int64
	Supporter int64
	ByChan    ChannelTotals
	TopAmount d.Money
}

type AccountAggregate struct {
	AccountID string
	Count     int64
	Volume    d.Money
	Fee       d.Money
	Total     d.Money
	TotalFee  d.Money
	Balance   d.Money
}

type MetricsReader interface {
	Series(ctx context.Context, accountID string, from, to time.Time, b Bucket, loc *time.Location) ([]SeriesPoint, error)
	Totals(ctx context.Context, accountID string, from, to time.Time) (PeriodTotals, error)
	AccountAggregates(ctx context.Context, from, to time.Time) (map[string]AccountAggregate, error)
	BalanceTotal(ctx context.Context, accountID string) (d.Money, error)
}

type PaymentRequest struct {
	Ref       string
	Amount    d.Money
	Channel   d.Channel
	Phone     string
	CardToken string
	Merchant  string
}

type PaymentInit struct {
	ProviderRef string
	NextAction  string
	ExpiresAt   time.Time
}

type PaymentState struct {
	Status     d.SupportStatus
	FailReason string
}

type PaymentGateway interface {
	Name() string
	Initiate(ctx context.Context, req PaymentRequest) (PaymentInit, error)
	Status(ctx context.Context, providerRef string) (PaymentState, error)
	SubmitOTP(ctx context.Context, providerRef, code string) (PaymentState, error)
	ParseWebhook(h http.Header, body []byte) (providerRef string, st PaymentState, err error)
}

// PaymentSimulator is implemented only by sandbox gateways.
type PaymentSimulator interface {
	Simulate(ctx context.Context, providerRef string, success bool) error
}

type Message struct {
	To   string
	Kind string
	Body string
}

type Notifier interface {
	Send(ctx context.Context, m Message) error
}

type PasswordHasher interface {
	Hash(pw string) (string, error)
	Compare(hash, pw string) bool
}

type Claims struct {
	UserID       string
	Role         d.Role
	TokenVersion int
	ExpiresAt    time.Time
}

type TokenIssuer interface {
	Issue(u *d.User) (string, time.Time, error)
	Parse(token string) (*Claims, error)
}

type Clock interface{ Now() time.Time }
