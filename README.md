# Tailscale Load Balancer

This project is a basic load-balancer for forwarding [Tailscale][] TCP traffic.
This is useful for setting up [virtual IPs for services on Tailscale][].

[Tailscale]: https://tailscale.com/
[virtual IPs for services on Tailscale]: https://github.com/tailscale/tailscale/issues/465

## Status

**This project is largely a proof-of-concept/prototype.**
Having [virtual IPs for services on Tailscale][] has been discussed upstream
and may land eventually,
but I had an immediate need for this on my own Tailnet.

I'm sharing the code for this in the interest of
sharing the results of my experimentation,
but I don't have a ton of time to spare for this particular project.
Bugfixes welcome, but don't expect huge feature development or production-readiness.
If you find this program useful, consider [sponsoring me][]!

[sponsoring me]: https://github.com/sponsors/zombiezen

## Installation

tailscale-lb is distributed as a [Docker][] image:

```shell
docker pull ghcr.io/zombiezen/tailscale-lb
```

You can check out the available tags on [GitHub Container Registry][].

Alternatively, if you're using [Nix][], you can install the binary
by checking out the repository and running the following:

```shell
nix-env --file . --install
```

Or if you're using Nix flakes:

```shell
nix profile install github:zombiezen/tailscale-lb
```

If you are deploying to Kubernetes, example manifests are provided in the `deploy` folder that can also be built with `kustomize`. Make sure to update the value of `TAILSCALE_AUTH_KEY` in secret.yaml to be an authentication key that you have generated from your [Tailscale Console][].

```shell
kubectl apply -k deploy
```

[Tailscale Console]: https://login.tailscale.com/admin/settings/keys
[Docker]: https://www.docker.com/
[GitHub Container Registry]: https://github.com/zombiezen/tailscale-lb/pkgs/container/tailscale-lb
[Nix]: https://nixos.org/

## Usage

Create a configuration file:

```ini
# This is the hostname that will show up in the Tailscale console
# and be used by MagicDNS.
hostname = example
# (Optional) Use an authentication key from https://login.tailscale.com/admin/settings/keys
# If you don't provide an auth key,
# tailscale-lb will log a URL to visit in your browser to authenticate it.
auth-key = tskey-foo

# State persistence options (choose one):

# Option 1: File-based storage (default for non-ephemeral nodes)
# If given, the load balancer will be non-ephemeral
# and persist state in the given directory.
# If the path is relative, it is resolved relative to
# the directory the configuration file is located in.
state-directory = /var/lib/tailscale-lb

# Option 2: SQL database storage (PostgreSQL, MySQL, or SQLite)
# Use this for production HA deployments or shared state across multiple instances.
# See SQL_STORE.md for detailed setup instructions.
# sql-driver = postgres
# sql-dsn = postgres://user:password@localhost:5432/tailscale_lb?sslmode=require

# Examples:
# PostgreSQL: sql-dsn = postgres://user:password@host:5432/dbname?sslmode=require
# MySQL:      sql-dsn = user:password@tcp(host:3306)/dbname
# SQLite:     sql-dsn = file:/var/lib/tailscale-lb/state.db?cache=shared&mode=rwc

# (Optional) Specify the coordination server URL
# control-url = https://headscale.example

# For each port you want to listen on,
# add a section like this:
[tcp 22]
# ... and then add one or more backends.
# tailscale-lb will round-robin TCP connections
# among the various IP addresses it discovers.
# A backend can be one of:

# a) An IPv4 address. If the port is omitted, then the section's port is used.
backend = 127.0.0.1:22

# b) An IPv6 address. If the port is omitted, then the section's port is used.
backend = [2001:db8::1234]:22

# c) A DNS name. If the port is omitted, then the section's port is used.
backend = example.com:22

# d) SRV records. The port is obtained from the SRV record.
# Priority and weight are ignored.
backend = srv _ssh._tcp.example.com

# For each HTTP port you want to listen on,
# add a section like this:
[http 80]

# Backends are specified the same as above.
backend = 127.0.0.1:80

# Add the following request headers (default true):
# Tailscale-User: The connecting user's email address
# Tailscale-Name: The connecting user's display name
# Tailscale-Profile-Picture: A URL to the connecting user's profile picture
whois = true
# Use the MagicDNS HTTPS Certificates described in https://tailscale.com/kb/1153/enabling-https/
# (default false)
tls = false
# Whether to use the request-supplied X-Forwarded-For (default false).
trust-x-forwarded-for = false
```

Then run tailscale-lb with the configuration file as its argument.
If you're using Docker:

```shell
docker run --rm \
  --volume "$(pwd)/foo.ini":/etc/tailscale-lb.ini \
  ghcr.io/zombiezen/tailscale-lb /etc/tailscale-lb.ini
```

Or if you're using a standalone binary:

```shell
tailscale-lb foo.ini
```

You can then see the load balancer's IP address in the logs
or in the Tailscale admin console.

## SQL State Storage

tailscale-lb supports persistent state storage in SQL databases for production deployments. This enables:

- **High Availability**: Share state across multiple load balancer instances
- **Durability**: Use enterprise database backup and recovery tools
- **Flexibility**: Deploy on cloud platforms without persistent volumes

### Supported Databases

- **PostgreSQL** - Recommended for production HA deployments
- **MySQL/MariaDB** - Good for existing MySQL infrastructure
- **SQLite** - Suitable for single-node deployments

### Quick Example

```ini
hostname = my-loadbalancer
auth-key = tskey-auth-YOUR_KEY

# Use PostgreSQL for state storage
sql-driver = postgres
sql-dsn = postgres://user:password@localhost:5432/tailscale_lb?sslmode=require

[tcp 22]
backend = 127.0.0.1:22
```

### High Availability Setup

Multiple tailscale-lb instances can share the same PostgreSQL database:

```
Instance 1 ──┐
             ├──▶ PostgreSQL (shared state)
Instance 2 ──┘
```

Each instance uses a unique `hostname` but shares the same `sql-dsn`.

For complete setup instructions, configuration examples, and deployment guides, see **[SQL_STORE.md](SQL_STORE.md)**.

## License

[Apache 2.0](LICENSE)

