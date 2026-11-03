package postgres

import (
	"time"

	"gorm.io/datatypes"
)

type userModel struct {
	ID           string `gorm:"type:uuid;primaryKey"`
	Email        string `gorm:"size:254;uniqueIndex;not null"`
	Phone        *string `gorm:"size:9;uniqueIndex"`
	Name         string `gorm:"size:80;not null"`
	PasswordHash string `gorm:"size:100;not null"`
	Role         string `gorm:"size:20;not null;index"`
	TokenVersion int    `gorm:"not null;default:0"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (userModel) TableName() string { return "users" }

type accountModel struct {
	ID          string `gorm:"type:uuid;primaryKey"`
	OwnerID     string `gorm:"type:uuid;not null;index"`
	Handle      string `gorm:"size:24;uniqueIndex;not null"`
	Name        string `gorm:"size:40;not null"`
	Category    string `gorm:"size:40;not null"`
	Bio         string `gorm:"size:800;not null;default:''"`
	Thanks      string `gorm:"size:800;not null;default:''"`
	Amounts     datatypes.JSONSlice[int64] `gorm:"type:jsonb;not null"`
	DefaultIdx  int    `gorm:"not null"`
	AllowCustom bool   `gorm:"not null"`
	MinAmount   int64  `gorm:"not null"`
	Socials     datatypes.JSONMap `gorm:"type:jsonb;not null"`
	ChMpesa     bool   `gorm:"not null"`
	ChEmola     bool   `gorm:"not null"`
	ChCard      bool   `gorm:"not null"`
	FeeBPS      int    `gorm:"not null;check:fee_bps_range,fee_bps >= 0 AND fee_bps <= 5000"`
	FeeNote     string `gorm:"size:240;not null;default:''"`
	FeeUntil    *time.Time
	Status      string `gorm:"size:20;not null;index"`
	StatusNote  string `gorm:"size:800;not null;default:''"`
	InviteCode  string `gorm:"size:24;index"`
	AutoSplit   bool   `gorm:"not null"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (accountModel) TableName() string { return "accounts" }

type memberModel struct {
	ID           string  `gorm:"type:uuid;primaryKey"`
	AccountID    string  `gorm:"type:uuid;not null;index"`
	UserID       *string `gorm:"type:uuid;uniqueIndex"`
	Name         string  `gorm:"size:40;not null"`
	Role         string  `gorm:"size:40;not null"`
	Pct          int     `gorm:"not null;check:pct_range,pct >= 0 AND pct <= 100"`
	Wallet       string  `gorm:"size:10;not null"`
	WalletNumber string  `gorm:"size:9;not null"`
	IsAdmin      bool    `gorm:"not null"`
	Status       string  `gorm:"size:20;not null"`
	Balance      int64   `gorm:"not null;default:0;check:balance_non_negative,balance >= 0"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (memberModel) TableName() string { return "members" }

type payoutAccountModel struct {
	ID        string `gorm:"type:uuid;primaryKey"`
	MemberID  string `gorm:"type:uuid;not null;index"`
	Kind      string `gorm:"size:10;not null"`
	Number    string `gorm:"size:32;not null"`
	BankName  string `gorm:"size:40;not null;default:''"`
	IsMain    bool   `gorm:"not null"`
	CreatedAt time.Time
}

func (payoutAccountModel) TableName() string { return "payout_accounts" }

type supportModel struct {
	ID            string `gorm:"type:uuid;primaryKey"`
	Ref           string `gorm:"size:16;uniqueIndex;not null"`
	AccountID     string `gorm:"type:uuid;not null;index:idx_supports_account_created,priority:1"`
	Amount        int64  `gorm:"not null;check:amount_positive,amount > 0"`
	FeeBPS        int    `gorm:"not null;default:0"`
	Fee           int64  `gorm:"not null;default:0"`
	Net           int64  `gorm:"not null;default:0"`
	Preset        bool   `gorm:"not null"`
	Channel       string `gorm:"size:10;not null"`
	PayerPhone    string `gorm:"size:9;not null;default:''"`
	CardBrand     string `gorm:"size:12;not null;default:''"`
	CardLast4     string `gorm:"size:4;not null;default:''"`
	SupporterName string `gorm:"size:160;not null;default:''"`
	SupporterNet  string `gorm:"size:12;not null;default:''"`
	Message       string `gorm:"size:1200;not null;default:''"`
	Status        string `gorm:"size:12;not null;index"`
	FailReason    string `gorm:"size:120;not null;default:''"`
	ProviderRef   *string `gorm:"size:80;uniqueIndex"`
	NextAction    string `gorm:"size:20;not null;default:''"`
	ExpiresAt     time.Time
	IdemKey       *string `gorm:"size:64;uniqueIndex:idx_supports_idem,composite:idem"`
	IdemAccount   *string `gorm:"type:uuid;uniqueIndex:idx_supports_idem,composite:idem"`
	ReadAt        *time.Time
	ConfirmedAt   *time.Time
	CreatedAt     time.Time `gorm:"index:idx_supports_account_created,priority:2,sort:desc"`
	UpdatedAt     time.Time
}

func (supportModel) TableName() string { return "supports" }

type supportSplitModel struct {
	SupportID string `gorm:"type:uuid;primaryKey"`
	MemberID  string `gorm:"type:uuid;primaryKey"`
	Name      string `gorm:"size:40;not null"`
	Pct       int    `gorm:"not null"`
	Amount    int64  `gorm:"not null"`
}

func (supportSplitModel) TableName() string { return "support_splits" }

type withdrawalModel struct {
	ID          string `gorm:"type:uuid;primaryKey"`
	Number      int64  `gorm:"type:bigint;uniqueIndex;not null;default:nextval('withdrawal_number_seq')"`
	AccountID   string `gorm:"type:uuid;not null;index"`
	MemberID    string `gorm:"type:uuid;not null;index"`
	Amount      int64  `gorm:"not null;check:withdrawal_positive,amount > 0"`
	DestKind    string `gorm:"size:10;not null"`
	DestNumber  string `gorm:"size:32;not null"`
	DestBank    string `gorm:"size:40;not null;default:''"`
	Status      string `gorm:"size:12;not null;index"`
	Reason      string `gorm:"size:80;not null;default:''"`
	Note        string `gorm:"size:1200;not null;default:''"`
	TransferRef string `gorm:"size:40;not null;default:''"`
	DueAt       time.Time
	DecidedAt   *time.Time
	PaidAt      *time.Time
	DecidedBy   string `gorm:"size:36;not null;default:''"`
	CreatedAt   time.Time `gorm:"index"`
	UpdatedAt   time.Time
}

func (withdrawalModel) TableName() string { return "withdrawals" }

type inviteModel struct {
	Code      string `gorm:"size:24;primaryKey"`
	Note      string `gorm:"size:160;not null"`
	MaxUses   int    `gorm:"not null"`
	Uses      int    `gorm:"not null;default:0;check:uses_le_max,uses <= max_uses"`
	FeeBPS    int    `gorm:"not null"`
	Status    string `gorm:"size:12;not null"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (inviteModel) TableName() string { return "invites" }

type proposalModel struct {
	ID        string `gorm:"type:uuid;primaryKey"`
	AccountID string `gorm:"type:uuid;not null;index"`
	Splits    datatypes.JSON `gorm:"type:jsonb;not null"`
	Status    string `gorm:"size:12;not null;index"`
	CreatedBy string `gorm:"type:uuid;not null"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (proposalModel) TableName() string { return "split_proposals" }

type approvalModel struct {
	ProposalID string `gorm:"type:uuid;primaryKey"`
	MemberID   string `gorm:"type:uuid;primaryKey"`
	TokenHash  string `gorm:"size:64;uniqueIndex;not null"`
	ApprovedAt *time.Time
}

func (approvalModel) TableName() string { return "split_approvals" }

type auditModel struct {
	ID        string `gorm:"type:uuid;primaryKey"`
	AccountID string `gorm:"type:uuid;not null;index:idx_audit_account_scope,priority:1"`
	Scope     string `gorm:"size:16;not null;index:idx_audit_account_scope,priority:2"`
	Text      string `gorm:"size:600;not null"`
	CreatedAt time.Time
}

func (auditModel) TableName() string { return "audit_entries" }

type resetModel struct {
	ID             string `gorm:"type:uuid;primaryKey"`
	UserID         string `gorm:"type:uuid;not null;index"`
	CodeHash       string `gorm:"size:64;not null"`
	ExpiresAt      time.Time
	Attempts       int     `gorm:"not null;default:0"`
	ResetTokenHash *string `gorm:"size:64;uniqueIndex"`
	VerifiedAt     *time.Time
	UsedAt         *time.Time
	CreatedAt      time.Time
}

func (resetModel) TableName() string { return "password_resets" }

type waitlistModel struct {
	ID        int64  `gorm:"primaryKey;autoIncrement"`
	Email     string `gorm:"size:254;uniqueIndex;not null"`
	CreatedAt time.Time
}

func (waitlistModel) TableName() string { return "waitlist" }

type ledgerModel struct {
	ID        int64  `gorm:"primaryKey;autoIncrement"`
	MemberID  string `gorm:"type:uuid;not null;index"`
	AccountID string `gorm:"type:uuid;not null;index"`
	Kind      string `gorm:"size:24;not null"`
	Amount    int64  `gorm:"not null"`
	Ref       string `gorm:"size:64;not null"`
	CreatedAt time.Time
}

func (ledgerModel) TableName() string { return "ledger_entries" }

// dailyStatModel is the aggregate every metric reads from: one row per account, local day and channel.
type dailyStatModel struct {
	AccountID string    `gorm:"type:uuid;primaryKey"`
	Day       time.Time `gorm:"type:date;primaryKey;index"`
	Channel   string    `gorm:"size:10;primaryKey"`
	Gross     int64     `gorm:"not null;default:0"`
	Fee       int64     `gorm:"not null;default:0"`
	Count     int64     `gorm:"not null;default:0"`
}

func (dailyStatModel) TableName() string { return "daily_stats" }

func allModels() []any {
	return []any{&userModel{}, &accountModel{}, &memberModel{}, &payoutAccountModel{}, &supportModel{}, &supportSplitModel{},
		&withdrawalModel{}, &inviteModel{}, &proposalModel{}, &approvalModel{}, &auditModel{}, &resetModel{}, &waitlistModel{},
		&ledgerModel{}, &dailyStatModel{}}
}
