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
