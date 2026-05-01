package models

import "testing"

func TestCalculateByTokensWithCacheAndContextMicros_UsesOver200KPricing(t *testing.T) {
	m := &AIModel{
		Pricing: ModelPricing{
			InputPrice:               1,
			OutputPrice:              2,
			CacheCreatePrice:         3,
			CacheReadPrice:           4,
			Over200KInputPrice:       10,
			Over200KOutputPrice:      20,
			Over200KCacheCreatePrice: 30,
			Over200KCacheReadPrice:   40,
		},
	}

	input := int64(1_000_000)
	output := int64(1_000_000)
	cacheCreate := int64(1_000_000)
	cacheRead := int64(1_000_000)

	gotNormal := m.CalculateByTokensWithCacheAndContextMicros(input, output, cacheCreate, cacheRead, 200_000)
	wantNormal := MoneyFloatToMicros(1 + 2 + 3 + 4)
	if gotNormal != wantNormal {
		t.Fatalf("expected normal pricing cost %d, got %d", wantNormal, gotNormal)
	}

	gotOver := m.CalculateByTokensWithCacheAndContextMicros(input, output, cacheCreate, cacheRead, 200_001)
	wantOver := MoneyFloatToMicros(10 + 20 + 30 + 40)
	if gotOver != wantOver {
		t.Fatalf("expected over200k pricing cost %d, got %d", wantOver, gotOver)
	}
}

func TestCalculateByTokensWithCacheAndContextMicros_FallbackWhenOver200KPriceMissing(t *testing.T) {
	m := &AIModel{
		Pricing: ModelPricing{
			InputPrice:               1,
			OutputPrice:              2,
			CacheCreatePrice:         3,
			CacheReadPrice:           4,
			Over200KInputPrice:       10,
			Over200KOutputPrice:      0,
			Over200KCacheCreatePrice: 30,
			Over200KCacheReadPrice:   0,
		},
	}

	got := m.CalculateByTokensWithCacheAndContextMicros(
		1_000_000,
		1_000_000,
		1_000_000,
		1_000_000,
		200_001,
	)
	want := MoneyFloatToMicros(10 + 2 + 30 + 4)
	if got != want {
		t.Fatalf("expected fallback pricing cost %d, got %d", want, got)
	}
}

func TestCalculateByRequestWithContextMicros_UsesOver200KAndFallback(t *testing.T) {
	m := &AIModel{
		Pricing: ModelPricing{
			PerRequestPrice:         0.5,
			Over200KPerRequestPrice: 1.25,
		},
	}

	gotNormal := m.CalculateByRequestWithContextMicros(200_000)
	wantNormal := MoneyFloatToMicros(0.5)
	if gotNormal != wantNormal {
		t.Fatalf("expected normal request pricing %d, got %d", wantNormal, gotNormal)
	}

	gotOver := m.CalculateByRequestWithContextMicros(200_001)
	wantOver := MoneyFloatToMicros(1.25)
	if gotOver != wantOver {
		t.Fatalf("expected over200k request pricing %d, got %d", wantOver, gotOver)
	}

	m.Pricing.Over200KPerRequestPrice = 0
	gotFallback := m.CalculateByRequestWithContextMicros(300_000)
	wantFallback := MoneyFloatToMicros(0.5)
	if gotFallback != wantFallback {
		t.Fatalf("expected fallback request pricing %d, got %d", wantFallback, gotFallback)
	}
}
