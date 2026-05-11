package rabbitmq

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildURI_defaultVHost(t *testing.T) {
	s := BuildURI("u", "p", "rabbit", 5672, "/")

	u, err := url.Parse(s)
	require.NoError(t, err)
	require.Equal(t, "amqp", u.Scheme)
	require.Equal(t, "u:p", u.User.String())
	require.Equal(t, "rabbit:5672", u.Host)
	require.Equal(t, "amqp://u:p@rabbit:5672/%2F", s)
}

func TestBuildURI_nestedVHostEscapesSegment(t *testing.T) {
	s := BuildURI("u", "p", "rabbit", 5672, "tenant/a")
	require.Contains(t, s, "tenant%2Fa")

	_, err := url.Parse(s)
	require.NoError(t, err)
}
