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
