package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type User struct {
	Username string
	Password string
	Role     string
}

type JWTService struct {
	secretKey []byte
	users     []User
}

type Claims struct {
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

func NewJWTService(secretKey string, users []User) *JWTService {
	return &JWTService{
		secretKey: []byte(secretKey),
		users:     users,
	}
}

func (j *JWTService) GenerateToken(username, password string) (string, error) {
	// Find user
	var user *User
	for _, u := range j.users {
		if u.Username == username && u.Password == password {
			user = &u
			break
		}
	}

	if user == nil {
		return "", errors.New("invalid credentials")
	}

	// Create claims
	claims := &Claims{
		Username: user.Username,
		Role:     user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}

	// Create token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(j.secretKey)
}

func (j *JWTService) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return j.secretKey, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}

func (j *JWTService) GetUserFromToken(tokenString string) (*User, error) {
	claims, err := j.ValidateToken(tokenString)
	if err != nil {
		return nil, err
	}

	// Find user by username
	for _, u := range j.users {
		if u.Username == claims.Username {
			return &u, nil
		}
	}

	return nil, errors.New("user not found")
}
