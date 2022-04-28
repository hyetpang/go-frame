package common

import (
	"strconv"
	"time"

	"github.com/HyetPang/go-frame/pkgs/logs"
	"github.com/golang-jwt/jwt"
	"go.uber.org/zap"
)

const (
	TOKEN_SKEY        = "hero_ultra_sdk_center_go_token"
	TOKEN_EXPIRE_TIME = 86400 * 7 // 7天内有效
	// PASSWORD_PREFIX   = "xydYDJH83DJHRDF6A=k"
	// SIGN_KEY = "ultrasdk.go_token_sign_key"
	SIGN_KEY = "21232f297a57a5a743894a0e4a801fc3"
)

type TokenClaims struct {
	UserId int `json:"user_id"`
	jwt.StandardClaims
}

// 生成token
func GetToken(userId int, signKey string) (string, error) {
	if len(signKey) < 1 {
		signKey = SIGN_KEY
	}
	mySigningKey := []byte(signKey)
	sign := Md5(strconv.Itoa(userId) + TOKEN_SKEY)
	expired := int64(time.Now().Unix() + TOKEN_EXPIRE_TIME)
	acc_token := ""
	tokenClaims := TokenClaims{
		UserId: userId,
		StandardClaims: jwt.StandardClaims{
			NotBefore: int64(time.Now().Unix() - TOKEN_EXPIRE_TIME),
			ExpiresAt: expired,
			Issuer:    strconv.Itoa(userId),
			Subject:   sign,
		},
	}
	acc_token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, tokenClaims).SignedString(mySigningKey)
	if err != nil {
		return acc_token, err
	}

	return acc_token, nil
}

// token解析
func ParseToken(tokenString, signKey string) (*TokenClaims, error) {
	if len(signKey) < 1 {
		signKey = SIGN_KEY
	}
	jwtKey := String2Byte(signKey)
	tokenClaims, err := jwt.ParseWithClaims(tokenString, new(TokenClaims), func(token *jwt.Token) (interface{}, error) {
		return jwtKey, nil
	})
	if err != nil {
		logs.Error("token解析失败", zap.Error(err))
		return nil, err
	}
	claims := tokenClaims.Claims.(*TokenClaims)
	if err := claims.Valid(); err != nil {
		logs.Error("token验证失败", zap.Error(err))
		return nil, err
	}
	return claims, nil
}
