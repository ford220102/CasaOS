package middleware

import (
	"net/http"
	"strings"

	"github.com/IceWhaleTech/CasaOS/model"
	"github.com/IceWhaleTech/CasaOS/pkg/jwt"
	"github.com/IceWhaleTech/CasaOS/utils"
	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
)

// AuthMiddleware weryfikuje token JWT
func AuthMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		// Pobierz token z nagłówka Authorization
		authHeader := c.Request().Header.Get("Authorization")
		if authHeader == "" {
			return c.JSON(http.StatusUnauthorized, model.Result{
				Success: false,
				Message: "Missing authorization header",
				Data:    nil,
			})
		}

		// Token powinien być w formacie "Bearer <token>"
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			return c.JSON(http.StatusUnauthorized, model.Result{
				Success: false,
				Message: "Invalid authorization format",
				Data:    nil,
			})
		}

		tokenString := parts[1]

		// Waliduj token
		claims, err := jwt.ValidateToken(tokenString)
		if err != nil {
			logrus.Warnf("Invalid JWT token: %v", err)
			return c.JSON(http.StatusUnauthorized, model.Result{
				Success: false,
				Message: "Invalid or expired token",
				Data:    nil,
			})
		}

		// Zapisz dane użytkownika w kontekście
		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("role", claims.Role)

		return next(c)
	}
}

// OptionalAuthMiddleware sprawdza token, ale nie wymaga go (opcjonalna autoryzacja)
func OptionalAuthMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		authHeader := c.Request().Header.Get("Authorization")
		if authHeader != "" {
			parts := strings.Split(authHeader, " ")
			if len(parts) == 2 && strings.ToLower(parts[0]) == "bearer" {
				tokenString := parts[1]
				claims, err := jwt.ValidateToken(tokenString)
				if err == nil {
					c.Set("user_id", claims.UserID)
					c.Set("username", claims.Username)
					c.Set("role", claims.Role)
				}
			}
		}
		return next(c)
	}
}

// AdminMiddleware sprawdza czy użytkownik ma rolę admina
func AdminMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		role := c.Get("role")
		if role == nil || role.(string) != "admin" {
			return c.JSON(http.StatusForbidden, model.Result{
				Success: false,
				Message: "Admin privileges required",
				Data:    nil,
			})
		}
		return next(c)
	}
}
