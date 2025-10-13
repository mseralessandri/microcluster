package resources

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/canonical/microcluster/v3/internal/state"
	"github.com/canonical/microcluster/v3/rest"
	"github.com/canonical/microcluster/v3/rest/response"
)

var databaseCmd = rest.Endpoint{
	AllowedBeforeInit: true,
	Path:              "database",

	Post:  rest.EndpointAction{Handler: databasePost},
	Patch: rest.EndpointAction{Handler: databasePatch},
}

func databasePost(state state.State, r *http.Request) response.Response {
	// Compare the dqlite version of the connecting client with our own.
	versionHeader := r.Header.Get("X-Dqlite-Version")
	if versionHeader == "" {
		// No version header means an old pre dqlite 1.0 client.
		versionHeader = "0"
	}

	_, err := strconv.Atoi(versionHeader)
	if err != nil {
		return response.BadRequest(fmt.Errorf("Invalid dqlite version: %w", err))
	}

	// Handle leader address requests.
	if r.Header.Get("Upgrade") != "dqlite" {
		return response.BadRequest(fmt.Errorf("Missing or invalid upgrade header"))
	}

	return response.EmptySyncResponse
}

func databasePatch(s state.State, r *http.Request) response.Response {
	// Compare the dqlite version of the connecting client with our own.
	versionHeader := r.Header.Get("X-Dqlite-Version")
	if versionHeader == "" {
		// No version header means an old pre dqlite 1.0 client.
		versionHeader = "0"
	}

	_, err := strconv.Atoi(versionHeader)
	if err != nil {
		return response.BadRequest(fmt.Errorf("Invalid dqlite version: %w", err))
	}

	intState, err := state.ToInternal(s)
	if err != nil {
		return response.SmartError(err)
	}

	// Notify this node that a schema upgrade has occurred, in case we are waiting on one.
	intState.InternalDatabase.NotifyUpgraded()

	return response.EmptySyncResponse
}
