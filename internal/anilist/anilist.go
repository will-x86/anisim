package anilist

import (
	"context"

	"github.com/machinebox/graphql"
	"github.com/will-x86/anisim/internal/types"
)

var client *graphql.Client

func SetupClient(endpoint string) {
	client = graphql.NewClient(endpoint)
}

// Returns anime, manga, then error
func GetUsersMediaListCollection(id int) (types.MediaListCollection, types.MediaListCollection, error) {
	query := `
query ($type: MediaType!, $userId: Int!) {
  MediaListCollection(type: $type, userId: $userId) {
    lists {
      name
      status
      entries {
        id
        media {
          title {
            romaji
            english
          }
          coverImage {
            large
            medium
          }
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

	// Fetch Anime
	animeReq := graphql.NewRequest(query)
	animeReq.Var("userId", id)
	animeReq.Var("type", "ANIME")

	var animeResp struct {
		MediaListCollection types.MediaListCollection `json:"MediaListCollection"`
	}

	if err := client.Run(ctx, animeReq, &animeResp); err != nil {
		return types.MediaListCollection{}, types.MediaListCollection{}, err
	}

	// Fetch Manga
	mangaReq := graphql.NewRequest(query)
	mangaReq.Var("userId", id)
	mangaReq.Var("type", "MANGA")

	var mangaResp struct {
		MediaListCollection types.MediaListCollection `json:"MediaListCollection"`
	}

	if err := client.Run(ctx, mangaReq, &mangaResp); err != nil {
		return types.MediaListCollection{}, types.MediaListCollection{}, err
	}

	return animeResp.MediaListCollection, mangaResp.MediaListCollection, nil
}
func GetBasicUserInfo(username string) (types.User, error) {
	req := graphql.NewRequest(`query ($username: String) {
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

	if err := client.Run(ctx, req, &respData); err != nil {
		return types.User{}, err
	}
	return respData.User, nil

}
