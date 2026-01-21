A comparison site for Anilist profiles.




Compare against your friends etc.



## Installation/running:
Install [Taskfile](https://taskfile.dev/docs/getting-started) then run:


```bash
cp .env.example .env 
docker compose up -d 
task setup 
task sqlc:generate
task db:migrate
task templ
task tailwind:build
task
```

Then open browser to localhost:3000

## Project Structure
```
.
├── cmd
│   └── anisim
│       └── main.go
├── docker-compose.yml # local development only at the moment 
├── go.mod/go.sum # Go modules
├── internal
│   ├── analyzer # For now just gets shared media for comparison
│   │   ├── analyzer.go/analyzer_types.go
│   ├── anilist
│   │   ├── anilist.go # GraphQL api client
│   │   └── recommendation/recommendation.go # Algorithm for generating recommendations
│   ├── cache/media.go # caches media from anilist
│   ├── db # sqlc generated code 
│   ├── handlers/ui/ui.go # handlers for endpoints
│   ├── server/server.go # server setup
│   ├── types/anisim.go # self explanatory
│   └── worker/cache_worker.go # background worker for anilist cache
├── migrations # database migrations
├── package/-lock.json # for tailwind
├── queries/queries.sql/sqlc.yaml # sqlc queries
├── static/css/.../tailwind.config.js # tailwind 
├── Taskfile.yml # Makefile alternative
└── templates
    ├── components/comparison.templ # comparisons component
    ├── layouts/base.templ # base layout 
    └── pages/comparison-home-recommendations.templ # pages 
```


Tech used:
- Go
- Tailwind
- Anilist GraphQL API
- Docker(compose
- SQLC
- Taskfile
- Postgres 
- Chi
- Templ



## Caching

Anilist has rigorous api limits, throttled to 15 per minute as of writing, so caching is fairly important.

The process is:
1. User requests comparison
2. We get their "media list" from Anilist (list of all anime/manga they have plan to watch / have watched)
- Previously at this step, I got all of the media info, aka tags, genre etc, but for users with large lists, this resulted in ~10s response time...
3. Respond with comparison, add all media ID's to database cache queue
4. Background worker runs on a 5s ticker, grabbing 50 at a a time from the DB, admittedly I could do all at once, but to save bursts of traffic, this is limited to 50.
5. If the media has *not* been updated in the last 24 hours, we fetch the full media from Anilist API.
6. If the entire batch of 50 are cached, and fresh, we ignore ticker and continue.
7. Otherwise, hit anilist API, store in cache, repeat.

- Above respects rate-limit, though through limitations of `github.com/machinebox/graphql` package, I cannot see the response headers, so I cannot dynamically adjust rate-limiting based on remaining requests. Sleeping for 60s instead.

