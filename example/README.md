# Example implementation of Microcluster

This example package can be used as a starting point for creating projects with Microcluster. This package contains:
- A Microcluster daemon command (`microd`) and a control command (`microctl`)
- Examples for the built-in Microcluster API
- Examples of how Microcluster can be extended with additional listeners with user-defined endpoints and schema versions

## Contents

- [How to build and run the package](#how-to-build-and-run-the-package)
  - [Prerequisites](#prerequisites)
  - [Build](#build)
  - [Run](#run)
  - [Control](#control)
- [Tutorial](#tutorial)
  - [Step 1: Build the package](#step-1-build-the-package)
  - [Step 2: Start three Microcluster daemons](#step-2-start-three-microcluster-daemons)
  - [Step 3: Start the Dqlite database](#step-3-start-the-dqlite-database)
  - [Step 4: Join the other members to the cluster](#step-4-join-the-other-members-to-the-cluster)
  - [Step 5: Interact with the cluster](#step-5-interact-with-the-cluster)
    - [Step 5.1: Remove a cluster member](#step-51-remove-a-cluster-member)
    - [Step 5.2: Perform SQL queries](#step-52-perform-sql-queries)
    - [Step 5.3: Perform an extended API interaction](#step-53-perform-an-extended-api-interaction)
  - [Step 6: Shut down the Microcluster](#step-6-shut-down-the-microcluster)

## How to build and run the package

### Prerequisites

#### Required packages

- `Go` must be installed on your system.
  - If [snaps are available on your system](https://snapcraft.io/docs/installing-snapd), you can install Go by running:

    ```
    sudo snap install go --classic
    ```

  - If you cannot use snaps, follow the [installation instructions from Go.dev](https://go.dev/doc/install).

- Run the following commands to install the remaining required packages:

  ```bash
  sudo apt-get update
  sudo apt-get install --no-install-recommends -y \
            shellcheck \
            pkg-config \
            autoconf \
            automake \
            libtool \
            make \
            libuv1-dev \
            libsqlite3-dev \
            liblz4-dev
  ```

#### Environment variables

After Go is installed, ensure that the CGO_ENABLED environment variable is persistently set to `1`, which allows Go programs to interface with C libraries:

```bash
go env -w CGO_ENABLED=1
```

Additional variables must be set in your shell to ensure that Dqlite dependencies can be loaded during the build:

| ENVIRONMENT VARIABLE | VALUE                                                       |
|----------------------|-------------------------------------------------------------|
| CGO_CFLAGS           | -I$HOME/go/deps/dqlite/include/                             |
| CGO_LDFLAGS          | -L$HOME/go/deps/dqlite/.libs/                               |
| LD_LIBRARY_PATH      | $HOME/go/deps/dqlite/.libs/                                 |
| CGO_LDFLAGS_ALLOW    | (-Wl,-wrap,pthread_create)\|(-Wl,-z,now)                    |

If you are using `bash` as your shell, you can set the above as persistent variables with the following commands:

```bash
cat << EOF >> ~/.bashrc
export CGO_CFLAGS="-I$HOME/go/deps/dqlite/include/"
export CGO_LDFLAGS="-L$HOME/go/deps/dqlite/.libs/"
export LD_LIBRARY_PATH="$HOME/go/deps/dqlite/.libs/"
export CGO_LDFLAGS_ALLOW="(-Wl,-wrap,pthread_create)|(-Wl,-z,now)"
EOF
source ~/.bashrc
```

### Build

Clone this repository to your system, then run the following command from the repository's root directory:

```bash
make -C example
```

After a successful build, test the `microctl` and `microd` commands by running:

```bash
microctl --help
microd --help
```

If either command returns a `not found` error, confirm that the `microctl` and `microd` have been generated. Typically, they are generated in the `~/go/bin/` directory. If this directory is not defined in your system path, you must add it.

If you're using `bash`, you can add it by running:

```bash
echo 'export PATH="$HOME/go/bin:$PATH"' >> ~/.bashrc
source ~/.bashrc
```

The `microd` command starts the Microcluster daemon, and `microctl` controls it.

Note: These commands are not a core component of the Microcluster library; they are included in this example package to demonstrate how you can create similar commands for your own implementation.

### Run

Use the `microd` command along with the `--state-dir <path/to/state/directory>` flag to start a Microcluster daemon with a running control socket and no database.

You must specify its [state directory](../doc/state.md), the path where the daemon's information is stored. If the state directory does not exist at the path you provide, `microd` creates the directory.

View the `microd --help` documentation for other options, or [view the tutorial](#tutorial) for example usage.

### Control

Use the `microctl` command to control the Microcluster. View `microctl --help` for options, or [view the tutorial](#tutorial) for example usage.

## Tutorial

This tutorial walks you through using the example package to start up a Microcluster and interact with it.

### Step 1: Build the package

Ensure that your system meets the prerequisites in the [how-to guide](#how-to-build-and-run-the-package) above, then [build](#build) the package.

### Step 2: Start three Microcluster daemons

The commands below start three Microcluster daemons in the background, creating state directories for each and waiting for the daemon to be ready to process requests. Each daemon's PID is stored in a shell variable (`proc1`, `proc2`, and `proc3`) for later use.

Run:

```bash
microd --state-dir /tmp/mc1 & proc1=$!
microd --state-dir /tmp/mc2 & proc2=$!
microd --state-dir /tmp/mc3 & proc3=$!
```

You should see three "Microcluster database is uninitialized" warnings. Don't worry; this is expected.

### Step 3: Start the Dqlite database

Run the following command to bootstrap the first Microcluster member, which starts a new cluster:

```bash
microctl --state-dir /tmp/mc1 init mc1 127.0.0.1:9001 --bootstrap
```

You might see a warning that "The 'missing_extension' is not registered". Disregard this warning.

To confirm creation of the cluster, run the following command:

```bash
microctl --state-dir /tmp/mc1 cluster list
```

You should see a table that displays a single cluster member with the name of `mc1` and an address of `127.0.0.1:9001`, along with its role, fingerprint, and status. Ensure that the status is `ONLINE` before you proceed.

### Step 4: Join the other members to the cluster

To generate and use join tokens for the second and third Microcluster members, using the names `mc2` and `mc3`, run:

```bash
token=$(microctl --state-dir /tmp/mc1 tokens add mc2)
microctl --state-dir /tmp/mc2 init mc2 127.0.0.1:9002 --token "$token"
token=$(microctl --state-dir /tmp/mc1 tokens add mc3)
microctl --state-dir /tmp/mc3 init mc3 127.0.0.1:9003 --token "$token"
```

Note: Each token can only be used once, because they are deleted from the cluster after use.

To confirm that the second and third Microcluster members have joined the cluster, view the cluster list again.

### Step 5: Interact with the cluster

#### Step 5.1: Remove a cluster member

You have created three Microcluster daemons, used one to bootstrap a new cluster, then joined the other two daemons to that cluster. Next, remove the `mc3` cluster member:

```bash
microctl --state-dir /tmp/mc1 cluster remove mc3
```

Note: When using `cluster remove`, for the `--state-dir` argument, you can use the state directory for any online cluster member. This includes the state directory of the cluster member being removed.

#### Step 5.2: Perform SQL queries

The `microctl` command includes an option to execute an SQL query against the cluster's Dqlite database.

Run the following command to view all available tables:

```bash
microctl --state-dir /tmp/mc1 sql "SELECT name FROM sqlite_master WHERE type='table';"
```

Expected output:

```
+----------------------+
|         name         |
+----------------------+
| sqlite_sequence      |
| schemas              |
| core_cluster_members |
| core_token_records   |
| extended_table       |
| some_other_table     |
+----------------------+
```

Try querying data from the `core_cluster_members` table:

```bash
microctl --state-dir /tmp/mc1 sql "SELECT name,address,heartbeat FROM core_cluster_members"
```

Expected output:

```
+------+----------------+--------------------------------+
| name |    address     |           heartbeat            |
+------+----------------+--------------------------------+
| mc1  | 127.0.0.1:9001 | 2025-03-20T17:52:03.101704405Z |
| mc2  | 127.0.0.1:9002 | 2025-03-20T17:52:03.125658611Z |
+------+----------------+--------------------------------+
```

Finally, perform an SQL query on an extended schema table called `extended_table`:

```bash
microctl --state-dir /tmp/mc1 sql "insert into extended_table (key, value) values ('some_key', 'some_value')"
```

Expected output:

```
Rows affected: 1
```

View the updated table:

```bash
microctl --state-dir /tmp/mc1 sql "select * from extended_table"
```

Expected output:

```
+----+----------+------------+
| id |   key    |   value    |
+----+----------+------------+
| 1  | some_key | some_value |
+----+----------+------------+
```

#### Step 5.3: Perform an extended API interaction

Run:

```bash
microctl --state-dir /tmp/mc2 extended 127.0.0.1:9001
```

Expected output:

```
cluster member at address "127.0.0.1:9002" received message "Testing 1 2 3..." from cluster member at address "127.0.0.1:9001"
```

This demonstrates using the extended API to communicate between `mc1` (127.0.0.1:9001) and `mc2` (127.0.0.1:9002).

### Step 6: Shut down the Microcluster

To shut down the Microcluster, stop the daemons using the PID variables defined in Step 2:

```bash
kill $proc1 $proc2 $proc3
```

If you are done using the example package, or you want to start the tutorial over from the beginning, also remove the `/tmp/mc1`, `/tmp/mc2`, and `/tmp/mc3` directories:

```bash
rm -rf /tmp/mc*
```

## Next steps

- Explore [more of the Microcluster developer documentation](../doc/).