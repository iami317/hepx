package validation

import (
	"errors"

	v1 "github.com/iami317/hepx/pkg/config/v1"
	splugin "github.com/iami317/hepx/pkg/plugin/server"
)

var (
	SupportedTransportProtocols = []string{
		"tcp",
		"kcp",
		"quic",
		"websocket",
		"wss",
	}

	SupportedAuthMethods = []v1.AuthMethod{
		"token",
		"oidc",
	}

	SupportedAuthAdditionalScopes = []v1.AuthScope{
		"HeartBeats",
		"NewWorkConns",
	}

	SupportedLogLevels = []string{
		"trace",
		"debug",
		"info",
		"warn",
		"error",
	}

	SupportedHTTPPluginOps = []string{
		splugin.OpLogin,
		splugin.OpNewProxy,
		splugin.OpCloseProxy,
		splugin.OpPing,
		splugin.OpNewWorkConn,
		splugin.OpNewUserConn,
	}
)

type Warning error

func AppendError(err error, errs ...error) error {
	if len(errs) == 0 {
		return err
	}
	return errors.Join(append([]error{err}, errs...)...)
}
