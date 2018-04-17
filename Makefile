run-redis:
	docker run --rm --name redis -d -p 6379:6379 redis:4.0 redis-server
#	docker run --rm --name redis -d -p 8888:6379 redis redis-server --appendonly yes
stop-redis:
	docker stop redis

