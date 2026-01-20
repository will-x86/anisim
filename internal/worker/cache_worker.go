package worker

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/will-x86/anisim/internal/anilist"
	"github.com/will-x86/anisim/internal/cache"
	"github.com/will-x86/anisim/internal/db"
)

const (
	batchSize    = 50               // media match size
	pollInterval = 5 * time.Second  // polling interval, how often to check the queue
	retryDelay   = 60 * time.Second // rate limit retry delay
)

type CacheWorker struct {
	queries *db.Queries
	ctx     context.Context
	done    chan bool
}

func NewCacheWorker(queries *db.Queries) *CacheWorker {
	return &CacheWorker{
		queries: queries,
		ctx:     context.Background(),
		done:    make(chan bool),
	}
}

// Start processing the queue
func (w *CacheWorker) Start() {
	log.Println("Media cache worker started")
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			var skippedNumber int
			for {
				n, err := w.processQueue()
				if err != nil {
					log.Printf("Error processing queue: %v", err)
					break
				}
				if skippedNumber == batchSize {
					log.Printf("All %d items skipped, checking for more", skippedNumber)
					n, err := w.processQueue()
					if err != nil {
						log.Printf("Error processing queue: %v", err)
						break
					}
					skippedNumber = n
				}
				if n == 0 {
					break
				}

			}
		case <-w.done:
			log.Println("Media cache worker stopped")
			return
		}
	}
}

func (w *CacheWorker) Stop() {
	w.done <- true
}

func (w *CacheWorker) processQueue() (int, error) {
	items, err := w.queries.GetPendingQueueItems(w.ctx, int32(batchSize))
	if err != nil {
		return 0, err
	}

	if len(items) == 0 {
		return 0, nil
	}

	log.Printf("Processing %d queue items", len(items))
	skippedNumber := 0
	// Group by media type, skip items that don't need updating
	animeIDs := make(map[int32]int32) // media_id -> queue_item_id
	mangaIDs := make(map[int32]int32)

	for _, item := range items {
		// Mark as processing
		if err := w.queries.MarkQueueItemProcessing(w.ctx, item.ID); err != nil {
			log.Printf("Error marking item %d as processing: %v", item.ID, err)
			continue
		}

		needsUpdate, err := w.queries.NeedsCacheUpdate(w.ctx, item.MediaID)
		if err != nil {
			// Media doesn't exist in cache yet, so fetch it
			if item.MediaType == "ANIME" {
				animeIDs[item.MediaID] = item.ID
			} else {
				mangaIDs[item.MediaID] = item.ID
			}
			continue
		}

		if needsUpdate {
			if item.MediaType == "ANIME" {
				animeIDs[item.MediaID] = item.ID
			} else {
				mangaIDs[item.MediaID] = item.ID
			}
		} else {
			skippedNumber++
			log.Printf("Skipping media %d - recently cached", item.MediaID)
			if err := w.queries.MarkQueueItemCompleted(w.ctx, item.ID); err != nil {
				log.Printf("Error marking item %d as completed: %v", item.ID, err)
			}
		}
	}

	if len(animeIDs) > 0 {
		if err := w.processBatch(animeIDs, "ANIME"); err != nil {
			log.Printf("Error processing anime batch: %v", err)
		}
	}

	if len(mangaIDs) > 0 {
		if err := w.processBatch(mangaIDs, "MANGA"); err != nil {
			log.Printf("Error processing manga batch: %v", err)
		}
	}

	return skippedNumber, nil
}

func (w *CacheWorker) processBatch(mediaMap map[int32]int32, mediaType string) error {
	ids := make([]int, 0, len(mediaMap))
	for mediaID := range mediaMap {
		ids = append(ids, int(mediaID))
	}

	log.Printf("Fetching %d %s items from AniList", len(ids), mediaType)

	mediaList, err := anilist.GetMediaBatch(ids, mediaType)
	if err != nil {
		var rateLimitErr *anilist.RateLimitError
		if errors.As(err, &rateLimitErr) {
			log.Printf("Rate limited, retrying after %d seconds", rateLimitErr.RetryAfter)
			retryAfter := time.Now().Add(retryDelay)

			for _, queueItemID := range mediaMap {
				if err := w.queries.MarkQueueItemFailed(w.ctx, db.MarkQueueItemFailedParams{
					ID:         queueItemID,
					RetryAfter: pgtype.Timestamp{Time: retryAfter, Valid: true},
				}); err != nil {
					log.Printf("Error marking item %d for retry: %v", queueItemID, err)
				}
			}
			return rateLimitErr
		}
		return err
	}

	// Cache each media item
	for _, media := range mediaList {
		if err := cache.CacheMedia(w.ctx, w.queries, media); err != nil {
			log.Printf("Error caching media %d: %v", media.ID, err)
			queueItemID := mediaMap[int32(media.ID)]
			if err := w.queries.MarkQueueItemFailed(w.ctx, db.MarkQueueItemFailedParams{
				ID:         queueItemID,
				RetryAfter: pgtype.Timestamp{Valid: false},
			}); err != nil {
				log.Printf("Error marking item %d as failed: %v", queueItemID, err)
			}
			continue
		}

		// Mark as completed
		queueItemID := mediaMap[int32(media.ID)]
		if err := w.queries.MarkQueueItemCompleted(w.ctx, queueItemID); err != nil {
			log.Printf("Error marking item %d as completed: %v", queueItemID, err)
		}
	}

	log.Printf("Successfully cached %d %s items", len(mediaList), mediaType)
	return nil
}
