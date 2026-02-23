package blockchain

import (
	"errors"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// buildFC constructs a FailoverClient directly without network I/O, for unit tests.
func buildFC(eps []*endpointStatus) *FailoverClient {
	return &FailoverClient{
		endpoints:    eps,
		currentIndex: 0,
	}
}

// fakeEthClient returns a non-nil *ethclient.Client pointer (zero value, no connection).
// Only safe to use in paths that do not call methods on the returned client.
func fakeEthClient() *ethclient.Client {
	return new(ethclient.Client)
}

// healthyEP builds a healthy endpoint with no real connection.
func healthyEP(url string) *endpointStatus {
	return &endpointStatus{
		url:     url,
		client:  fakeEthClient(),
		healthy: true,
	}
}

// unhealthyEP builds an unhealthy endpoint whose cooldown has NOT yet expired.
func unhealthyEP(url string) *endpointStatus {
	return &endpointStatus{
		url:           url,
		client:        nil,
		healthy:       false,
		lastError:     errors.New("connection refused"),
		lastErrorTime: time.Now(), // cooldown not yet expired
	}
}

// expiredEP builds an unhealthy endpoint whose cooldown HAS expired.
// GetClient will try to reconnect; we keep client nil so the Dial inside
// GetClient will fail immediately and the endpoint stays unhealthy.
func expiredEP(url string) *endpointStatus {
	return &endpointStatus{
		url:           url,
		client:        nil,
		healthy:       false,
		lastError:     errors.New("connection refused"),
		lastErrorTime: time.Now().Add(-2 * unhealthyDuration),
	}
}

// --- GetEndpointsHealth ---

func TestGetEndpointsHealth_AllHealthy(t *testing.T) {
	fc := buildFC([]*endpointStatus{
		healthyEP("https://rpc1.example.com"),
		healthyEP("https://rpc2.example.com"),
	})

	health := fc.GetEndpointsHealth()

	assert.True(t, health["https://rpc1.example.com"])
	assert.True(t, health["https://rpc2.example.com"])
}

func TestGetEndpointsHealth_AllUnhealthy(t *testing.T) {
	fc := buildFC([]*endpointStatus{
		unhealthyEP("https://rpc1.example.com"),
		unhealthyEP("https://rpc2.example.com"),
	})

	health := fc.GetEndpointsHealth()

	assert.False(t, health["https://rpc1.example.com"])
	assert.False(t, health["https://rpc2.example.com"])
}

func TestGetEndpointsHealth_Mixed(t *testing.T) {
	fc := buildFC([]*endpointStatus{
		healthyEP("https://ok.example.com"),
		unhealthyEP("https://down.example.com"),
	})

	health := fc.GetEndpointsHealth()

	assert.True(t, health["https://ok.example.com"])
	assert.False(t, health["https://down.example.com"])
}

func TestGetEndpointsHealth_Empty(t *testing.T) {
	fc := buildFC(nil)
	health := fc.GetEndpointsHealth()
	assert.Empty(t, health)
}

// --- GetClient ---

func TestGetClient_SingleHealthyEndpoint(t *testing.T) {
	ep := healthyEP("https://rpc.example.com")
	fc := buildFC([]*endpointStatus{ep})

	client, url, err := fc.GetClient()

	require.NoError(t, err)
	assert.Equal(t, ep.client, client)
	assert.Equal(t, "https://rpc.example.com", url)
}

func TestGetClient_FirstUnhealthySecondHealthy(t *testing.T) {
	ep1 := unhealthyEP("https://rpc1.example.com")
	ep2 := healthyEP("https://rpc2.example.com")
	fc := buildFC([]*endpointStatus{ep1, ep2})

	client, url, err := fc.GetClient()

	require.NoError(t, err)
	assert.Equal(t, ep2.client, client)
	assert.Equal(t, "https://rpc2.example.com", url)
}

func TestGetClient_AllUnhealthy_CooldownActive_ReturnsError(t *testing.T) {
	fc := buildFC([]*endpointStatus{
		unhealthyEP("https://rpc1.example.com"),
		unhealthyEP("https://rpc2.example.com"),
	})

	_, _, err := fc.GetClient()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "no healthy RPC endpoints available")
}

func TestGetClient_AdvancesCurrentIndex(t *testing.T) {
	ep1 := unhealthyEP("https://rpc1.example.com")
	ep2 := healthyEP("https://rpc2.example.com")
	fc := buildFC([]*endpointStatus{ep1, ep2})

	_, _, err := fc.GetClient()
	require.NoError(t, err)

	// currentIndex should have advanced to ep2 (index 1)
	assert.Equal(t, 1, fc.currentIndex)
}

func TestGetClient_SingleEndpoint_Unhealthy_NoExpiry(t *testing.T) {
	// Single unhealthy endpoint with unexpired cooldown — no retry attempt.
	fc := buildFC([]*endpointStatus{
		unhealthyEP("https://rpc.example.com"),
	})

	_, _, err := fc.GetClient()

	require.Error(t, err)
}

// --- MarkUnhealthy ---

func TestMarkUnhealthy_KnownURL_ChangesStatus(t *testing.T) {
	// Use nil client to avoid calling Close() on a zero-value ethclient.Client.
	ep := &endpointStatus{url: "https://rpc.example.com", client: nil, healthy: true}
	fc := buildFC([]*endpointStatus{ep})

	fc.MarkUnhealthy("https://rpc.example.com", errors.New("timeout"))

	health := fc.GetEndpointsHealth()
	assert.False(t, health["https://rpc.example.com"])
}

func TestMarkUnhealthy_NilClient_DoesNotPanic(t *testing.T) {
	ep := &endpointStatus{
		url:     "https://rpc.example.com",
		client:  nil, // already nil — no Close() call
		healthy: true,
	}
	fc := buildFC([]*endpointStatus{ep})

	assert.NotPanics(t, func() {
		fc.MarkUnhealthy("https://rpc.example.com", errors.New("oops"))
	})
	assert.False(t, ep.healthy)
}

func TestMarkUnhealthy_SetsLastError(t *testing.T) {
	ep := &endpointStatus{url: "https://rpc.example.com", client: nil, healthy: true}
	fc := buildFC([]*endpointStatus{ep})
	sentErr := errors.New("dial failed")

	fc.MarkUnhealthy("https://rpc.example.com", sentErr)

	ep.mu.RLock()
	defer ep.mu.RUnlock()
	assert.Equal(t, sentErr, ep.lastError)
	assert.False(t, ep.healthy)
	assert.WithinDuration(t, time.Now(), ep.lastErrorTime, time.Second)
}

func TestMarkUnhealthy_UnknownURL_IsNoOp(t *testing.T) {
	ep := &endpointStatus{url: "https://rpc.example.com", client: nil, healthy: true}
	fc := buildFC([]*endpointStatus{ep})

	// Should not panic and must not change the known endpoint
	fc.MarkUnhealthy("https://unknown.example.com", errors.New("nope"))

	health := fc.GetEndpointsHealth()
	assert.True(t, health["https://rpc.example.com"], "known endpoint must remain healthy")
}

func TestMarkUnhealthy_CooldownTimestampUpdated(t *testing.T) {
	ep := &endpointStatus{
		url:           "https://rpc.example.com",
		client:        nil,
		healthy:       false,
		lastError:     errors.New("old error"),
		lastErrorTime: time.Now().Add(-time.Hour), // old timestamp
	}
	fc := buildFC([]*endpointStatus{ep})

	fc.MarkUnhealthy("https://rpc.example.com", errors.New("new error"))

	ep.mu.RLock()
	defer ep.mu.RUnlock()
	// lastErrorTime must have been refreshed
	assert.WithinDuration(t, time.Now(), ep.lastErrorTime, time.Second)
}

// --- Close ---

func TestClose_AllNilClients_DoesNotPanic(t *testing.T) {
	fc := buildFC([]*endpointStatus{
		{url: "https://rpc1.example.com", client: nil},
		{url: "https://rpc2.example.com", client: nil},
	})

	assert.NotPanics(t, func() {
		fc.Close()
	})
}

func TestClose_SetsClientToNil(t *testing.T) {
	// Use a nil client to avoid calling ethclient.Close() on a zero-value struct.
	ep := &endpointStatus{
		url:     "https://rpc.example.com",
		client:  nil,
		healthy: true,
	}
	fc := buildFC([]*endpointStatus{ep})

	fc.Close()

	ep.mu.RLock()
	defer ep.mu.RUnlock()
	assert.Nil(t, ep.client)
}

// --- NewFailoverClient (error paths only) ---

func TestNewFailoverClient_EmptyURLs_ReturnsError(t *testing.T) {
	_, err := NewFailoverClient([]string{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "at least one RPC URL")
}

func TestNewFailoverClient_AllUnreachable_ReturnsError(t *testing.T) {
	// Use addresses that will fail to connect immediately.
	_, err := NewFailoverClient([]string{"http://127.0.0.1:1", "http://127.0.0.1:2"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no healthy RPC endpoints available")
}
