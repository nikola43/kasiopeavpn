package middleware

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
	"github.com/nikola43/kasiopeavpn/utils"
)

func WebSocketUpgradeMiddleware(context *fiber.Ctx) error {
	// IsWebSocketUpgrade returns true if the client
	// requested upgrade to the WebSocket protocol.
	if websocket.IsWebSocketUpgrade(context) {
		context.Locals("allowed", true)
		return context.Next()
	}

	return fiber.ErrUpgradeRequired
}

func XApiKeyMiddleware(context *fiber.Ctx) error {
	requestApiKey := context.Get("XAPIKEY")
	serverApiKey := utils.GetEnvVariable("XAPIKEY")
	// fmt.Println("requestApiKey")
	// fmt.Println(requestApiKey)
	// fmt.Println("serverApiKey")
	// fmt.Println(serverApiKey)
	// context.h

	if requestApiKey != serverApiKey {
		return context.Status(fiber.StatusUnauthorized).JSON(&fiber.Map{
			"error": "unauthorized",
		})
	}

	return context.Next()
}
