package anilist

import (
	"context"
	"fmt"
	"time"

	"github.com/machinebox/graphql"
	"github.com/will-x86/anisim/internal/types"
)

var client *graphql.Client

type RateLimitError struct {
	RetryAfter int
	ResetAt    time.Time
}

func (e *RateLimitError) Error() string {
	return fmt.Sprintf("rate limit exceeded, retry after %d seconds", e.RetryAfter)
}

func SetupClient(endpoint string) {
	client = graphql.NewClient(endpoint)
}

func runWithRateLimit(ctx context.Context, req *graphql.Request, resp any) error {
	err := client.Run(ctx, req, resp)
	if err != nil {
		if err.Error() == "Too Many Requests." || err.Error() == "graphql: Too Many Requests." {
			return &RateLimitError{
				RetryAfter: 60, // machinebox doesn't provide headers
			}
		}
		return err
	}
	return nil
}

// GetUsersMediaListCollection returns anime, manga, then error
func GetUsersMediaListCollection(id int) (types.MediaListCollection, types.MediaListCollection, error) {
	query := `
	query ($userId: Int!) {
		anime: MediaListCollection(type: ANIME, userId: $userId) {
			lists {
				name
				status
				entries {
					id
					media {
						id
						type
						title {
							romaji
							english
						}
						coverImage {
							large
							medium
						}
						genres
						tags {
							name
							rank
							isMediaSpoiler
							isGeneralSpoiler
						}
						averageScore
						meanScore
						popularity
						favourites
						format
						status
						seasonYear
						season
						isAdult
					}
					mediaId
					progress
					score
					startedAt {
						year
						month
					}
					status
					progressVolumes
				}
				isCustomList
			}
		}
		manga: MediaListCollection(type: MANGA, userId: $userId) {
			lists {
				name
				status
				entries {
					id
					media {
						id
						type
						title {
							romaji
							english
						}
						coverImage {
							large
							medium
						}
						genres
						tags {
							name
							rank
							isMediaSpoiler
							isGeneralSpoiler
						}
						averageScore
						meanScore
						popularity
						favourites
						format
						status
						seasonYear
						season
						isAdult
					}
					mediaId
					progress
					score
					startedAt {
						year
						month
					}
					status
					progressVolumes
				}
				isCustomList
			}
		}
	}`

	ctx := context.Background()

	req := graphql.NewRequest(query)
	req.Var("userId", id)

	var resp struct {
		Anime types.MediaListCollection `json:"anime"`
		Manga types.MediaListCollection `json:"manga"`
	}

	if err := runWithRateLimit(ctx, req, &resp); err != nil {
		return types.MediaListCollection{}, types.MediaListCollection{}, err
	}

	return resp.Anime, resp.Manga, nil
}

func GetBasicUserInfo(username string) (types.User, error) {
	req := graphql.NewRequest(`
		query ($username: String) {
			User(name: $username) {
				id
				name
				avatar {
					large
					medium
				}
				createdAt
				statistics {
					anime {
						episodesWatched
						minutesWatched
					}
					manga {
						chaptersRead
						meanScore
					}
				}
			}
		}
	`)

	req.Var("username", username)
	ctx := context.Background()

	var respData struct {
		User types.User `json:"User"`
	}

	if err := runWithRateLimit(ctx, req, &respData); err != nil {
		return types.User{}, err
	}

	return respData.User, nil
}

// GetMediaBatch fetches media metadata for multiple IDs using Page query
func GetMediaBatch(ids []int, mediaType string) ([]types.Media, error) {
	if len(ids) == 0 {
		return []types.Media{}, nil
	}

	query := `
	query ($ids: [Int], $type: MediaType) {
		Page {
			media(id_in: $ids, type: $type) {
				id
				type
				title {
					romaji
					english
				}
				coverImage {
					large
					medium
				}
				genres
				tags {
					name
					rank
					isMediaSpoiler
					isGeneralSpoiler
				}
				averageScore
				meanScore
				popularity
				favourites
				format
				status
				seasonYear
				season
				isAdult
			}
		}
	}`

	ctx := context.Background()
	req := graphql.NewRequest(query)
	req.Var("ids", ids)
	req.Var("type", mediaType)

	var resp struct {
		Page struct {
			Media []types.Media `json:"media"`
		} `json:"Page"`
	}

	if err := runWithRateLimit(ctx, req, &resp); err != nil {
		return nil, err
	}

	return resp.Page.Media, nil
}

func GetComparisonData(creatorUsername, comparatorUsername string) (
	creatorUser types.User,
	creatorAnime types.MediaListCollection,
	creatorManga types.MediaListCollection,
	comparatorUser types.User,
	comparatorAnime types.MediaListCollection,
	comparatorManga types.MediaListCollection,
	err error,
) {
	query := `
	query ($creatorUsername: String, $comparatorUsername: String) {
		creator: User(name: $creatorUsername) {
			id
			name
			avatar {
				large
				medium
			}
			createdAt
			statistics {
				anime {
					episodesWatched
					minutesWatched
				}
				manga {
					chaptersRead
					meanScore
				}
			}
		}
		comparator: User(name: $comparatorUsername) {
			id
			name
			avatar {
				large
				medium
			}
			createdAt
			statistics {
				anime {
					episodesWatched
					minutesWatched
				}
				manga {
					chaptersRead
					meanScore
				}
			}
		}
	}`

	ctx := context.Background()
	req := graphql.NewRequest(query)
	req.Var("creatorUsername", creatorUsername)
	req.Var("comparatorUsername", comparatorUsername)

	var resp struct {
		Creator    types.User `json:"creator"`
		Comparator types.User `json:"comparator"`
	}

	if err := runWithRateLimit(ctx, req, &resp); err != nil {
		return types.User{}, types.MediaListCollection{}, types.MediaListCollection{},
			types.User{}, types.MediaListCollection{}, types.MediaListCollection{}, err
	}

	// Now fetch media lists with the user IDs we just got
	creatorAnime, creatorManga, err = GetUsersMediaListCollection(resp.Creator.ID)
	if err != nil {
		return types.User{}, types.MediaListCollection{}, types.MediaListCollection{},
			types.User{}, types.MediaListCollection{}, types.MediaListCollection{}, err
	}

	comparatorAnime, comparatorManga, err = GetUsersMediaListCollection(resp.Comparator.ID)
	if err != nil {
		return types.User{}, types.MediaListCollection{}, types.MediaListCollection{},
			types.User{}, types.MediaListCollection{}, types.MediaListCollection{}, err
	}

	return resp.Creator, creatorAnime, creatorManga,
		resp.Comparator, comparatorAnime, comparatorManga, nil
}
