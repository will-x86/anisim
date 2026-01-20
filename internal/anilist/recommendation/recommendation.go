package recommendation

import (
	"fmt"

	"github.com/will-x86/anisim/internal/db"
)

// have users, and entries. Create recommendation based upon this.
/*
score=(alignment_score*0.4) + (quality_score*0.3) + (relavence_score * 0.3)
alignment= (creator_score-mean_score)/10
quality=(mean-score/10 * 0.5) + (popularity_percentile * 0.3) + (favorites_percentile * 0.2)
popularity_percentile, most popular has: 942779
favorites_percentile, most favourited has: 91446
relevance = (matching_tags / total_tags) × (comparator_avg_score_for_those_tags / 10)
// Exclude:
- User B already watched
- Dropped / On-Hold by User A
- Boost - Completed by User A
Rank by total score



*/
func GetRecommendation(comparison db.Comparison, sharedEntries []db.SharedEntry) {
	// SharedEntry has score from users & mediaId
	// For now, score for creator, not comparator.

}
