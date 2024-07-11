package v1

import (
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"
)

func TestClientConfigComplete(t *testing.T) {
	require := require.New(t)
	c := &ClientConfig{}
	c.Complete()

	require.EqualValues("token", c.Auth.Method)
	require.Equal(true, lo.FromPtr(c.Transport.TCPMux))
	require.Equal(true, lo.FromPtr(c.LoginFailExit))
	require.Equal(true, lo.FromPtr(c.Transport.TLS.Enable))
	require.Equal(true, lo.FromPtr(c.Transport.TLS.DisableCustomTLSFirstByte))
	require.NotEmpty(c.NatHoleSTUNServer)
}
