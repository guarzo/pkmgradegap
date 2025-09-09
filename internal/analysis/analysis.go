package analysis

import (
	"fmt"
	"math"
	"strconv"

	"github.com/guarzo/pkmgradegap/internal/model"
)

type Grades struct {
	PSA10   float64
	Grade9  float64
	Grade95 float64 // maps to 9.5 (PC "box-only-price")
	BGS10   float64
}

type Row struct {
	Card    model.Card
	RawUSD  float64
	RawSrc  string
	RawNote string
	Grades  Grades
}

// Prefer TCGplayer USD market if present; fallback to Cardmarket trend (EUR) with a note.
// (You can add FX conversion or user-selectable sources later.)
func ExtractUngradedUSD(c model.Card) (value float64, source string, note string) {
	// Try TCGplayer markets in order of likeliest printing buckets.
	if c.TCGPlayer != nil && c.TCGPlayer.Prices != nil {
		typeNameOrder := []string{"normal", "holofoil", "reverseHolofoil", "1stEditionHolofoil", "1stEditionNormal"}
		for _, t := range typeNameOrder {
			if p, ok := c.TCGPlayer.Prices[t]; ok && p.Market != nil && *p.Market > 0 {
				return round2(*p.Market), "tcgplayer.market", "USD"
			}
		}
	}
	// Fallback to Cardmarket trend (EUR). Caller can decide how to treat this.
	if c.Cardmarket != nil && c.Cardmarket.Prices.TrendPrice != nil && *c.Cardmarket.Prices.TrendPrice > 0 {
		return round2(*c.Cardmarket.Prices.TrendPrice), "cardmarket.trend", "EUR"
	}
	return 0, "", ""
}

func ReportRawVsPSA10(rows []Row) [][]string {
	out := [][]string{
		{"Card", "Number", "RawUSD", "RawSource", "PSA10_USD", "Delta_USD", "Notes"},
	}
	for _, r := range rows {
		if r.RawUSD <= 0 || r.Grades.PSA10 <= 0 {
			continue
		}
		delta := r.Grades.PSA10 - r.RawUSD
		notes := r.RawNote
		out = append(out, []string{
			fmt.Sprintf("%s", r.Card.Name),
			r.Card.Number,
			money(r.RawUSD),
			r.RawSrc,
			money(r.Grades.PSA10),
			money(delta),
			notes,
		})
	}
	return out
}

func ReportMultiVsPSA10(rows []Row) [][]string {
	out := [][]string{
		{"Card", "Number", "PSA9_USD", "CGC/BGS_9.5_USD", "BGS10_USD", "PSA10_USD", "PSA9/10_%", "9.5/10_%", "BGS10/PSA10_%"},
	}
	for _, r := range rows {
		if r.Grades.PSA10 <= 0 {
			continue
		}
		psa9pct := pct(r.Grades.Grade9, r.Grades.PSA10)
		g95pct := pct(r.Grades.Grade95, r.Grades.PSA10)
		bgs10pct := pct(r.Grades.BGS10, r.Grades.PSA10)
		out = append(out, []string{
			r.Card.Name,
			r.Card.Number,
			money(r.Grades.Grade9),
			money(r.Grades.Grade95),
			money(r.Grades.BGS10),
			money(r.Grades.PSA10),
			psa9pct,
			g95pct,
			bgs10pct,
		})
	}
	return out
}

func pct(a, b float64) string {
	if a <= 0 || b <= 0 {
		return ""
	}
	return fmt.Sprintf("%.1f%%", (a/b)*100.0)
}

func money(v float64) string {
	if v <= 0 {
		return ""
	}
	return "$" + strconv.FormatFloat(round2(v), 'f', 2, 64)
}

func round2(f float64) float64 {
	return math.Round(f*100) / 100
}