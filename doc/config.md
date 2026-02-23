# Configuration Reference

Lumenta is configured entirely through a single YAML configuration file.

## Configuration File Location

At startup, Lumenta resolves the configuration file location using the following logic:

1. If the environment variable `LUMENTA_CONFIG` is defined, its value is used as the absolute or relative path to the configuration file.
2. If `LUMENTA_CONFIG` is not defined, Lumenta attempts to load `config.yaml` from the same directory where the application binary is located.

## Configuration Structure Overview

The root configuration is divided into the following logical groups, each responsible for a clearly separated part of the system:

- [Server](#server)  
  Defines HTTP server settings such as bind address, timeouts, and proxy handling.

- [Database](#database)  
  Configures the database connection and connection pooling behavior.

- [Filesystem](#filesystem)  
  Source image directories and derivative storage location.

- [Authentication](#authentication)  
  System access control and authentication behavior.

- [Derivatives](#derivatives)  
  Thumbnail generation and available thumbnail types.

- [Sync](#sync)  
  Controls filesystem-to-database synchronization rules.

- [Site](#site)  
  Personalization and site-level metadata.

- [Presentation](#presentation)  
  UI layout and visual behavior configuration.

---
## Server

The `server` configuration group controls how Lumenta exposes its HTTP endpoint.

It directly configures the internal HTTP server.

### Example

```
server:
  addr: ":8080"

  timeouts:
    read: 15s
    write: 15s
    readHeader: 5s
    idle: 60s
```

### addr

Specifies the listening address of the HTTP server.

This value is passed directly to the underlying HTTP server and follows the standard `host:port` format.

#### Format

```
host:port
```

Both parts may influence how and where the server is reachable.

#### Common Patterns

```
addr: ":8080"
```
- Listens on port 8080
- Binds to all available network interfaces (IPv4 and IPv6 depending on OS)
- Reachable externally unless restricted by firewall

```
addr: "127.0.0.1:8080"
```
- Listens only on localhost
- Not reachable from external machines

```
addr: "0.0.0.0:8080"
```
- Explicitly binds to all IPv4 interfaces

```
addr: "[::]:8080"
```
- Explicitly binds to all IPv6 interfaces

#### Important Notes

- A port must always be specified.
- If only `:PORT` is provided, the server binds to all interfaces.
- Firewall and reverse proxy configuration may still restrict access.
- This setting only controls where the HTTP server listens, not TLS termination.

### timeouts

Defines HTTP connection timeouts applied at the server level.

All values use duration format such as:

- `5s`
- `30s`
- `1m`
- `500ms`

These settings protect the server from stalled or abusive connections and ensure predictable behavior under load.

#### read

Maximum time allowed to read the entire HTTP request, including the body.

If exceeded, the connection is closed.

#### write

Maximum time allowed to write the full HTTP response.

If exceeded, the response is aborted.

#### readHeader

Maximum time allowed to read the HTTP request headers.

Helps mitigate slow header attacks (e.g., slowloris).

#### idle

Maximum time an idle keep-alive connection is kept open.

If no new request is received within this period, the connection is closed.

### Recommended Production Baseline

```
timeouts:
  read: 15s
  write: 15s
  readHeader: 5s
  idle: 60s
```

Timeout configuration is strongly recommended for production deployments.

---
## Database

The `database` configuration group defines how Lumenta connects to its relational database.

Currently supported database engines:
- MySQL
- MariaDB

Lumenta stores metadata, relationships, and synchronization state in the database.  
Image files themselves remain on the filesystem.

Lumenta builds the connection string internally using the provided parameters.

### Example

```
database:
  host: "localhost"
  port: 3306
  name: "lumenta"
  user: "lumenta_user"
  password: "change-me"
```

### host

Specifies the hostname or IP address of the database server.

Examples:

```
host: "localhost"
host: "127.0.0.1"
host: "192.168.1.10"
host: "db.internal.network"
```

Behavior:

- If set to `localhost`, the connection targets the local machine.
- If set to a remote IP or hostname, Lumenta connects over the network.
- DNS resolution must be available if a hostname is used.

### port

Specifies the TCP port of the database server.

Example:

```
port: 3306
```

Typical default for:

- MariaDB: `3306`
- MySQL: `3306`

The port must match the database server configuration.

### name

Specifies the database schema (database name) Lumenta uses.

Example:

```
name: "lumenta"
```

The database must already exist unless created externally during deployment.

Lumenta stores (example):

- images metadata
- tags
- albums
- sync state
- relationships
- users


### user

Specifies the database user used for authentication.

Example:

```
user: "lumenta_user"
```

The user must have sufficient privileges to:

- read
- insert
- update
- delete
- create indexes (if migrations are used)

### password

Specifies the password for the configured database user.

Example:

```
password: "strong-password"
```

For production deployments:

- Avoid committing credentials to version control.
- Prefer environment variable substitution or secret management systems.
- Restrict database user permissions to the minimum required.

### Connection Behavior

Lumenta establishes a TCP connection using:

```
host + port
```

Authentication is performed using:

```
user + password
```

The selected schema is:

```
name
```

If the database is unreachable or authentication fails, Lumenta will not start.

### Deployment Notes

- Ensure the database server is reachable from the host running Lumenta.
- Ensure firewall rules allow access to the configured port.
- For container deployments, use the service name as `host`.

Example (Docker Compose):

```
database:
  host: "mariadb"
  port: 3306
  name: "lumenta"
  user: "lumenta"
  password: "${DB_PASSWORD}"
```
---

## Filesystem

The `filesystem` configuration group defines Lumenta’s relationship with the underlying filesystem.

It has two main sections:

- `originals` — where your original images live (strictly read-only)
- `derivatives` — where generated thumbnails (and other derived assets) are stored

### Key Principles

- `originals` are never modified by Lumenta.
- `derivatives` are generated by Lumenta and may be created, overwritten, and removed as needed.
- Neither `originals` nor `derivatives` are exposed as browseable server endpoints.
  - Originals are not accessible through the server at all.
  - Derivatives are only served indirectly (by image ID) after ACL checks.

Relative paths are stored using forward slashes, independent of operating system.

### Example

```
filesystem:
  originals:
    main:
      root: "/mnt/photos"
      excluded:
        - "tmp"
        - "exports"
        - "private/unlisted"
    archive:
      root: "/mnt/archive"
      excluded:
        - "incoming"
  derivatives: "/var/lib/lumenta/derivatives"
```

### originals

`originals` defines one or more source roots. Each root:

- has a unique name (the map key)
- points to an absolute directory path (`root`)
- can declare excluded subdirectories (`excluded`) relative to that root

A root entry looks like:

```
originals:
  <rootName>:
    root: "/absolute/path"
    excluded:
      - "relative/path/to/skip"
```

#### root name

The root name is the identifier of the source root.

It is stored in the database together with the image’s path relative to that root.

This allows Lumenta to represent an image location as:

```
(rootName, relativePath)
```

Example:

- rootName: `main`
- root: `/mnt/photos`
- file: `/mnt/photos/2024/Iceland/IMG_0001.JPG`
- stored relativePath: `2024/Iceland/IMG_0001.JPG`

#### root

Absolute path to the root directory that contains original images.

Examples:

```
root: "/mnt/photos"
root: "/srv/storage/photos"
```

#### excluded

List of subdirectories to skip during scanning.

All entries must be paths relative to the root.

Examples:

```
excluded:
  - "tmp"
  - "exports"
  - "private/unlisted"
  - "2023/Rejected"
```

Notes:

- Exclusions apply only within the given root.
- Use relative paths (do not start with `/`).
- Excluded directories are not scanned and their contents are ignored.

### derivatives

Absolute path to the directory where Lumenta stores generated thumbnails and other derived assets.

Examples:

```
derivatives: "/var/lib/lumenta/derivatives"
derivatives: "/mnt/cache/lumenta-derivatives"
```

Behavior:

- Lumenta creates required subdirectories under this location.
- Generated files are placed here and may be replaced during regeneration.

### Server Exposure and Security Model

Filesystem paths are not directly mapped to HTTP endpoints:

- Original files are never served directly and cannot be accessed via URL.
- The derivatives directory is not exposed as a static file server.
  They are only served through Lumenta’s image endpoints using image IDs, and only after ACL checks succeed.

---

## Authentication

The `authentication` configuration group defines how identity and access control are resolved.

Lumenta supports two external authentication providers:

- `forward`
- `oidc`

There is no standalone JWT-only mode and no disabled mode.

### Example

```
authentication:
  mode: "forward"      # forward | oidc
  guest_enabled: true

  forward:
    user_header: "X-Forwarded-User"
    groups_header: "X-Forwarded-Groups"
    trusted_proxy_cidr:
      - "127.0.0.1/32"
    admin_role: "admin"

  oidc:
    issuer: "https://auth.example.com"
    client_id: "lumenta"
    admin_role: "admin"

  jwt:
    secret: "change-me"
```

### Core Authentication Model

Authentication is resolved per request using two sources:

1. External provider (`forward` or `oidc`)
2. Internal JWT token

The JWT is not an authority source.  
It is used only as a session cache.

### Role Resolution Rules

#### Admin Role

Admin privileges require external authentication.

Admin is granted only if:

- an external authentication context exists, and
- the user belongs to the configured `admin_role`

If no external authentication is present:

- admin privileges are impossible
- the highest possible role is `user`

#### User Role

A user is considered authenticated if:

- external authentication, or
- a valid JWT token

If only a JWT is present:

- the user is treated as authenticated
- role is set to `user`
- no admin privileges are possible

#### Guest

If neither external authentication nor JWT is present:

- the request is treated as guest (if `guest_enabled` is true)

### Authentication Resolution Matrix

Authentication is resolved per request by evaluating:

- presence of external authentication
- presence of a valid JWT
- username consistency (if both exist)

The resulting context is determined as follows:

### Authentication Resolution Matrix

For each request, authentication is resolved according to the following complete decision matrix.

| External Auth | JWT Present | Username Match | Final Identity Source | Final Role | JWT Used | Result |
|--------------|------------|----------------|------------------------|-----------|----------|--------|
| No           | No         | —              | None                   | guest     | No       | Guest context (if `guest_enabled` is true) |
| No           | Yes        | —              | JWT                    | user      | Yes      | Authenticated user (no admin possible) |
| Yes          | No         | —              | External               | user      | No       | Authenticated user |
| Yes          | No         | —              | External               | admin     | No       | Admin (if user has `admin_role`) |
| Yes          | Yes        | Yes            | External               | user      | Yes (ID only) | Authenticated user |
| Yes          | Yes        | Yes            | External               | admin     | Yes (ID only) | Admin (if user has `admin_role`) |
| Yes          | Yes        | No             | External               | user      | No (discarded) | Authenticated user |
| Yes          | Yes        | No             | External               | admin     | No (discarded) | Admin (if user has `admin_role`) |

- External authentication is always authoritative when present.
- JWT never determines the role.
- JWT is used only to recover internal user ID when usernames match.
- If usernames differ, the JWT is ignored.
- Admin role is possible only when:
  - external authentication is present, and
  - the user belongs to `admin_role`.
- Without external authentication, admin privileges are impossible.

### JWT Behavior

```
jwt:
  secret: "change-me"
```

The JWT:

- stores the username
- may store internal user ID
- is validated on each request

If both JWT and external authentication are present:

- the usernames must match
- if they differ, the JWT is discarded

The JWT is used only to avoid repeated database lookups and to preserve session continuity.

### Forward Authentication

Used in reverse proxy setups (e.g., Traefik, Authelia).

```
forward:
  user_header: "X-Forwarded-User"
  groups_header: "X-Forwarded-Groups"
  trusted_proxy_cidr:
    - "127.0.0.1/32"
  admin_role: "admin"
```

- `user_header` contains the authenticated username.
- `groups_header` contains group membership.
- `trusted_proxy_cidr` restricts which IP ranges may inject authentication headers.
- `admin_role` defines which group grants admin privileges.

Forward headers are trusted only when the request originates from a configured CIDR.

### OIDC

OIDC validates identity using an external OpenID Connect provider.

```
oidc:
  issuer: "https://auth.example.com"
  client_id: "lumenta"
  admin_role: "admin"
```

- `issuer` identifies the OIDC provider.
- `client_id` must match the provider configuration.
- `admin_role` defines which claim grants admin privileges.

### Security Model Summary

- External authentication is authoritative.
- JWT is a secondary session mechanism.
- Admin privileges are impossible without external authentication.
- Role resolution happens per request.
- Authentication context is attached before route handling.



---