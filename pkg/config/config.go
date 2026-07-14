package config

import (
	"os"
	"path/filepath"

	"github.com/IceWhaleTech/CasaOS/model"
	"github.com/IceWhaleTech/CasaOS/pkg/jwt"
	"github.com/sirupsen/logrus"
)

// InitConfig inicjalizuje wszystkie konfiguracje
func InitConfig() error {
	// Inicjalizuj JWT secret
	if err := jwt.InitJWTSecret(); err != nil {
		logrus.Errorf("Failed to init JWT secret: %v", err)
		return err
	}

	// Tutaj inne inicjalizacje konfiguracji...
	
	return nil
}

// GetJWTSecretPath zwraca ścieżkę do pliku z sekretem
func GetJWTSecretPath() string {
	// Możesz użyć zmiennej środowiskowej lub domyślnej ścieżki
	if path := os.Getenv("JWT_SECRET_PATH"); path != "" {
		return path
	}
	
	// Dla Dockera: /var/lib/casaos/jwt_secret.key
	// Dla Windows: C:\ProgramData\CasaOS\jwt_secret.key
	// Dla Linux: /var/lib/casaos/jwt_secret.key
	
	baseDir := "/var/lib/casaos"
	if os.Getenv("CASAOS_BASE_PATH") != "" {
		baseDir = os.Getenv("CASAOS_BASE_PATH")
	}
	
	return filepath.Join(baseDir, "jwt_secret.key")
}
