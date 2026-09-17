module go.xyrillian.de/gg/benchmark

go 1.26

require (
	github.com/go-gorp/gorp/v3 v3.1.0
	github.com/gorilla/mux v1.8.1
	github.com/jackc/pgx/v5 v5.10.0
	github.com/lib/pq v1.12.3
	github.com/mattn/go-sqlite3 v1.14.48
	go.xyrillian.de/gg v1.15.0
	go.xyrillian.de/gg-pgx v0.1.1
	gorm.io/driver/sqlite v1.6.0
	gorm.io/gorm v1.31.2
)

require (
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	golang.org/x/sync v0.22.0 // indirect
	golang.org/x/text v0.40.0 // indirect
)

replace go.xyrillian.de/gg => ..
