package validation

import (
	"fmt"
	"slices"

	v1 "github.com/iami317/hepx/pkg/config/v1"
)

func validateWebServerConfig(c *v1.WebServerConfig) error {
	if c.TLS != nil {
		if c.TLS.CertFile == "" {
			return fmt.Errorf("tls.certFile must be specified when tls is enabled")
		}
		if c.TLS.KeyFile == "" {
			return fmt.Errorf("tls.keyFile must be specified when tls is enabled")
		}
	}

	return ValidatePort(c.Port, "webServer.port")
}

// ValidatePort checks that the network port is in range
func ValidatePort(port int, fieldPath string) error {
	if 0 <= port && port <= 65535 {
		return nil
	}
	return fmt.Errorf("%s: port number %d must be in the range 0..65535", fieldPath, port)
}

func validateLogConfig(c *v1.LogConfig) error {
	if !slices.Contains(SupportedLogLevels, c.Level) {
		return fmt.Errorf("invalid log level, optional values are %v", SupportedLogLevels)
	}
	return nil
}
