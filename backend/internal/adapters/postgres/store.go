package postgres

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	pg "gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"

	d "github.com/refresco/refresco-v2/backend/internal/domain"
	"github.com/refresco/refresco-v2/backend/internal/ports"
)

type Store struct{ db *gorm.DB }

func Open(dsn string, maxOpen int, debug bool) (*gorm.DB, error) {
	lvl := logger.Warn
	if debug {
		lvl = logger.Info
	}
	db, err := gorm.Open(pg.Open(dsn), &gorm.Config{
		Logger: logger.New(log.New(os.Stderr, "", log.LstdFlags), logger.Config{
			SlowThreshold: 500 * time.Millisecond, LogLevel: lvl, IgnoreRecordNotFoundError: true, ParameterizedQueries: true,
		}),
		NowFunc:                func() time.Time { return time.Now().UTC() },
		TranslateError:         false,
		SkipDefaultTransaction: true,
	})
	if err != nil {
		return nil, err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(maxOpen)
	sqlDB.SetMaxIdleConns(maxOpen / 2)
	sqlDB.SetConnMaxLifetime(30 * time.Minute)
	return db, nil
}

func Migrate(db *gorm.DB) error {
	if err := db.Exec("CREATE SEQUENCE IF NOT EXISTS withdrawal_number_seq START 1").Error; err != nil {
		return fmt.Errorf("sequence: %w", err)
	}
	if err := db.AutoMigrate(allModels()...); err != nil {
		return fmt.Errorf("automigrate: %w", err)
	}
	return nil
}

func New(db *gorm.DB) *Store { return &Store{db: db} }

func (s *Store) Tx(ctx context.Context, fn func(ports.Store) error) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error { return fn(&Store{db: tx}) })
}

func (s *Store) Users() ports.UserRepo                   { return userRepo{s.db} }
func (s *Store) Accounts() ports.AccountRepo             { return accountRepo{s.db} }
func (s *Store) Members() ports.MemberRepo               { return memberRepo{s.db} }
func (s *Store) PayoutAccounts() ports.PayoutAccountRepo { return payoutRepo{s.db} }
func (s *Store) Supports() ports.SupportRepo             { return supportRepo{s.db} }
func (s *Store) Withdrawals() ports.WithdrawalRepo       { return withdrawalRepo{s.db} }
func (s *Store) Invites() ports.InviteRepo               { return inviteRepo{s.db} }
func (s *Store) Proposals() ports.ProposalRepo           { return proposalRepo{s.db} }
func (s *Store) Audit() ports.AuditRepo                  { return auditRepo{s.db} }
func (s *Store) Resets() ports.ResetRepo                 { return resetRepo{s.db} }
func (s *Store) Waitlist() ports.WaitlistRepo            { return waitlistRepo{s.db} }
func (s *Store) Ledger() ports.LedgerRepo                { return ledgerRepo{s.db} }
func (s *Store) Stats() ports.StatsWriter                { return statsRepo{s.db} }

func mapErr(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return d.ErrNotFound
	}
	var pe *pgconn.PgError
	if errors.As(err, &pe) {
		switch pe.Code {
		case "23505":
			return d.Conflict("duplicate", "Já existe um registo com estes dados.")
		case "23514":
			return d.Invalid("constraint", "Operação inválida.")
		}
	}
	return err
}

func forUpdate(db *gorm.DB) *gorm.DB { return db.Clauses(clause.Locking{Strength: "UPDATE"}) }

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func deref(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}
