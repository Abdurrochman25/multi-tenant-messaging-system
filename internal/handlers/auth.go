package handlers

import (
	"net/http"
	"time"

	"github.com/Abdurrochman25/multi-tenant-messaging-system/internal/config"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v4"
)

type AuthHandler struct {
	secret string
}

type LoginRequest struct {
	TenantID string `json:"tenant_id"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type TokenResponse struct {
	Token string `json:"token"`
}

func NewAuthHandler(s *config.Server, secret string) []fiber.Router {
	handler := AuthHandler{secret: secret}

	return []fiber.Router{
		s.Fiber.Post("/v1/auth/login", handler.Login),
	}
}

// Login generates a JWT token for authentication
// @Summary Login and get JWT token
// @Description Authenticate and receive a JWT token for API access
// @Tags auth
// @Accept json
// @Produce json
// @Param credentials body LoginRequest true "Login credentials"
// @Success 200 {object} TokenResponse
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Router /auth/login [post]
func (h *AuthHandler) Login(c *fiber.Ctx) error {
	var req LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Simple validation (in production, use proper authentication)
	if req.Username == "" || req.Password == "" || req.TenantID == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Username, password, and tenant_id are required",
		})
	}

	// Create JWT token with tenant information
	token := jwt.New(jwt.SigningMethodHS256)
	claims := token.Claims.(jwt.MapClaims)
	claims["username"] = req.Username
	claims["tenant_id"] = req.TenantID
	claims["exp"] = time.Now().Add(time.Hour * 72).Unix()

	t, err := token.SignedString([]byte(h.secret))
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Could not generate token",
		})
	}

	return c.JSON(TokenResponse{
		Token: t,
	})
}