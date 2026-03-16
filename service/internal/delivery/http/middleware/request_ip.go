package middleware

import (
	"net"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/m11ano/neurochar-experiments-3/service/internal/infra/loghandler"
	"github.com/samber/lo"
)

// RequestIP - middleware for request ip
func RequestIP(serverIPs []string) func(*fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		requestIP := parseRealIP(c, serverIPs)

		c.Locals("requestIP", requestIP)

		ctxData, ctxKey := loghandler.SetData(c.Context(), "request.ip", requestIP)
		c.Locals(ctxKey, ctxData)

		return c.Next()
	}
}

func parseRealIP(c *fiber.Ctx, serverIPs []string) string {
	ips := strings.Split(c.IP(), ",")
	for i := range ips {
		ips[i] = strings.TrimSpace(ips[i])
	}

	for i := len(ips) - 1; i >= 0; i-- {
		ip := ips[i]
		parsed := net.ParseIP(ip)
		if parsed == nil {
			continue
		}
		if !lo.Contains(serverIPs, ip) && !IsPrivateIP(parsed) {
			return ip
		}
	}

	if len(ips) > 0 {
		return ips[len(ips)-1]
	}

	return c.IP()
}

func GetRealIP(c *fiber.Ctx) string {
	ip := c.Locals("requestIP")
	ipStr, ok := ip.(string)
	if ok {
		return ipStr
	}

	return c.IP()
}

var privateBlocks = []string{
	"10.0.0.0/8",
	"172.16.0.0/12",
	"192.168.0.0/16",
	"127.0.0.0/8",
	"::1/128",
}

func IsPrivateIP(ip net.IP) bool {
	for _, block := range privateBlocks {
		_, subnet, err := net.ParseCIDR(block)
		if err != nil {
			continue
		}
		if subnet.Contains(ip) {
			return true
		}
	}

	return false
}
