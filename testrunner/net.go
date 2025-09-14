package testrunner

import (
	"net"
	"testing"

	"github.com/stretchr/testify/require"
)

func RandLocalTCPAddr(t *testing.T) *net.TCPAddr {
	t.Helper()
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer func() {
		require.NoError(t, lis.Close())
	}()
	return lis.Addr().(*net.TCPAddr)
}
