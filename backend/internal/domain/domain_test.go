package domain

import (
	"testing"
	"time"
)

func TestFeeRounding(t *testing.T) {
	cases := []struct {
		bps  BPS
		in   Money
		want Money
	}{{1500, 30000, 4500}, {1200, 30000, 3600}, {1500, 3333, 500}, {1250, 101, 13}, {500, 1, 0}}
	for _, c := range cases {
		if got := c.bps.Of(c.in); got != c.want {
			t.Errorf("%d bps of %d = %d, want %d", c.bps, c.in, got, c.want)
		}
	}
}

func TestSplitConservesMoney(t *testing.T) {
	for _, net := range []Money{1, 99, 25500, 26401, 849999} {
		parts := Split(net, []int{70, 20, 10}, 0)
		var sum Money
		for _, p := range parts {
			sum += p
		}
		if sum != net {
			t.Fatalf("split of %d sums to %d", net, sum)
		}
	}
	if p := Split(1000, []int{0, 100}, 0); p[0] != 0 || p[1] != 1000 {
		t.Fatalf("unexpected split %v", p)
	}
}

func TestPhones(t *testing.T) {
	ok := map[string]Channel{"845123398": ChannelMpesa, "+258 85 123 4567": ChannelMpesa, "86 123 4567": ChannelEmola, "258871234567": ChannelEmola}
	for in, ch := range ok {
		n := NormalizeMZPhone(in)
		if n == "" || WalletOf(n) != ch {
			t.Errorf("%q -> %q (%s)", in, n, WalletOf(n))
		}
	}
	for _, bad := range []string{"", "81 123 4567", "8412345", "841234567890", "abc"} {
		if NormalizeMZPhone(bad) != "" {
			t.Errorf("%q should be invalid", bad)
		}
	}
	if WalletOf("821234567") != "" {
		t.Error("tmcel must not map to a wallet")
	}
}

func TestHandles(t *testing.T) {
	for _, h := range []string{"tupratu", "a.b.c", "abc123"} {
		if !ValidHandle(h) {
			t.Errorf("%q should be valid", h)
		}
	}
	for _, h := range []string{"ab", ".abc", "abc.", "a..b", "ABC", "a b", "averyveryverylonghandle12345"} {
		if ValidHandle(h) {
			t.Errorf("%q should be invalid", h)
		}
	}
}

func TestEffectiveFee(t *testing.T) {
	now := time.Now()
	past, future := now.Add(-time.Hour), now.Add(time.Hour)
	a := Account{FeeBPS: 1000, FeeUntil: &future}
	if a.EffectiveFee(now) != 1000 {
		t.Fatal("negotiated fee should apply before expiry")
	}
	a.FeeUntil = &past
	if a.EffectiveFee(now) != DefaultFeeBPS {
		t.Fatal("fee should revert after expiry")
	}
	if !BPS(1250).Valid() || BPS(1225).Valid() || BPS(2100).Valid() {
		t.Fatal("fee validation")
	}
}

func TestValidateSplits(t *testing.T) {
	ms := []Member{{ID: "a"}, {ID: "b"}}
	if err := ValidateSplits(ms, []MemberPct{{"a", 60}, {"b", 40}}); err != nil {
		t.Fatal(err)
	}
	for _, bad := range [][]MemberPct{{{"a", 60}, {"b", 30}}, {{"a", 100}}, {{"a", 50}, {"a", 50}}, {{"a", 110}, {"b", -10}}} {
		if ValidateSplits(ms, bad) == nil {
			t.Errorf("%v should be rejected", bad)
		}
	}
}
