sudo apt install postgresql postgresql-client

sudo -u postgres createuser nuno
sudo -u postgres createdb -O nuno nuno
# sudo -u postgres createuser galadriel
# sudo -u postgres -i

psql
psql -d nuno -h localhost -U nuno

# https://github.com/jackc/pgx/wiki/Getting-started-with-pgx
go get github.com/jackc/pgx
DATABASE_URL=postgresql://localhost:5432

https://cloud.digitalocean.com/databases/dbe5b6ad-dca6-458f-924f-139700ae2414/pools?i=3b1212

Later: Look into Connection pools

psql $DATABASE_URL -c "CREATE TABLE IF NOT EXISTS sources (id SERIAL PRIMARY KEY, title TEXT NOT NULL, link TEXT NOT NULL UNIQUE, date TIMESTAMP NOT NULL, summary TEXT, importance_bool BOOLEAN, importance_reasoning TEXT, created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP);"

psql $DATABASE

ALTER TABLE sources ADD COLUMN "processed" BOOLEAN DEFAULT FALSE;
ALTER TABLE sources ADD COLUMN "relevant_per_human_check" TEXT DEFAULT 'maybe';

psql $DATABASE_URL -f filename.sql

psql $DATABASE_URL -c "SELECT link FROM sources WHERE created_at < NOW() - INTERVAL '2 weeks';"
psql $DATABASE_URL -c "COPY (SELECT link FROM sources WHERE created_at < NOW() - INTERVAL '2 weeks') TO STDOUT WITH CSV;"

source .env && psql $DATABASE_URL -c "COPY (SELECT link FROM sources) TO STDOUT WITH CSV;" | grep gmw.cn

psql $DATABASE_URL -c "COPY (SELECT link FROM sources WHERE relevant_per_human_check = 'yes') TO STDOUT WITH CSV;"


To drop other connections

```
SELECT pg_terminate_backend(pg_stat_activity.pid)
FROM pg_stat_activity
WHERE datname = current_database()
  AND pid <> pg_backend_pid();
```
