package jwt

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/IceWhaleTech/CasaOS/model"
	"github.com/IceWhaleTech/CasaOS/utils"
	"github.com/sirupsen/logrus"
)

var (
	once           sync.Once
	secretKey      []byte
	secretFilePath = "/var/lib/casaos/jwt_secret.key" // dostosuj ścieżkę
)

// InitJWTSecret inicjalizuje lub ładuje sekret JWT
func InitJWTSecret() error {
	var err error
	once.Do(func() {
		secretKey, err = loadOrGenerateSecret()
	})
	return err
}

// loadOrGenerateSecret ładuje istniejący sekret lub generuje nowy
func loadOrGenerateSecret() ([]byte, error) {
	// Sprawdź czy plik z sekretem istnieje
	if _, err := os.Stat(secretFilePath); err == nil {
		// Wczytaj istniejący sekret
		secret, err := os.ReadFile(secretFilePath)
		if err != nil {
			return nil, err
		}
		logrus.Info("JWT secret loaded from file")
		return secret, nil
	}

	// Wygeneruj nowy, losowy sekret (32 bajty = 256 bitów)
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		return nil, err
	}

	// Zapisz sekret do pliku
	if err := os.MkdirAll(filepath.Dir(secretFilePath), 0755); err != nil {
		return nil, err
	}
	if err := os.WriteFile(secretFilePath, secret, 0600); err != nil {
		return nil, err
	}

	logrus.Info("New JWT secret generated and saved")
	return secret, nil
}

// GetSecret zwraca aktualny sekret JWT
func GetSecret() []byte {
	if len(secretKey) == 0 {
		if err := InitJWTSecret(); err != nil {
			logrus.Fatal("Failed to initialize JWT secret:", err)
		}
	}
	return secretKey
}

// GenerateToken generuje nowy token JWT
func GenerateToken(user *model.User) (string, error) {
	claims := &model.Claims{
		UserID:   user.ID,
		Username: user.Username,
		Role:     user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(GetSecret())
}

// ValidateToken waliduje token JWT
func ValidateToken(tokenString string) (*model.Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &model.Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return GetSecret(), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*model.Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}

// RefreshToken odświeża token (przedłuża ważność)
func RefreshToken(tokenString string) (string, error) {
	claims, err := ValidateToken(tokenString)
	if err != nil {
		return "", err
	}

	// Przedłuż ważność o kolejne 24h
	claims.ExpiresAt = jwt.NewNumericDate(time.Now().Add(24 * time.Hour))
	
	newToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return newToken.SignedString(GetSecret())
}
