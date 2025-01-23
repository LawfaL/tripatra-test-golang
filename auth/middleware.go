package auth

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/lawfal/go-graph-tripatra/config"
	"github.com/lawfal/go-graph-tripatra/repository"
	"github.com/lawfal/go-graph-tripatra/utils"
)

func GetAuth(userService repository.UserRepository, redisClient *redis.Client) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var access_token string
		cookie, err := ctx.Cookie("access_token")

		authorizationHeader := ctx.Request.Header.Get("Authorization")
		fields := strings.Fields(authorizationHeader)

		if len(fields) != 0 && fields[0] == "Bearer" {
			access_token = fields[1]
		} else if err == nil {
			access_token = cookie
		}

		if access_token == "" {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"status": "fail", "message": "You are not logged in"})
			return
		}

		envConf, _ := config.LoadConfig(".")
		td, err := utils.ValidateToken(access_token, envConf.AccessTokenPublicKey)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"status": "fail", "message": err.Error()})
			return
		}

		userid, err := redisClient.Get(ctx, td.TokenUuid).Result()
		if err == redis.Nil {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"status": "fail", "message": "Token is invalid or session has expired"})
			return
		}

		user, err := userService.FindUserById(userid)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"status": "fail", "message": "The user no logger exists"})
			return
		}

		ctx.Set("currentUser", user)
		ctx.Set("access_token_uuid", td.TokenUuid)
		ctx.Next()
	}
}
