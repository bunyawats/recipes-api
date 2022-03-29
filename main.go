// Recipes API
//
// This is a sample recipes API. You can find out more about the API at https://github.com/PacktPublishing/Building-Distributed-Applications-in-Gin.
//
//  Schemes: http
//  Host: localhost:8080
//  BasePath: /
//  Version: 1.0.0
//  Contact: John Doe<john.doe@example.com> http://john.doe.com
//
//  Consumes:
//  - application/json
//
//  Produces:
//  - application/json
// swagger:meta
package main

import (
	"context"
	"encoding/json"
	"fmt"
	handler "github.com/bunyawats/recipes-api/handlers"
	"github.com/bunyawats/recipes-api/models"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"log"
	"os"
)

var recipesHandler *handler.RecipesHandler

func InitLoad() {

	var databaseUri = os.Getenv("MONGO_URI")
	var databaseName = os.Getenv("MONGO_DATABASE")
	const collectionName = "recipes"

	fmt.Println("databaseUri", databaseUri)
	fmt.Println("databaseName", databaseName)
	fmt.Println("collectionName", collectionName)

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
	collection := client.Database(databaseName).Collection(collectionName)
	insertManyResult, err := collection.InsertMany(ctx, lisOfRecipes)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Inserted recipes: ", len(insertManyResult.InsertedIDs))
}

func init() {

	var databaseUri = os.Getenv("MONGO_URI")
	var databaseName = os.Getenv("MONGO_DATABASE")
	var xApiKey = os.Getenv("X-API-KEY")
	const collectionName = "recipes"

	fmt.Println("databaseUri", databaseUri)
	fmt.Println("databaseName", databaseName)
	fmt.Println("xApiKey", xApiKey)
	fmt.Println("collectionName", collectionName)

	ctx := context.Background()
	client, err := mongo.Connect(
		ctx,
		options.Client().ApplyURI(databaseUri),
	)
	if err != nil {
		log.Fatal("Connect to MongoDB failed:", err.Error())
	}
	collection := client.Database(databaseName).Collection(collectionName)
	log.Println("Connected to MongoDB")

	redisClient := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})
	status := redisClient.Ping()
	fmt.Println(status)

	recipesHandler = handler.NewRecipesHandler(
		ctx,
		collection,
		redisClient,
		xApiKey,
	)

}

func main() {
	router := gin.Default()
	router.POST("/recipes", recipesHandler.NewRecipeHandler)
	router.GET("/recipes", recipesHandler.ListRecipesHandler)
	router.PUT("/recipes/:id", recipesHandler.UpdateRecipeHandler)
	router.DELETE("/recipes/:id", recipesHandler.DeleteRecipesHandler)
	//	router.GET("/recipes/search", SearchRecipesHandler)
	err := router.Run()
	if err != nil {
		log.Fatal(err)
	}
}
