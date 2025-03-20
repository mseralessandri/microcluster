# Microcluster dqlite database

Microcluster uses the distributed database [dqlite](https://github.com/canonical/dqlite) to persist and replicate state information across all cluster members.

All related database files are stored in the `/database` directory in the Microcluster state directory. A single database is used for both internal Microcluster data and application data.

# Schema

By default, Microcluster offers 3 tables:

* `core_cluster_members`: Current members of the cluster.
* `core_token_records`: Currently active join token secrets and their associated joiner names.
* `schemas`: Record of the currently active schema version of the cluster.

The table name prefix `core_` is considered reserved for internal data, and any supplied extensions to the schema should not create any tables prefixed with `core_`.

It is safe to make foreign key references to these tables. Such references will be preserved should these tables encounter internal schema changes.

For information about schema updates, refer to [upgrades.md](doc/upgrades.md).
