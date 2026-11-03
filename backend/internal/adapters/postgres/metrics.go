package postgres

import (
	"context"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	d "github.com/refresco/refresco-v2/backend/internal/domain"
	"github.com/refresco/refresco-v2/backend/internal/ports"
)

type statsRepo struct{ db *gorm.DB }

func dayOf(t time.Time) time.Time { return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC) }

func (r statsRepo) RecordSupport(ctx context.Context, acc string, at time.Time, ch d.Channel, amount, fee d.Money) error {
	row := dailyStatModel{AccountID: acc, Day: dayOf(at), Channel: string(ch), Gross: int64(amount), Fee: int64(fee), Count: 1}
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "account_id"}, {Name: "day"}, {Name: "channel"}},
		DoUpdates: clause.Assignments(map[string]any{
			"gross": gorm.Expr("daily_stats.gross + EXCLUDED.gross"),
			"fee":   gorm.Expr("daily_stats.fee + EXCLUDED.fee"),
			"count": gorm.Expr("daily_stats.count + EXCLUDED.count"),
		}),
	}).Create(&row).Error
}

type Metrics struct{ db *gorm.DB }

func NewMetrics(db *gorm.DB) *Metrics { return &Metrics{db: db} }

func (m *Metrics) stats(ctx context.Context, acc string, from, to time.Time) *gorm.DB {
	q := m.db.WithContext(ctx).Table("daily_stats").Where("day >= ? AND day < ?", dayOf(from), dayOf(to))
	if acc != "" {
		q = q.Where("account_id = ?", acc)
	}
	return q
}

func addChan(c *ports.ChannelTotals, ch string, v d.Money) {
	switch d.Channel(ch) {
	case d.ChannelMpesa:
		c.Mpesa += v
	case d.ChannelEmola:
		c.Emola += v
	case d.ChannelCard:
		c.Card += v
	}
}

func (m *Metrics) Series(ctx context.Context, acc string, from, to time.Time, b ports.Bucket, loc *time.Location) ([]ports.SeriesPoint, error) {
	trunc := "day"
	if b == ports.BucketMonth {
		trunc = "month"
	}
	var rows []struct {
		Bucket  time.Time
		Channel string
		Gross   int64
		Fee     int64
		Count   int64
	}
	err := m.stats(ctx, acc, from, to).
		Select("date_trunc('" + trunc + "', day)::date AS bucket, channel, SUM(gross) AS gross, SUM(fee) AS fee, SUM(count) AS count").
		Group("bucket, channel").Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	idx := map[string]int{}
	var pts []ports.SeriesPoint
	for t := from; t.Before(to); {
		idx[t.Format("2006-01-02")] = len(pts)
		pts = append(pts, ports.SeriesPoint{Start: t})
		if b == ports.BucketMonth {
			t = t.AddDate(0, 1, 0)
		} else {
			t = t.AddDate(0, 0, 1)
		}
	}
	for _, r := range rows {
		i, ok := idx[r.Bucket.Format("2006-01-02")]
		if !ok {
			continue
		}
		p := &pts[i]
		p.Total += d.Money(r.Gross)
		p.Fee += d.Money(r.Fee)
		p.Count += r.Count
		addChan(&p.ByChan, r.Channel, d.Money(r.Gross))
	}
	return pts, nil
}

func (m *Metrics) Totals(ctx context.Context, acc string, from, to time.Time) (ports.PeriodTotals, error) {
	var out ports.PeriodTotals
	var rows []struct {
		Channel string
		Gross   int64
		Fee     int64
		Count   int64
	}
	if err := m.stats(ctx, acc, from, to).Select("channel, SUM(gross) AS gross, SUM(fee) AS fee, SUM(count) AS count").Group("channel").Scan(&rows).Error; err != nil {
		return out, err
	}
	for _, r := range rows {
		out.Gross += d.Money(r.Gross)
		out.Fee += d.Money(r.Fee)
		out.Count += r.Count
		addChan(&out.ByChan, r.Channel, d.Money(r.Gross))
	}
	var agg struct {
		Supporters int64
		Mode       *int64
	}
	q := m.db.WithContext(ctx).Table("supports").
		Select(`COUNT(DISTINCT COALESCE(NULLIF(payer_phone, ''), card_brand || card_last4 || lower(supporter_name))) AS supporters,
			mode() WITHIN GROUP (ORDER BY amount) AS mode`).
		Where("status = ? AND confirmed_at >= ? AND confirmed_at < ?", string(d.SupportConfirmed), from.UTC(), to.UTC())
	if acc != "" {
		q = q.Where("account_id = ?", acc)
	}
	if err := q.Scan(&agg).Error; err != nil {
		return out, err
	}
	out.Supporter = agg.Supporters
	if agg.Mode != nil {
		out.TopAmount = d.Money(*agg.Mode)
	}
	return out, nil
}

func (m *Metrics) AccountAggregates(ctx context.Context, from, to time.Time) (map[string]ports.AccountAggregate, error) {
	out := map[string]ports.AccountAggregate{}
	var rows []struct {
		AccountID string
		Count     int64
		Gross     int64
		Fee       int64
		Total     int64
		TotalFee  int64
	}
	err := m.db.WithContext(ctx).Table("daily_stats").Select(`account_id,
		SUM(count) FILTER (WHERE day >= @from AND day < @to) AS count,
		SUM(gross) FILTER (WHERE day >= @from AND day < @to) AS gross,
		SUM(fee) FILTER (WHERE day >= @from AND day < @to) AS fee,
		SUM(gross) AS total, SUM(fee) AS total_fee`, map[string]any{"from": dayOf(from), "to": dayOf(to)}).
		Group("account_id").Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, r := range rows {
		out[r.AccountID] = ports.AccountAggregate{AccountID: r.AccountID, Count: r.Count, Volume: d.Money(r.Gross), Fee: d.Money(r.Fee), Total: d.Money(r.Total), TotalFee: d.Money(r.TotalFee)}
	}
	var bals []struct {
		AccountID string
		Balance   int64
	}
	if err := m.db.WithContext(ctx).Table("members").Select("account_id, SUM(balance) AS balance").Group("account_id").Scan(&bals).Error; err != nil {
		return nil, err
	}
	for _, b := range bals {
		a := out[b.AccountID]
		a.AccountID, a.Balance = b.AccountID, d.Money(b.Balance)
		out[b.AccountID] = a
	}
	return out, nil
}

func (m *Metrics) BalanceTotal(ctx context.Context, acc string) (d.Money, error) {
	var v *int64
	q := m.db.WithContext(ctx).Table("members").Select("SUM(balance)")
	if acc != "" {
		q = q.Where("account_id = ?", acc)
	}
	if err := q.Scan(&v).Error; err != nil || v == nil {
		return 0, err
	}
	return d.Money(*v), nil
}
