go get github.com/gin-gonic/gin
go get github.com/rs/xid
go get go.mongodb.org/mongo-driver/mongo
go get github.com/go-redis/redis/v8
go get github.com/gomodule/redigo@latest
go get github.com/dgrijalva/jwt-go
go get github.com/gin-contrib/sessions
go get -v gopkg.in/square/go-jose.v2
go get -v github.com/auth0-community/go-auth0

go get -u
go mod tidy

brew install jq

brew tap go-swagger/go-swagger
brew install go-swagger
swagger version

swagger generate spec -o ./swagger.json
swagger serve --flavor=swagger ./swagger.json

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

https://jwt.io/

https://auth0.com/

https://ngrok.com/
brew install ngrok/ngrok/ngrok
ngrok http 8080
http://ed7e-58-97-79-30.ngrok.io/recipes

openssl req -x509 -nodes -days 365 -newkey rsa:2048 -keyout certs/localhost.key -out certs/localhost.crt
chrome://flags/#allow-insecure-localhost

curl --cacert certs/localhost.crt https://localhost/recipes
curl -k https://localhost/recipes

sudo nano /etc/hosts
127.0.0.1 api.recipes.io
ping api.recipes.io