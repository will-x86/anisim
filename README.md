A comparison site for Anilist profiles.




Compare against your friends etc.



## Installation/running:
Install [Taskfile](https://taskfile.dev/docs/getting-started) then run:


```bash
docker compose up -d 
task setup 
task sqlc:generate
task db:migrate
task templ
task tailwind:build
task
```

Then open browser to localhost:3000
