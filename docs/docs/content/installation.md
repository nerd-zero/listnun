# Installation

listnun is a single Go binary, with the admin app and static files built in. Its only dependency is PostgreSQL 12 or newer.

!!! note "Using hosted listnun?"
    You don't need to install anything: your workspace is ready at its own address. This page is for running listnun yourself.

listnun doesn't publish prebuilt binaries or images yet, so you build it from the source on [GitHub](https://github.com/nerd-zero/listnun).

## Build the binary

You need Go, Node.js and Yarn.

```shell
git clone https://github.com/nerd-zero/listnun.git
cd listnun
make dist
```

This builds the frontend and the visual email builder, compiles the Go code, and packs everything into one `listnun` binary.

## Install with the binary

1. `./listnun --new-config` writes a `config.toml`. Edit at least the `[db]` section.
2. `./listnun --install` creates the tables in your Postgres database.
3. `./listnun` starts the server. Open `http://localhost:9000` to create the first admin user and sign in.

!!! tip
    To create the admin user during installation instead, set:
    `LISTNUN_ADMIN_USER=myuser LISTNUN_ADMIN_PASSWORD=xxxxx ./listnun --install`

## Install with Docker

After `make dist`, build the image from the repository's `Dockerfile`, then start it with the repository's `docker-compose.yml`, which runs listnun and Postgres:

```shell
docker build -t listnun/listnun:latest .
docker compose up -d
```

Open `http://localhost:9000` to create the admin user. To set it from the start, run `LISTNUN_ADMIN_USER=myuser LISTNUN_ADMIN_PASSWORD=xxxxx docker compose up -d` the first time.

On every start the container installs the database if needed, then runs any pending upgrades, then starts listnun.

### Mounting a config.toml

The compose file configures listnun through `LISTNUN_*` environment variables. To use a config file instead:

1. Save a `config.toml` on the host:

    ```toml
    [app]
    address = "0.0.0.0:9000"

    [db]
    host = "listnun_db" # The Postgres container's name in the compose file.
    port = 5432
    user = "listnun"
    password = "listnun"
    database = "listnun"
    ssl_mode = "disable"
    max_open = 25
    max_idle = 25
    max_lifetime = "300s"
    ```

2. Mount it in `docker-compose.yml`:

    ```yaml
      app:
        volumes:
        - /path/on/your/host/config.toml:/listnun/config.toml
    ```

3. Point the `command:` at it:

    ```yaml
    command: [sh, -c, "./listnun --install --idempotent --yes --config /listnun/config.toml && ./listnun --upgrade --yes --config /listnun/config.toml && ./listnun --config /listnun/config.toml"]
    ```

## After installing

- Set up email sending under **Settings › SMTP**.
- Put listnun behind a reverse proxy with HTTPS, and set **Settings › General › Root URL** to its public address. Links in emails use it.
- Keep only the public routes reachable from the internet. See [Configuration](configuration.md#http-routes).
- To host several workspaces on one instance, see [Workspaces and the operator API](operators/workspaces.md).
- For media, see [Media storage](operators/media-storage.md).

To work on listnun's code, see [Developer setup](developer-setup.md).
