package client

import (
	"context"
	"net/http"

	"github.com/canonical/lxd/shared/api"
	"github.com/gorilla/websocket"

	"github.com/canonical/microcluster/v3/internal/rest/client"
	"github.com/canonical/microcluster/v3/rest/types"
)

// Client is a rest client for the microcluster daemon.
type Client struct {
	client.Client
}

// IsNotification determines if this request is to be considered a cluster-wide notification.
func IsNotification(r *http.Request) bool {
	return r.Header.Get("User-Agent") == client.UserAgentNotifier
}

// Query is a helper for initiating a request on any endpoints defined external to microcluster. This function should be used for all client
// methods defined externally from microcluster.
func (c *Client) Query(ctx context.Context, method string, prefix types.EndpointPrefix, path *api.URL, in any, out any) error {
	queryCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	return c.QueryStruct(queryCtx, method, prefix, path, in, &out)
}

// QueryRaw is a helper for initiating a request on any endpoints defined external to microcluster.
// Unlike Query it returns the raw HTTP response.
func (c *Client) QueryRaw(ctx context.Context, method string, prefix types.EndpointPrefix, path *api.URL, in any) (*http.Response, error) {
	queryCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	return c.QueryStructRaw(queryCtx, method, prefix, path, in)
}

// Websocket is a helper for upgrading a request to websocket on any endpoints defined external to microcluster.
// This function should be used for all client methods defined externally from microcluster.
func (c *Client) Websocket(ctx context.Context, prefix types.EndpointPrefix, path *api.URL) (*websocket.Conn, error) {
	websocketCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	return c.RawWebsocket(websocketCtx, prefix, path)
}

// UseTarget returns a new client with the query "?target=name" set.
func (c *Client) UseTarget(name string) *Client {
	newClient := c.Client.UseTarget(name)

	return &Client{Client: *newClient}
}
