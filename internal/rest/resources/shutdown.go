package resources

import (
	"fmt"
	"net/http"

	internalState "github.com/canonical/microcluster/v3/internal/state"
	"github.com/canonical/microcluster/v3/rest"
	"github.com/canonical/microcluster/v3/rest/access"
	"github.com/canonical/microcluster/v3/rest/response"
	"github.com/canonical/microcluster/v3/rest/types"
	"github.com/canonical/microcluster/v3/state"
)

var shutdownCmd = rest.Endpoint{
	AllowedBeforeInit: true,
	Path:              "shutdown",

	Post: rest.EndpointAction{Handler: shutdownPost, AccessHandler: access.AllowAuthenticated},
}

func shutdownPost(state state.State, r *http.Request) response.Response {
	intState, err := internalState.ToInternal(state)
	if err != nil {
		return response.SmartError(err)
	}

	if intState.Context.Err() != nil {
		return response.SmartError(fmt.Errorf("Shutdown already in progress"))
	}

	return response.ManualResponse(func(w http.ResponseWriter) error {
		// If the database is waiting for an upgrade, we may never become ready, so go ahead and shut down the database anyway.
		if state.Database().Status() != types.DatabaseWaiting {
			<-intState.ReadyCh // Wait for daemon to start.
		}

		// Run shutdown sequence synchronously.
		exit, stopErr := intState.Stop()
		err := response.SmartError(stopErr).Render(w, r)
		if err != nil {
			return err
		}

		// Send the response before the daemon process ends.
		f, ok := w.(http.Flusher)
		if ok {
			return fmt.Errorf("ResponseWriter is not type http.Flusher")
		}

		f.Flush()

		// Send result of d.Stop() to cmdDaemon so that process stops with correct exit code from Stop().
		go func() {
			<-r.Context().Done() // Wait until request is finished.
			exit()
		}()

		return nil
	})
}
