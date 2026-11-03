package domain

// Money is an amount in centavos (1 MT = 100).
type Money int64

const MT Money = 100

// BPS expresses a rate in basis points (1500 = 15%).
type BPS int

const (
	DefaultFeeBPS BPS = 1500
	MinFeeBPS     BPS = 500
	MaxFeeBPS     BPS = 2000
)

func (r BPS) Valid() bool { return r >= MinFeeBPS && r <= MaxFeeBPS && r%50 == 0 }

// Of returns the share of m at rate r, rounded half up.
func (r BPS) Of(m Money) Money {
	return Money((int64(m)*int64(r) + 5000) / 10000)
}

// Split divides net across weights (percentages that sum to 100). The
// remainder from rounding goes to the index given by remainderTo.
func Split(net Money, pcts []int, remainderTo int) []Money {
	out := make([]Money, len(pcts))
	var used Money
	for i, p := range pcts {
		out[i] = Money(int64(net) * int64(p) / 100)
		used += out[i]
	}
	if len(out) > 0 && remainderTo >= 0 && remainderTo < len(out) {
		out[remainderTo] += net - used
	}
	return out
}
