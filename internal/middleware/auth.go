package middleware

import (
	"github.com/gofiber/fiber/v2"
	jwtware "github.com/gofiber/jwt/v3"
	"github.com/golang-jwt/jwt/v4"
)

func JWTProtected(secret string) fiber.Handler {
	return jwtware.New(jwtware.Config{
		SigningKey:   []byte(secret),
		ErrorHandler: jwtError,
	})
}

func jwtError(c *fiber.Ctx, err error) error {
	if err.Error() == "Missing or malformed JWT" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Missing or malformed JWT",
		})
	}
	return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
		"error": "Invalid or expired JWT",
	})
}

func ExtractTenantFromJWT(c *fiber.Ctx) string {
	user := c.Locals("user").(*jwt.Token)
	claims := user.Claims.(jwt.MapClaims)
	
	if tenantID, ok := claims["tenant_id"].(string); ok {
		return tenantID
	}
	
	return ""
}

func TenantSpecific() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Extract tenant from JWT
		tenantIDFromJWT := ExtractTenantFromJWT(c)
		
		// Extract tenant from URL
		tenantIDFromURL := c.Params("tenant_id")
		
		// If both exist, they must match
		if tenantIDFromJWT != "" && tenantIDFromURL != "" {
			if tenantIDFromJWT != tenantIDFromURL {
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
					"error": "Access denied: tenant mismatch",
				})
			}
		}
		
		return c.Next()
	}
}