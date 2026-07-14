package route

import (
	"net/http"
	"time"

	"github.com/IceWhaleTech/CasaOS/model"
	"github.com/IceWhaleTech/CasaOS/pkg/jwt"
	"github.com/IceWhaleTech/CasaOS/service"
	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt"
)

// LoginRequest struktura żądania logowania
type LoginRequest struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
}

// Login obsługuje logowanie użytkownika
func Login(c echo.Context) error {
	var req LoginRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, model.Result{
			Success: false,
			Message: "Invalid request",
			Data:    nil,
		})
	}

	// Pobierz użytkownika z bazy
	user, err := service.GetUserByUsername(req.Username)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, model.Result{
			Success: false,
			Message: "Invalid credentials",
			Data:    nil,
		})
	}

	// Sprawdź czy użytkownik jest aktywny
	if !user.Enabled {
		return c.JSON(http.StatusForbidden, model.Result{
			Success: false,
			Message: "User account is disabled",
			Data:    nil,
		})
	}

	// Porównaj hasło
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return c.JSON(http.StatusUnauthorized, model.Result{
			Success: false,
			Message: "Invalid credentials",
			Data:    nil,
		})
	}

	// Generuj token JWT
	token, err := jwt.GenerateToken(user)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, model.Result{
			Success: false,
			Message: "Failed to generate token",
			Data:    nil,
		})
	}

	// Aktualizuj ostatnie logowanie
	service.UpdateLastLogin(user.ID, time.Now())

	return c.JSON(http.StatusOK, model.Result{
		Success: true,
		Message: "Login successful",
		Data: map[string]interface{}{
			"token": token,
			"user": map[string]interface{}{
				"id":       user.ID,
				"username": user.Username,
				"email":    user.Email,
				"role":     user.Role,
			},
		},
	})
}

// GetProfile zwraca profil zalogowanego użytkownika
func GetProfile(c echo.Context) error {
	userID, ok := c.Get("user_id").(uint)
	if !ok {
		return c.JSON(http.StatusUnauthorized, model.Result{
			Success: false,
			Message: "User not authenticated",
			Data:    nil,
		})
	}

	user, err := service.GetUserByID(userID)
	if err != nil {
		return c.JSON(http.StatusNotFound, model.Result{
			Success: false,
			Message: "User not found",
			Data:    nil,
		})
	}

	return c.JSON(http.StatusOK, model.Result{
		Success: true,
		Message: "Profile retrieved",
		Data: map[string]interface{}{
			"id":       user.ID,
			"username": user.Username,
			"email":    user.Email,
			"role":     user.Role,
			"enabled":  user.Enabled,
		},
	})
}

// Logout obsługuje wylogowanie (po stronie klienta usuwa token)
func Logout(c echo.Context) error {
	// Ponieważ JWT jest stateless, serwer nie musi nic robić
	// Klient powinien usunąć token ze swojej pamięci
	return c.JSON(http.StatusOK, model.Result{
		Success: true,
		Message: "Logged out successfully",
		Data:    nil,
	})
}
