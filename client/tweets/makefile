include .env

run:
	go run src/main.go

trim:
	psql $(DATABASE_URL) -c "DELETE FROM sources WHERE processed = TRUE AND created_at < NOW() - INTERVAL '2 weeks';"
	psql $(DATABASE_URL) -c "SELECT * FROM sources WHERE processed = TRUE AND created_at < NOW() - INTERVAL '2 weeks';"

## SELECT COUNT(*) FROM your_table_name
# WHERE created_at >= NOW() - INTERVAL '8 HOUR';

