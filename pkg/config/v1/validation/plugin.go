package validation

import (
	"errors"

	v1 "github.com/iami317/hepx/pkg/config/v1"
)

func ValidateClientPluginOptions(c v1.ClientPluginOptions) error {
	switch v := c.(type) {
	case *v1.HTTP2HTTPSPluginOptions:
		return validateHTTP2HTTPSPluginOptions(v)
	case *v1.HTTPS2HTTPPluginOptions:
		return validateHTTPS2HTTPPluginOptions(v)
	case *v1.HTTPS2HTTPSPluginOptions:
		return validateHTTPS2HTTPSPluginOptions(v)
	case *v1.StaticFilePluginOptions:
		return validateStaticFilePluginOptions(v)
	case *v1.UnixDomainSocketPluginOptions:
		return validateUnixDomainSocketPluginOptions(v)
	}
	return nil
}

func validateHTTP2HTTPSPluginOptions(c *v1.HTTP2HTTPSPluginOptions) error {
	if c.LocalAddr == "" {
		return errors.New("localAddr is required")
	}
	return nil
}

func validateHTTPS2HTTPPluginOptions(c *v1.HTTPS2HTTPPluginOptions) error {
	if c.LocalAddr == "" {
		return errors.New("localAddr is required")
	}
	return nil
}

func validateHTTPS2HTTPSPluginOptions(c *v1.HTTPS2HTTPSPluginOptions) error {
	if c.LocalAddr == "" {
		return errors.New("localAddr is required")
	}
	return nil
}

func validateStaticFilePluginOptions(c *v1.StaticFilePluginOptions) error {
	if c.LocalPath == "" {
		return errors.New("localPath is required")
	}
	return nil
}

func validateUnixDomainSocketPluginOptions(c *v1.UnixDomainSocketPluginOptions) error {
	if c.UnixPath == "" {
		return errors.New("unixPath is required")
	}
	return nil
}
