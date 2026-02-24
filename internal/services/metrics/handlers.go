// Package metrics contains logic to get the sensor metrics.
package metrics

import (
	"time"

	"github.com/gofiber/contrib/v3/websocket"
	"github.com/gofiber/fiber/v3"
)

func (m *Service) GetMetrics(c fiber.Ctx) error {
	c.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
	return c.JSON(m.buildSnapshot())
}

func (m *Service) NewMetricsWS() fiber.Handler {
	return websocket.New(func(conn *websocket.Conn) {
		ticker := time.NewTicker(m.sampleInterval)
		defer ticker.Stop()
		defer conn.Close()

		// Initial snapshot
		if err := conn.WriteJSON(m.buildSnapshot()); err != nil {
			return
		}

		// Periodic updates
		for range ticker.C {
			if err := conn.WriteJSON(m.buildSnapshot()); err != nil {
				return
			}
		}
	})
}
