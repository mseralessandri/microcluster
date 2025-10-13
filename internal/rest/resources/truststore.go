package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/gorilla/mux"

	"github.com/canonical/microcluster/v3/client"
	internalClient "github.com/canonical/microcluster/v3/internal/rest/client"
	"github.com/canonical/microcluster/v3/internal/trust"
	"github.com/canonical/microcluster/v3/rest"
	"github.com/canonical/microcluster/v3/rest/access"
	"github.com/canonical/microcluster/v3/rest/response"
	"github.com/canonical/microcluster/v3/rest/types"
	"github.com/canonical/microcluster/v3/state"
)

var trustCmd = rest.Endpoint{
	Path:              "truststore",
	AllowedBeforeInit: true,

	Post: rest.EndpointAction{Handler: trustPost, AccessHandler: access.AllowAuthenticated},
}

var trustEntryCmd = rest.Endpoint{
	Path:              "truststore/{name}",
	AllowedBeforeInit: true,

	Delete: rest.EndpointAction{Handler: trustDelete, AccessHandler: access.AllowAuthenticated},
}

func trustPost(s state.State, r *http.Request) response.Response {
	req := types.ClusterMemberLocal{}

	// Parse the request.
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		return response.BadRequest(err)
	}

	newRemote := trust.Remote{
		Location:    trust.Location{Name: req.Name, Address: req.Address},
		Certificate: req.Certificate,
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	if !client.IsNotification(r) {
		cluster, err := s.Cluster(true)
		if err != nil {
			return response.SmartError(err)
		}

		err = cluster.Query(ctx, true, func(ctx context.Context, c *client.Client) error {
			// No need to send a request to ourselves, or to the node we are adding.
			if s.Address().URL.Host == c.URL().URL.Host || req.Address.String() == c.URL().URL.Host {
				return nil
			}

			return internalClient.AddTrustStoreEntry(ctx, &c.Client, req)
		})
		if err != nil {
			return response.SmartError(err)
		}
	}

	// At this point, the node has joined dqlite so we can add a local record for it if we haven't already from a heartbeat (or if we are the leader).
	remotes := s.Remotes()
	_, ok := remotes.RemotesByName()[newRemote.Name]
	if !ok {
		err = remotes.Add(s.FileSystem().TrustDir, newRemote)
		if err != nil {
			return response.SmartError(fmt.Errorf("Failed adding local record of newly joined node %q: %w", req.Name, err))
		}
	}

	return response.EmptySyncResponse
}

func trustDelete(s state.State, r *http.Request) response.Response {
	name, err := url.PathUnescape(mux.Vars(r)["name"])
	if err != nil {
		return response.SmartError(err)
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	remotesMap := s.Remotes().RemotesByName()
	nodeToRemove, ok := remotesMap[name]
	if !ok {
		return response.SmartError(fmt.Errorf("No truststore entry found for node with name %q", name))
	}

	if !client.IsNotification(r) {
		cluster, err := s.Cluster(true)
		if err != nil {
			return response.SmartError(err)
		}

		err = cluster.Query(ctx, true, func(ctx context.Context, c *client.Client) error {
			// No need to send a request to ourselves, or to the node we are adding.
			if s.Address().URL.Host == c.URL().URL.Host || nodeToRemove.URL().URL.Host == c.URL().URL.Host {
				return nil
			}

			return internalClient.DeleteTrustStoreEntry(ctx, &c.Client, name)
		})
		if err != nil {
			return response.SmartError(err)
		}
	}

	remotes := s.Remotes()
	remotesMap = remotes.RemotesByName()
	delete(remotesMap, name)

	newRemotes := make([]types.ClusterMember, 0, len(remotesMap))
	for _, remote := range remotesMap {
		newRemote := types.ClusterMember{
			ClusterMemberLocal: types.ClusterMemberLocal{
				Name:        remote.Name,
				Address:     remote.Address,
				Certificate: remote.Certificate,
			},
		}

		newRemotes = append(newRemotes, newRemote)
	}

	err = remotes.Replace(s.FileSystem().TrustDir, newRemotes...)
	if err != nil {
		return response.SmartError(fmt.Errorf("Failed to remove truststore entry for node with name %q: %w", name, err))
	}

	return response.EmptySyncResponse
}
