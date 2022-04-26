// Recipes API
//
// This is a sample recipes API. You can find out more about the API at https://github.com/PacktPublishing/Building-Distributed-Applications-in-Gin.
//
//	Schemes: http
//  Host: api.recipes.io:8080
//	BasePath: /
//	Version: 1.0.0
//	Contact: Mohamed Labouardy <mohamed@labouardy.com> https://labouardy.com
//  SecurityDefinitions:
//  api_key:
//    type: apiKey
//    name: Authorization
//    in: header
//
//	Consumes:
//	- application/json
//
//	Produces:
//	- application/json
// swagger:meta
package main

import (
	"context"
	"encoding/json"
	"fmt"
	handler "github.com/bunyawats/recipes-api/handlers"
	"github.com/bunyawats/recipes-api/models"
	"github.com/gin-contrib/sessions"
	redisStore "github.com/gin-contrib/sessions/redis"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"golang.org/x/crypto/bcrypt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
)

const (
	mongoUriEnv           = "MONGO_URI"
	mongoDatabaseEnv      = "MONGO_DATABASE"
	collectionNameRecipes = "recipes"
	collectionNameUsers   = "users"
	apiKey                = "X_API_KEY"
	jwtSecretKey          = "JWT_SECRET"
	redisUriEnv           = "REDIS_URI"
	sessionKey            = "recipes_api"
)

type (
	StaticRecipe struct {
		ID          string       `json:"ID"`
		Name        string       `json:"name"`
		Ingredients []Ingredient `json:"ingredients"`
		Steps       []string     `json:"steps"`
		Picture     string       `json:"imageURL"`
	}

	Ingredient struct {
		Quantity string `json:"quantity"`
		Name     string `json:"name"`
		Type     string `json:"type"`
	}
)

var (
	authHandler    *handler.AuthHandler
	recipesHandler *handler.RecipesHandler
	xApiKey        string
	store          sessions.Store

	staticRecipes []StaticRecipe
)

func initLoadRecipes() {

	var databaseUri = os.Getenv(mongoUriEnv)
	var databaseName = os.Getenv(mongoDatabaseEnv)

	fmt.Println("databaseUri", databaseUri)
	fmt.Println("databaseName", databaseName)
	fmt.Println("collectionNameRecipes", collectionNameRecipes)
	fmt.Println("collectionNameUser", collectionNameUsers)

	recipes := make([]models.Recipe, 0)
	file, _ := os.ReadFile("recipes.json")
	_ = json.Unmarshal([]byte(file), &recipes)

	ctx := context.Background()
	client, err := mongo.Connect(
		ctx,
		options.Client().ApplyURI(databaseUri),
	)
	log.Println("Connected to MongoDB")

	var lisOfRecipes []interface{}
	for _, recipe := range recipes {
		lisOfRecipes = append(lisOfRecipes, recipe)
	}
	collection := client.Database(databaseName).Collection(collectionNameRecipes)
	insertManyResult, err := collection.InsertMany(ctx, lisOfRecipes)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Inserted recipes: ", len(insertManyResult.InsertedIDs))
}

//func AuthMiddleware() gin.HandlerFunc {
//	return func(c *gin.Context) {
//		if c.GetHeader(apiKey) != xApiKey {
//			c.AbortWithStatus(http.StatusUnauthorized)
//		}
//		c.Next()
//	}
//}

func initLoadUser() {

	var databaseUri = os.Getenv(mongoUriEnv)
	var databaseName = os.Getenv(mongoDatabaseEnv)

	fmt.Println("databaseUri", databaseUri)
	fmt.Println("databaseName", databaseName)
	fmt.Println("collectionNameUser", collectionNameUsers)

	users := map[string]string{
		"admin":    "password",
		"bunyawat": "password",
	}
	ctx := context.Background()
	client, _ := mongo.Connect(ctx, options.Client().ApplyURI(databaseUri))
	if err := client.Ping(
		context.TODO(),
		readpref.Primary(),
	); err != nil {
		log.Fatal(err)
	}
	collectionUsers := client.Database(databaseName).Collection(collectionNameUsers)

	for username, password := range users {
		fmt.Println(username, password)

		hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
		hsPassword := string(hash)
		fmt.Println("hsPassword", hsPassword)

		collectionUsers.InsertOne(
			ctx,
			bson.M{
				"username": username,
				"password": hsPassword,
			},
		)
	}
}

func _main() {
	initLoadUser()
}

//func IndexHandler(c *gin.Context) {
//	c.File("index.html")
//}

func init() {

	var databaseUri = os.Getenv(mongoUriEnv)
	var databaseName = os.Getenv(mongoDatabaseEnv)
	xApiKey = os.Getenv(apiKey)
	var jwtSecret = os.Getenv(jwtSecretKey)
	var redisUri = os.Getenv(redisUriEnv)

	fmt.Println("databaseUri", databaseUri)
	fmt.Println("databaseName", databaseName)
	fmt.Println("collectionNameRecipes", collectionNameRecipes)
	fmt.Println("collectionNameUser", collectionNameUsers)
	fmt.Println("xApiKey", xApiKey)
	fmt.Println("jwtSecret", jwtSecret)
	fmt.Println("redisUri", redisUri)

	ctx := context.Background()
	client, err := mongo.Connect(
		ctx,
		options.Client().ApplyURI(databaseUri),
	)
	if err != nil {
		log.Fatal("Connect to MongoDB failed:", err.Error())
	}
	collectionRecipes := client.Database(databaseName).Collection(collectionNameRecipes)
	collectionUsers := client.Database(databaseName).Collection(collectionNameUsers)
	log.Println("Connected to MongoDB")

	redisClient := redis.NewClient(&redis.Options{
		Addr:     redisUri,
		Password: "",
		DB:       0,
	})
	status := redisClient.Ping()
	fmt.Println(status)

	recipesHandler = handler.NewRecipesHandler(
		ctx,
		collectionRecipes,
		redisClient,
	)
	authHandler = handler.NewAuthHandler(ctx, collectionUsers)
	store, err = redisStore.NewStore(
		10,
		"tcp",
		redisUri,
		"",
		[]byte("secret"),
	)
	if err != nil {
		log.Fatal("Connect to Redis failed:", err.Error())
	}

	//staticRecipes = make([]StaticRecipe, 0)
	//file, _ := os.ReadFile("recipes.json")
	//_ = json.Unmarshal([]byte(file), &staticRecipes)
	//fmt.Println("staticRecipes:", staticRecipes)

	staticRecipes = make([]StaticRecipe, 0)
	json.Unmarshal(Assets.Files["/recipes.json"].Data,
		&staticRecipes)
}

func IndexHandler(c *gin.Context) {

	c.HTML(http.StatusOK, "/templates/index.tmpl", gin.H{
		"recipes": staticRecipes,
	})
}

func RecipeHandler(c *gin.Context) {
	for _, recipe := range staticRecipes {
		if recipe.ID == c.Param("id") {
			c.HTML(http.StatusOK, "/templates/recipe.tmpl", gin.H{
				"recipe": recipe,
			})
			return
		}
	}
	c.File("404.html")
}

func loadTemplate() (*template.Template, error) {
	t := template.New("")
	for name, file := range Assets.Files {
		if file.IsDir() || !strings.HasSuffix(name,
			".tmpl") {
			continue
		}
		h, err := ioutil.ReadAll(file)
		if err != nil {
			return nil, err
		}
		t, err = t.New(name).Parse(string(h))
		if err != nil {
			return nil, err
		}
	}
	return t, nil
}

func StaticHandler(c *gin.Context) {
	filepath := c.Param("filepath")
	data := Assets.Files["/assets"+filepath].Data
	c.Writer.Write(data)
}

func main() {

	t, err := loadTemplate()
	if err != nil {
		panic(err)
	}

	router := gin.Default()
	router.Use(sessions.Sessions(sessionKey, store))

	//router.Static("/assets", "./assets")
	//router.LoadHTMLGlob("templates/*")
	router.SetHTMLTemplate(t)
	router.GET("/assets/*filepath", StaticHandler)

	router.GET("/", IndexHandler)
	router.GET("/recipes/:id", RecipeHandler)

	router.GET("/recipes", recipesHandler.ListRecipesHandler)
	router.POST("/signin", authHandler.SignInHandler)
	router.POST("/refresh", authHandler.RefreshHandler)
	router.POST("/signout", authHandler.SignOutHandler)

	authorized := router.Group("/")
	authorized.Use(authHandler.AuthMiddleware())
	{
		authorized.POST("/recipes", recipesHandler.NewRecipeHandler)
		authorized.PUT("/recipes/:id", recipesHandler.UpdateRecipeHandler)
		authorized.DELETE("/recipes/:id", recipesHandler.DeleteRecipesHandler)
		//	router.GET("/recipes/search", SearchRecipesHandler)
	}

	err = router.RunTLS(
		":443",
		"certs/localhost.crt",
		"certs/localhost.key",
	)
	//err := router.Run()
	if err != nil {
		log.Fatal(err)
	}
}
