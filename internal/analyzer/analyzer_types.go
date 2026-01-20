package analyzer

import (
	"github.com/will-x86/anisim/internal/types"
)

type ComparisonResult struct {
	AllSharedAnime []SharedEntry // Flat list of all shared anime
	AllSharedManga []SharedEntry // Flat list of all shared manga
}

type SharedEntry struct {
	MediaID          int
	Media            types.Media
	CreatorScore     float64
	ComparatorScore  float64
	CreatorStatus    string
	ComparatorStatus string
}
