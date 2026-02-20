# XPFarm

## Installation

```bash
go install github.com/a3-n/xpfarm@latest
```

## Build

```bash
go build -o xpfarm

./xpfarm 
./xpfarm -debug
```

## Docker

```bash
# Using Docker Compose
docker-compose up --build

# Using standard Docker
docker build -t xpfarm .
docker run -p 8888:8888 -v $(pwd)/data:/app/data -v $(pwd)/screenshots:/app/screenshots xpfarm
```

## TODO

- [x] Redefine scan
- [x] Vuln scan change
- [ ] Vuln scan refine
- [x] Global Search
- [ ] Scan Settings
- [ ] System Settings
- [ ] SecretFinder JS
- [ ] Repo detect/scan
- [ ] Mobile scan
- [ ] Custom Module
- [ ] Agent Hell

Can we add something to the dashboard, just below Assets & Targets called Global Search. 
This should be a very dynamic search and super modular to allow searching for results based on all information we have. 

Make the search easy to understand and add mutiple fields types etc. 
So like search for tech stack X and Y in all assets where status code A and directories contain *.js 

As an example, this should be super customisable search, seeing as there might be 10s of thousands of results. 