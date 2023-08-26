export COMPOSE_FILE=docker/compose.yml

bump:
	@go get -u
	@go mod tidy -compat=1.21

db:
	@ssh -t root@rolledout.info sudo -iu postgres psql rolled-out

db-dev:
	@docker-compose exec db psql -U postgres rolled-out

db-diff:
	@diff --color --label live --label dev --strip-trailing-cr -su         \
	    <(ssh root@rolledout.info sudo -iu postgres pg_dump -Os rolled-out \
	    | sed -E 's/ \(Debian .+//')                                       \
	    <(docker-compose exec -T db pg_dump -OsU postgres rolled-out)

db-dump:
	@ssh root@code.golf sudo -iu postgres pg_dump -a rolled-out \
	    > sql/b-data.sql

	@bin/sort-sql

dev:
	@docker-compose rm --force
	@docker-compose up --build

live:
	@docker build --file docker/assets.dockerfile \
	    --pull --tag rolled-out-assets .

	@docker run --rm --volume ./assets:/assets rolled-out-assets

	@docker build --file docker/app.dockerfile \
	    --pull --tag rolled-out --target live .

	@docker save rolled-out | ssh root@rolledout.info "          \
	    docker load;                                             \
	    docker stop rolled-out;                                  \
	    docker rm   rolled-out;                                  \
	    docker run                                               \
	        --detach                                             \
	        --env-file   /etc/rolled-out.env                     \
	        --init                                               \
	        --name       rolled-out                              \
	        --network    caddy                                   \
	        --read-only                                          \
	        --restart    always                                  \
	        --volume     /var/run/postgresql:/var/run/postgresql \
	        rolled-out;                                          \
	    docker system prune --force"

logs:
	@ssh root@rolledout.info docker logs --tail 5 -f rolled-out
