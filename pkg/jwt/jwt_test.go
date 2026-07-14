package jwt

import (
	"os"
	"testing"
	"time"

	"github.com/IceWhaleTech/CasaOS/model"
	"github.com/golang-jwt/jwt/v4"
	"github.com/stretchr/testify/assert"
)

func TestJWTSecretGeneration(t *testing.T) {
	// Ustaw tymczasową ścieżkę dla testów
	testSecretPath := "/tmp/test_jwt_secret.key"
	secretFilePath = testSecretPath
	defer os.Remove(testSecretPath)

	// Inicjalizacja
	err := InitJWTSecret()
	assert.NoError(t, err)
	assert.NotEmpty(t, GetSecret())

	// Sprawdź czy plik został utworzony
	_, err = os.Stat(testSecretPath)
	assert.NoError(t, err)

	// Sprawdź czy sekret jest stały (długość 32 bajty)
	assert.Equal(t, 32, len(GetSecret()))
}

func TestGenerateAndValidateToken(t *testing.T) {
	// Ustaw tymczasową ścieżkę dla testów
	testSecretPath := "/tmp/test_jwt_secret.key"
	secretFilePath = testSecretPath
	defer os.Remove(testSecretPath)

	err := InitJWTSecret()
	assert.NoError(t, err)

	// Utwórz testowego użytkownika
	user := &model.User{
		ID:       1,
		Username: "testuser",
		Role:     "admin",
	}

	// Generuj token
	token, err := GenerateToken(user)
	assert.NoError(t, err)
	assert.NotEmpty(t, token)

	// Waliduj token
	claims, err := ValidateToken(token)
	assert.NoError(t, err)
	assert.Equal(t, user.ID, claims.UserID)
	assert.Equal(t, user.Username, claims.Username)
	assert.Equal(t, user.Role, claims.Role)

	// Sprawdź czy token wygasa
	claims.ExpiresAt = jwt.NewNumericDate(time.Now().Add(-1 * time.Hour))
	expiredToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	expiredTokenString, err := expiredToken.SignedString(GetSecret())
	assert.NoError(t, err)

	_, err = ValidateToken(expiredTokenString)
	assert.Error(t, err)
}

func TestInvalidToken(t *testing.T) {
	// Ustaw tymczasową ścieżkę dla testów
	testSecretPath := "/tmp/test_jwt_secret.key"
	secretFilePath = testSecretPath
	defer os.Remove(testSecretPath)

	err := InitJWTSecret()
	assert.NoError(t, err)

	// Próba walidacji nieprawidłowego tokena
	_, err = ValidateToken("invalid.token.string")
	assert.Error(t, err)

	// Token podpisany innym kluczem
	claims := &model.Claims{
		UserID:   1,
		Username: "test",
		Role:     "user",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	wrongSecret := []byte("wrong-secret-key")
	tokenString, err := token.SignedString(wrongSecret)
	assert.NoError(t, err)

	_, err = ValidateToken(tokenString)
	assert.Error(t, err)
}

func TestRefreshToken(t *testing.T) {
	// Ustaw tymczasową ścieżkę dla testów
	testSecretPath := "/tmp/test_jwt_secret.key"
	secretFilePath = testSecretPath
	defer os.Remove(testSecretPath)

	err := InitJWTSecret()
	assert.NoError(t, err)

	user := &model.User{
		ID:       1,
		Username: "testuser",
		Role:     "user",
	}

	// Generuj token
	token, err := GenerateToken(user)
	assert.NoError(t, err)

	// Odśwież token
	newToken, err := RefreshToken(token)
	assert.NoError(t, err)
	assert.NotEmpty(t, newToken)
	assert.NotEqual(t, token, newToken)

	// Sprawdź czy nowy token jest poprawny
	claims, err := ValidateToken(newToken)
	assert.NoError(t, err)
	assert.Equal(t, user.ID, claims.UserID)
	assert.Equal(t, user.Username, claims.Username)
}
