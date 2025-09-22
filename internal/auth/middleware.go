package auth

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
)

// JWTMiddleware validates JWT tokens
func (j *JWTService) JWTMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Get token from Authorization header
			authHeader := c.Request().Header.Get("Authorization")
			if authHeader == "" {
				return c.JSON(http.StatusUnauthorized, map[string]interface{}{
					"success": false,
					"error":   "Authorization header required",
				})
			}

			// Check if it starts with "Bearer "
			if !strings.HasPrefix(authHeader, "Bearer ") {
				return c.JSON(http.StatusUnauthorized, map[string]interface{}{
					"success": false,
					"error":   "Invalid authorization header format",
				})
			}

			// Extract token
			token := strings.TrimPrefix(authHeader, "Bearer ")

			// Validate token
			claims, err := j.ValidateToken(token)
			if err != nil {
				return c.JSON(http.StatusUnauthorized, map[string]interface{}{
					"success": false,
					"error":   "Invalid token",
				})
			}

			// Set user info in context
			c.Set("user", claims)
			c.Set("username", claims.Username)
			c.Set("role", claims.Role)

			return next(c)
		}
	}
}

// RequireAdmin middleware checks if user has admin role
func (j *JWTService) RequireAdmin() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			role, ok := c.Get("role").(string)
			if !ok || role != "admin" {
				return c.JSON(http.StatusForbidden, map[string]interface{}{
					"success": false,
					"error":   "Admin access required",
				})
			}
			return next(c)
		}
	}
}
