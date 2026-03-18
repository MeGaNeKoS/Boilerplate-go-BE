package utils

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"project-template/infrastructure/config"
	"reflect"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

const LoginExpirationDuration = time.Hour * 24

var randomReader = rand.Reader

// parseWithClaims is a wrapper around jwt.ParseWithClaims to allow overriding
// it in tests.
var parseWithClaims = func(tokenString string, claims jwt.Claims, keyFunc jwt.Keyfunc, options ...jwt.ParserOption) (*jwt.Token, error) {
	return jwt.ParseWithClaims(tokenString, claims, keyFunc, options...)
}

type JWTService interface {
	SignJWT(data map[string]interface{}) (string, error)
	ParseJWT(tokenString string, claims jwt.Claims) error
	GenerateRefreshToken(length int) (string, error)
}

type jwtService struct {
	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey
}

var (
	jwtInstance *jwtService
	jwtOnce     sync.Once
)

// InitializeJWTService prepares the JWT helper with optional private and public
// keys. It is safe to call multiple times; initialization occurs only once.
func InitializeJWTService(private, public bool) error {
	var err error
	jwtOnce.Do(func() {
		jwtInstance, err = newJWTService(private, public)
	})
	return err
}

// GetJWTService returns the singleton JWTService instance. The service must be
// initialized before calling this function.
func GetJWTService() JWTService {
	if jwtInstance == nil {
		panic("JWTService is not initialized. Call InitializeJWTService first.")
	}
	return jwtInstance
}

func newJWTService(private, public bool) (*jwtService, error) {
	service := &jwtService{}

	if private {
		privateKeyData, err := os.ReadFile(config.Cfg.Server.JWT.PrivateKey)
		if err != nil {
			return nil, fmt.Errorf("could not read private key: %w", err)
		}
		service.privateKey, err = jwt.ParseRSAPrivateKeyFromPEM(privateKeyData)
		if err != nil {
			return nil, fmt.Errorf("could not parse private key: %w", err)
		}
	}

	if public {
		publicKeyData, err := os.ReadFile(config.Cfg.Server.JWT.PublicKey)
		if err != nil {
			return nil, fmt.Errorf("could not read public key: %w", err)
		}
		service.publicKey, err = jwt.ParseRSAPublicKeyFromPEM(publicKeyData)
		if err != nil {
			return nil, fmt.Errorf("could not parse public key: %w", err)
		}
	}

	return service, nil
}

// Claims extends jwt.RegisteredClaims with environment information and arbitrary
// additional fields that will be encoded into the token.
type Claims struct {
	Environment string `json:"environment"`
	jwt.RegisteredClaims
	Additional map[string]interface{} `json:"-"`
}

// MarshalJSON implements custom JSON encoding for Claims so that additional
// fields are included in the output.
func (c Claims) MarshalJSON() ([]byte, error) {
	result := map[string]interface{}{
		"environment": c.Environment,
		"iss":         c.Issuer,
		"exp":         c.ExpiresAt.Unix(),
	}

	for key, value := range c.Additional {
		result[key] = value
	}

	return json.Marshal(result)
}

// GenerateRefreshToken creates a random string of the given length suitable for
// use as a refresh token.
func (s *jwtService) GenerateRefreshToken(length int) (string, error) {
	bytes := make([]byte, length)
	_, err := io.ReadFull(randomReader, bytes)
	if err != nil {
		return "", err
	}

	token := base64.URLEncoding.EncodeToString(bytes)
	return token[:length], nil
}

// SignJWT creates a signed JWT containing the provided data.
func (s *jwtService) SignJWT(data map[string]interface{}) (string, error) {
	if s.privateKey == nil {
		return "", fmt.Errorf("private key not initialized")
	}

	claims := &Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    config.Cfg.AppName,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(LoginExpirationDuration)),
		},
		Environment: config.Cfg.Server.Environment,
		Additional:  data,
	}

	accessToken := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	accessTokenString, err := accessToken.SignedString(s.privateKey)
	if err != nil {
		return "", err
	}

	return accessTokenString, nil
}

// ParseJWT validates the token string and populates the provided claims
// structure if the token is valid.
func (s *jwtService) ParseJWT(tokenString string, claims jwt.Claims) error {
	if s.publicKey == nil {
		return fmt.Errorf("public key not initialized")
	}

	token, err := parseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.publicKey, nil
	})

	if err != nil {
		return err
	}

	if !token.Valid {
		return fmt.Errorf("invalid token")
	}

	type environmentClaims interface {
		GetEnvironment() string
	}
	if ec, ok := claims.(environmentClaims); ok {
		if ec.GetEnvironment() != config.Cfg.Server.Environment {
			return fmt.Errorf("invalid environment: got %q, want %q", ec.GetEnvironment(), config.Cfg.Server.Environment)
		}
	} else {
		val := reflect.ValueOf(claims).Elem()
		envField := val.FieldByName("Environment")
		if !envField.IsValid() || envField.Kind() != reflect.String {
			return fmt.Errorf("claims type %T has no string Environment field", claims)
		}
		if envField.String() != config.Cfg.Server.Environment {
			return fmt.Errorf("invalid environment: got %q, want %q", envField.String(), config.Cfg.Server.Environment)
		}
	}

	return nil
}
