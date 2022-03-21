go get github.com/gin-gonic/gin
go get github.com/rs/xid
go get go.mongodb.org/mongo-driver/mongo
go get github.com/go-redis/redis/v8

go get -u
go mod tidy


docker run -p 6379:6379 --name goredis redis
docker stop goredis
docker start goredis

brew install redis
redis-cli ping

docker run -d --name redisinsight --link goredis -p 8001:8001 redislabs/redisinsight
