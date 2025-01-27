# Microcluster
## Introduction

Microcluster is a Go library that provides an easy-to-use framework for creating and managing highly available [Dqlite](https://github.com/canonical/dqlite) clusters (using [go-dqlite](https://github.com/canonical/go-dqlite)). It offers an extensible API and database, which can be used to directly connect to your underlying services.

Build your service with Microcluster-managed HTTPS endpoints, API extensions, schema updates, lifecycle actions, and additional HTTPS listeners.

## Import Microcluster

Get the latest LTS release of Microcluster:

```
$ go get github.com/canonical/microcluster/v2@latest
```

## Configure and start the Microcluster service
For a step-by-step practical example of creating a Microcluster project, check out the [example package](example). Examine the code or build and test it with the included Makefile.

All of Microcluster's state is stored in the `StateDir` directory. Define a new Microcluster app:

```go
m, err := microcluster.App(microcluster.Args{StateDir: "/path/to/state"})
if err != nil {
    // ...
}
```

Define application services for Microcluster in `DaemonArgs`:
```go

dargs := microcluster.DaemonArgs{
    Version: "0.0.1",

    ExtensionsSchema: []schema.Update{
        // Sequential list of modifications to the database.
    },

    APIExtensions: []string{
        // Sequential list of API capabilities.
    },

    Servers: map[string]rest.Server{
        "my-server": {
            CoreAPI: true,
            ServeUnix: true,
            Resources: []rest.Resources{
                {
                    PathPrefix: "1.0",
                    Endpoints: []rest.Endpoint{
                        // Additional API endpoints to offer on "/1.0".
                    },
                },
            }
        }
    }
}
```

Start the daemon with your configuration:
```go
err := m.Start(ctx, dargs)
if err != nil {
    // ...
}
```

### Define lifecycle actions (hooks)

The complete set of Microcluster hooks and their behaviors are defined [in this Go file](https://github.com/canonical/microcluster/blob/v3/internal/state/hooks.go).

```go
dargs.Hooks = &state.Hooks{
    OnStart: func(ctx context.Context, s state.State) error {
        // Execute code on each startup.
    }
}
```

### Query the Dqlite database directly
```go

_, batch, err := m.Sql(ctx, "SELECT name, address FROM core_cluster_members WHERE role='voter'")
if err != nil {
    // ...
}

for i, result := range batch.Results {
    fmt.Printf("Query %d:\n", i)

    fmt.Println("Type:", result.Type)
    fmt.Println("Columns:", result.Columns)
    fmt.Println("RowsAffected:", result.RowsAffected)
    fmt.Println("Rows:", result.Rows)
}

```

### Create your own API endpoints
```go
endpoint := rest.Endpoint{
    Path: "mypath" // API is served over /mypath

    Post: rest.EndpointAction{Handler: myHandler} // POST action for the endpoint.

func myHandler(s state.State, r *http.Request) response.Response {
    msg := fmt.Sprintf("This is a response from %q at %q", s.Name(), s.Address())

    return response.SyncResponse(true, msg)
}

// Include your endpoints in DaemonArgs like so to serve over /1.0 over the default listener.
dargs := microcluster.DaemonArgs{
    Servers: map[string]rest.Server{
        "my-server": {
            CoreAPI: true,
            ServeUnix: true,
            Resources: []rest.Resources{
                {
                    PathPrefix: "1.0",
                    Endpoints: []rest.Endpoint{endpoint},
                },
            }
        }
    }
}

```

### Create your own schema extensions
```go
var schemaUpdate1 schema.Update = func(ctx context.Context, tx *sql.Tx) error {
    _, err := tx.ExecContext(ctx, "CREATE TABLE services (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT);")

    return err
}

// Include your schema extensions in DaemonArgs.
dargs := microcluster.DaemonArgs{
    ExtensionsSchema: []schema.Update{
        schemaUpdate1,
    }
}
```
