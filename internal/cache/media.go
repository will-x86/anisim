package cache

import (
	"context"
	"log"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/will-x86/anisim/internal/db"
	"github.com/will-x86/anisim/internal/types"
)

// stores media metadata in the database for recommendation algorihtm
func CacheMedia(ctx context.Context, queries *db.Queries, media types.Media) error {
	// upsert, update if exists
	_, err := queries.UpsertMediaCache(ctx, db.UpsertMediaCacheParams{
		ID:               int32(media.ID),
		Type:             media.Type,
		TitleRomaji:      media.Title.Romaji,
		TitleEnglish:     pgtype.Text{String: media.Title.English, Valid: media.Title.English != ""},
		CoverImageLarge:  pgtype.Text{String: media.CoverImage.Large, Valid: media.CoverImage.Large != ""},
		CoverImageMedium: pgtype.Text{String: media.CoverImage.Medium, Valid: media.CoverImage.Medium != ""},
		Genres:           media.Genres,
		AverageScore:     pgtype.Int4{Int32: int32(ptrIntValue(media.AverageScore)), Valid: media.AverageScore != nil},
		MeanScore:        pgtype.Int4{Int32: int32(ptrIntValue(media.MeanScore)), Valid: media.MeanScore != nil},
		Popularity:       pgtype.Int4{Int32: int32(ptrIntValue(media.Popularity)), Valid: media.Popularity != nil},
		Favourites: pgtype.Int4{
			Int32: int32(media.Favourites), Valid: media.Favourites != 0,
		},
		Format:     pgtype.Text{String: media.Format, Valid: media.Format != ""},
		Status:     pgtype.Text{String: media.Status, Valid: media.Status != ""},
		SeasonYear: pgtype.Int4{Int32: int32(ptrIntValue(media.SeasonYear)), Valid: media.SeasonYear != nil},
		Season:     pgtype.Text{String: media.Season, Valid: media.Season != ""},
		IsAdult: pgtype.Bool{
			Bool: media.IsAdult, Valid: true,
		},
	})
	if err != nil {
		return err
	}

	for _, tag := range media.Tags {
		tagRecord, err := queries.GetOrCreateTag(ctx, tag.Name)
		if err != nil {
			log.Printf("Error creating tag %s: %v", tag.Name, err)
			continue
		}

		err = queries.UpsertMediaTag(ctx, db.UpsertMediaTagParams{
			MediaID: int32(media.ID),
			TagID:   tagRecord.ID,
			Rank:    int32(tag.Rank),
			IsMediaSpoiler: pgtype.Bool{
				Bool: tag.IsMediaSpoiler, Valid: true,
			},

			IsGeneralSpoiler: pgtype.Bool{
				Bool: tag.IsGeneralSpoiler, Valid: true,
			},
		})
		if err != nil {
			log.Printf("Error linking tag %s to media %d: %v", tag.Name, media.ID, err)
		}
	}

	return nil
}

// caches in background
func CacheMediaListInBackground(ctx context.Context, queries *db.Queries, collection types.MediaListCollection) {
	go func() {
		for _, list := range collection.Lists {
			for _, entry := range list.Entries {
				if err := CacheMedia(ctx, queries, entry.Media); err != nil {
					log.Printf("Error caching media %d: %v", entry.Media.ID, err)
				}
			}
		}
	}()
}

func ptrIntValue(ptr *int) int {
	if ptr == nil {
		return 0
	}
	return *ptr
}
