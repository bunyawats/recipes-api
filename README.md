go get github.com/gin-gonic/gin
go get github.com/rs/xid
go get go.mongodb.org/mongo-driver/mongo
go get github.com/go-redis/redis/v8

go get -u
go mod tidy

brew install jq

brew tap go-swagger/go-swagger
brew install go-swagger
swagger version

docker run -p 6379:6379 --name goredis redis
docker stop goredis
docker start goredis

brew install redis
redis-cli ping

docker run -d --name redisinsight --link goredis -p 8001:8001 redislabs/redisinsight

brew install apache2
ab -n 2000 -c 100 -g without-cache.data http://localhost:8080/recipes
ab -n 2000 -c 100 -g with-cache.data http://localhost:8080/recipes

brew install gnuplot
gnuplot apache-benchmark.p


