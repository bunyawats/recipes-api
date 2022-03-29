package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/bunyawats/recipes-api/models"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"log"
	"net/http"
	"strings"
	"time"
)

const recipes_key = "recipes"

var recipes []models.Recipe
var err error

type RecipesHandler struct {
	collection  *mongo.Collection
	ctx         context.Context
	redisClient *redis.Client
	xApiKey     string
}

func NewRecipesHandler(
	ctx context.Context,
	collection *mongo.Collection,
	redisClient *redis.Client,
	xApiKey string,
) *RecipesHandler {
	return &RecipesHandler{
		collection,
		ctx,
		redisClient,
		xApiKey,
	}
}

// swagger:operation GET /recipes recipes listRecipes
// Returns list of recipes
// ---
// produces:
// - application/json
// responses:
//     '200':
//         description: Successful operation
func (handler *RecipesHandler) ListRecipesHandler(c *gin.Context) {

	val, err := handler.redisClient.Get(recipes_key).Result()
	if err == redis.Nil {

		log.Printf("Request to MongoDB")

		cur, err := handler.collection.Find(handler.ctx, bson.M{})
		if err != nil {
			log.Println("error: ", err.Error())
			c.JSON(http.StatusInternalServerError,
				gin.H{
					"error": err.Error(),
				},
			)
			return
		}
		defer func(cur *mongo.Cursor, ctx context.Context) {
			err := cur.Close(ctx)
			if err != nil {
				log.Println("error: ", err.Error())
			}
		}(cur, handler.ctx)

		recipes := make([]models.Recipe, 0)
		for cur.Next(handler.ctx) {
			var recipe models.Recipe
			_ = cur.Decode(&recipe)
			recipes = append(recipes, recipe)
		}

		// cache to redis database
		data, _ := json.Marshal(recipes)
		handler.redisClient.Set(recipes_key, data, 0)

		c.JSON(http.StatusOK, recipes)

	} else if err != nil {
		c.JSON(http.StatusInternalServerError,
			gin.H{
				"error": err.Error(),
			})
	} else {
		log.Printf("Request to Redis")
		recipes := make([]models.Recipe, 0)
		json.Unmarshal([]byte(val), &recipes)

		c.JSON(http.StatusOK, recipes)
	}

}

// swagger:operation POST /recipes recipes newRecipe
// Update an existing recipe
// ---
// parameters:
// - name: id
//   in: path
//   description: ID of the recipe
//   required: true
//   type: string
// produces:
// - application/json
// responses:
//     '200':
//         description: Successful operation
//     '400':
//         description: Invalid input
//     '404':
//         description: Invalid recipe ID
func (handler *RecipesHandler) NewRecipeHandler(c *gin.Context) {

	// security validation
	if c.GetHeader("X-API-KEY") != handler.xApiKey {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "API key not provided or invalid",
		})
		return
	}

	// validate request
	var recipe models.Recipe
	if err := c.ShouldBindJSON(&recipe); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	// insert to database
	recipe.ID = primitive.NewObjectID()
	recipe.PublishedAt = time.Now()
	_, err := handler.collection.InsertOne(handler.ctx, recipe)

	// response the result
	if err != nil {
		fmt.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Error while inserting a new recipe",
		})
		return
	}

	// clear redis cache
	log.Println("Remove data from Redis")
	handler.redisClient.DebugObject(recipes_key)

	c.JSON(http.StatusOK, recipe)
}

// swagger:operation PUT /recipes/{id} recipes updateRecipe
// Update an existing recipe
// ---
// parameters:
// - name: id
//   in: path
//   description: ID of the recipe
//   required: true
//   type: string
// produces:
// - application/json
// responses:
//     '200':
//         description: Successful operation
//     '400':
//         description: Invalid input
//     '404':
//         description: Invalid recipe ID
func (handler *RecipesHandler) UpdateRecipeHandler(c *gin.Context) {
	// validate request
	id := c.Param("id")
	var recipe models.Recipe
	if err := c.ShouldBindJSON(&recipe); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	// update to database
	objectId, _ := primitive.ObjectIDFromHex(id)
	_, err = handler.collection.UpdateOne(
		handler.ctx,
		bson.M{
			"_id": objectId,
		},
		bson.D{
			{
				"$set", bson.D{
					{"name", recipe.Name},
					{"instructions", recipe.Instructions},
					{"ingredients", recipe.Ingredients},
					{"tags", recipe.Tags},
				},
			},
		},
	)

	// response the result
	if err != nil {
		fmt.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	// clear redis cache
	log.Println("Remove data from Redis")
	handler.redisClient.DebugObject(recipes_key)

	c.JSON(http.StatusOK, gin.H{
		"message": "Recipe has been updated",
	})
}

// swagger:operation DELETE /recipes/{id} recipes deleteRecipe
// Update an existing recipe
// ---
// parameters:
// - name: id
//   in: path
//   description: ID of the recipe
//   required: true
//   type: string
// produces:
// - application/json
// responses:
//     '200':
//         description: Successful operation
//     '400':
//         description: Invalid input
//     '404':
//         description: Invalid recipe ID
func (handler *RecipesHandler) DeleteRecipesHandler(c *gin.Context) {
	// validate request
	id := c.Param("id")
	objectId, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
	}

	// delete from database
	_, err = handler.collection.DeleteOne(handler.ctx, bson.M{
		"_id": objectId,
	})

	// response the result
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	// clear redis cache
	log.Println("Remove data from Redis")
	handler.redisClient.DebugObject(recipes_key)

	c.JSON(http.StatusOK, gin.H{
		"message": "Recipe has been deleted",
	})

}

// swagger:operation GET /recipes/search recipes searchRecipe
// Update an existing recipe
// ---
// parameters:
// - name: id
//   in: path
//   description: ID of the recipe
//   required: true
//   type: string
// produces:
// - application/json
// responses:
//     '200':
//         description: Successful operation
//     '400':
//         description: Invalid input
//     '404':
//         description: Invalid recipe ID
func (handler *RecipesHandler) SearchRecipesHandler(c *gin.Context) {
	tag := c.Query("tag")
	listOfRecipes := make([]models.Recipe, 0)
	for i := 0; i < len(recipes); i++ {
		found := false
		for _, t := range recipes[i].Tags {

			if strings.EqualFold(t, tag) {
				found = true
			}
		}
		if found {
			listOfRecipes = append(listOfRecipes, recipes[i])
		}
	}
	c.JSON(http.StatusOK, listOfRecipes)

}
