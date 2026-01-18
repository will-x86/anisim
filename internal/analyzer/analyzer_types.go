package analyzer

import (
	"github.com/will-x86/anisim/internal/types"
)

type ComparisonResult struct {
	SharedAnime map[string][]SharedEntry // Map of CURRENT|PLANNING|COMPLETED|DROPPED|PAUSED|REPEATING to list of shared entries
	SharedManga map[string][]SharedEntry
	//Stats 	  ComparisonStats
}

type SharedEntry struct {
	Media           types.Media
	CreatorScore    float64
	ComparatorScore float64
}
