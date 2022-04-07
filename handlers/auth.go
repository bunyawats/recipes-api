package handlers

import (
	"context"
	"fmt"
	"github.com/auth0-community/go-auth0"
	"github.com/bunyawats/recipes-api/models"
	"github.com/dgrijalva/jwt-go"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/rs/xid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/crypto/bcrypt"
	"gopkg.in/square/go-jose.v2"
	"net/http"
	"os"
	"time"
)

const (
	jwtSecretKey = "JWT_SECRET"
	authorKey    = "Authorization"
)

type AuthHandler struct {
	collection *mongo.Collection
	ctx        context.Context
}

type Claims struct {
	Username string `json:"username"`
	jwt.StandardClaims
}

type JWTOutput struct {
	Token   string    `json:"token"`
	Expires time.Time `json:"expires"`
}

func NewAuthHandler(ctx context.Context, collection *mongo.Collection) *AuthHandler {
	return &AuthHandler{
		collection: collection,
		ctx:        ctx,
	}
}

func (handler *AuthHandler) SignInForJwtHandler(c *gin.Context) {

	// validate request
	var user models.User
	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	// find user by username
	cur := handler.collection.FindOne(
		handler.ctx,
		bson.M{
			"username": user.Username,
		},
	)
	if cur.Err() != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Invalid username or password",
		})
		return
	}

	// compare hash and password
	var foundUser models.User
	cur.Decode(&foundUser)
	fmt.Println("foundUser", foundUser)
	byteHash := []byte(foundUser.Password)
	bytePlainPwd := []byte(user.Password)
	err := bcrypt.CompareHashAndPassword(byteHash, bytePlainPwd)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Invalid username or password",
		})
		return
	}

	// create jwt token
	expirationTime := time.Now().Add(10 * time.Minute)
	claims := &Claims{
		Username: user.Username,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: expirationTime.Unix(),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	jwtSecret := os.Getenv(jwtSecretKey)
	tokenString, err := token.SignedString([]byte(jwtSecret))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}
	jwtOutput := JWTOutput{
		Token:   tokenString,
		Expires: expirationTime,
	}

	c.JSON(http.StatusOK, jwtOutput)

}

// swagger:operation POST /signin auth signIn
// Login with username and password
// ---
// produces:
// - application/json
// responses:
//     '200':
//         description: Successful operation
//     '401':
//         description: Invalid credentials
func (handler *AuthHandler) SignInHandler(c *gin.Context) {

	// validate request
	var user models.User
	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	// find user by username
	cur := handler.collection.FindOne(
		handler.ctx,
		bson.M{
			"username": user.Username,
		},
	)
	if cur.Err() != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Invalid username or password",
		})
		return
	}

	// compare hash and password
	var foundUser models.User
	cur.Decode(&foundUser)
	fmt.Println("foundUser", foundUser)
	byteHash := []byte(foundUser.Password)
	bytePlainPwd := []byte(user.Password)
	err := bcrypt.CompareHashAndPassword(byteHash, bytePlainPwd)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Invalid username or password",
		})
		return
	}

	sessionToken := xid.New().String()
	session := sessions.Default(c)
	session.Set("username", user.Username)
	session.Set("token", sessionToken)
	session.Save()

	c.JSON(http.StatusOK, gin.H{
		"message": "User siged in",
	})
}

func (handler *AuthHandler) SignOutHandler(c *gin.Context) {
	session := sessions.Default(c)
	session.Clear()
	session.Save()
	c.JSON(http.StatusOK, gin.H{
		"message": "Signed out...",
	})
}

// swagger:operation POST /refresh auth refresh
// Get new token in exchange for an old one
// ---
// produces:
// - application/json
// responses:
//     '200':
//         description: Successful operation
//     '400':
//         description: Token is new and doesn't need
//                      a refresh
//     '401':
//         description: Invalid credentials
func (handler *AuthHandler) RefreshHandler(c *gin.Context) {
	tokenValue := c.GetHeader(authorKey)
	claims := &Claims{}
	tkn, err := jwt.ParseWithClaims(
		tokenValue,
		claims,
		func(token *jwt.Token) (interface{}, error) {
			return []byte(os.Getenv(jwtSecretKey)), nil
		},
	)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": err.Error(),
		})
		return
	}
	if tkn == nil || !tkn.Valid {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Invalid token",
		})
		return
	}
	if time.Unix(claims.ExpiresAt, 0).Sub(time.Now()) > 30*time.Second {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Token is not expired yet",
		})
		return
	}
	expirationTime := time.Now().Add(5 * time.Minute)
	claims.ExpiresAt = expirationTime.Unix()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	jwtSecret := os.Getenv(jwtSecretKey)
	tokenString, err := token.SignedString([]byte(jwtSecret))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	jwtOutput := JWTOutput{
		Token:   tokenString,
		Expires: expirationTime,
	}
	c.JSON(http.StatusOK, jwtOutput)
}

func AuthJwtMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenValue := c.GetHeader(authorKey)
		claims := &Claims{}
		tkn, err := jwt.ParseWithClaims(
			tokenValue,
			claims,
			func(token *jwt.Token) (interface{}, error) {
				return []byte(os.Getenv(jwtSecretKey)), nil
			},
		)
		if err != nil {
			c.AbortWithStatus(http.StatusUnauthorized)
		}
		if tkn == nil || !tkn.Valid {
			c.AbortWithStatus(http.StatusUnauthorized)
		}
		c.Next()
	}
}

func AuthSessionMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		session := sessions.Default(c)
		sessionToken := session.Get("token")
		if sessionToken == nil {
			c.JSON(http.StatusForbidden, gin.H{
				"message": "Not logged",
			})
			c.Abort()
		}
		c.Next()
	}
}

func (handler *AuthHandler) AuthMiddleware() gin.HandlerFunc {

	auth0DomainName := os.Getenv("AUTH0_DOMAIN")
	auth0ApiIdentifier := os.Getenv("AUTH0_API_IDENTIFIER")

	fmt.Println("auth0DomainName", auth0DomainName)
	fmt.Println("auth0ApiIdentifier", auth0ApiIdentifier)

	return func(c *gin.Context) {
		var auth0Domain = "https://" + auth0DomainName + "/"
		client := auth0.NewJWKClient(
			auth0.JWKClientOptions{
				URI: auth0Domain + ".well-known/jwks.json",
			},
			nil,
		)

		configuration := auth0.NewConfiguration(
			client,
			[]string{auth0ApiIdentifier},
			auth0Domain,
			jose.RS256)
		validator := auth0.NewValidator(configuration, nil)
		_, err := validator.ValidateRequest(c.Request)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"message": err.Error(),
			})
			c.Abort()
			return
		}
		c.Next()
	}
}
