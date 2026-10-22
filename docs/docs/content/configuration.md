# Configuration

### TOML Configuration file
One or more TOML files can be read by passing `--config config.toml` multiple times. Apart from a few low level configuration variables and the database configuration, all other settings can be managed from the `Settings` dashboard on the admin UI.

To generate a new sample configuration file, run `listnun --new-config`

### Environment variables
Variables defined in config.toml can also be provided as environment variables prefixed by `LISTNUN_` with periods replaced by `__` (double underscore). To start listnun purely with environment variables without a configuration file, set the environment variables and pass the config flag as `--config=""`.

Supported variables:

| **Environment variable**       | Example value  |
| ------------------------------ | -------------- |
| `LISTNUN_app__address`        | "0.0.0.0:9000" |
| `LISTNUN_db__host`            | db             |
| `LISTNUN_db__port`            | 9432           |
| `LISTNUN_db__user`            | listnun       |
| `LISTNUN_db__password`        | listnun       |
| `LISTNUN_db__database`        | listnun       |
| `LISTNUN_db__ssl_mode`        | disable        |


### listnun-specific settings

These are only in the configuration file (or environment), not in the Settings screens.

| Setting | Default | What it does |
| :------ | :------ | :----------- |
| `app.multi_tenancy_enabled` | `false` | Host many workspaces on one instance. See [Workspaces and the operator API](operators/workspaces.md). |
| `app.root_domain` | | With multi-tenancy, workspaces live at `<slug>.<root_domain>`. |
| `app.console_url` | | Where a suspended workspace's admin app redirects (`/billing` for suspended, the root for disabled). Empty returns a plain 503. |
| `app.webhooks_allow_private` | `false` | Let **Developers › Webhooks** call plain HTTP and private or loopback addresses. For local development only: on a shared install it lets workspaces reach internal hosts. |
| `operator.token` | | Turns on the operator API with this bearer token. Empty keeps it off. |
| `operator.db_user`, `operator.db_password` | | A separate Postgres role with `BYPASSRLS` for the operator API's cross-workspace reads. |
| `operator.env` | | `dev` adds a `-dev` suffix to generated workspace hosts, so dev and prod can share a root domain. |
| `inactive_cleanup.managed.*` | 180 days, either, unsubscribe, Sundays 02:00 | The rules workspaces get in **Inactive cleanup › Managed** mode: `days` (30–3650), `activity` (`opened`, `clicked` or `either`), `action` (`unsubscribe`, `blocklist` or `delete`), `list_ids` (empty for all lists) and `schedule` (a cron expression). Invalid values fall back to the defaults, with a log line. |

```toml
[inactive_cleanup.managed]
days = 180
activity = "either"
action = "unsubscribe"
list_ids = []
schedule = "0 2 * * 0"
```

### Customizing system templates
See [system templates](templating.md#system-templates).


### HTTP routes
When configuring auth proxies and web application firewalls, use this table.

#### Private admin endpoints.

| Methods | Route              | Description             |
| ------- | ------------------ | ----------------------- |
| `*`     | `/api/*`           | Admin APIs, the Developers API and the operator API. Your own systems calling the API need this too. |
| `GET`   | `/admin/*`         | Admin UI and HTML pages |
| `POST`  | `/webhooks/bounce` | Admin bounce webhook    |


#### Public endpoints to expose to the internet.

| Methods     | Route                 | Description                                   |
| ----------- | --------------------- | --------------------------------------------- |
| `GET, POST` | `/subscription/*`     | HTML subscription pages                       |
| `GET, `     | `/link/*`             | Tracked link redirection                      |
| `GET`       | `/campaign/*`         | Pixel tracking image                          |
| `GET`       | `/public/*`           | Static files for HTML subscription pages      |
| `GET`       | `/archive*`           | The public campaign archive and its RSS feed (`/archive.xml`) |
| `POST`      | `/webhooks/service/*` | Bounce webhook endpoints for SES, Azure ACS, Sendgrid, Postmark and other supported providers |
| `POST`      | `/webhooks/scrub/batch` | Mail Validation (Scrub) results |
| `GET`       | `/uploads/*`          | The file upload path configured in media settings |
| `GET`       | `/health`             | Health check for load balancers |


## Media uploads

#### Using filesystem

When configuring `docker` volume mounts for using filesystem media uploads, you can follow either of two approaches. The second may be necessary if your setup requires `sudo` for docker commands. For S3-compatible storage instead, see [Media storage](operators/media-storage.md).

After making any changes you will need to run `sudo docker compose stop ; sudo docker compose up`. 

And under `https://listnun.mysite.com/admin/settings` you put `/listnun/uploads`. 

#### Using volumes

Using `docker volumes`, you can specify the name of volume and destination for the files to be uploaded inside the container.


```yml
app:
    volumes:
      - type: volume
        source: listnun-uploads
        target: /listnun/uploads

volumes:
  listnun-uploads:
```

!!! note

    This volume is managed by `docker` itself, and you can see find the host path with `docker volume inspect listnun_listnun-uploads`.

#### Using bind mounts

```yml
  app:
    volumes:
      - ./path/on/your/host/:/path/inside/container
```
Eg:
```yml
  app:
    volumes:
      - ./data/uploads:/listnun/uploads
```
The files will be available inside `/data/uploads` directory on the host machine.

To use the default `uploads` folder:
```yml
  app:
    volumes:
      - ./uploads:/listnun/uploads
```

## Logs

### Docker

https://docs.docker.com/engine/reference/commandline/logs/
```
sudo docker logs -f
sudo docker logs listnun_app -t
sudo docker logs listnun_db -t
sudo docker logs --help
```
Container info: `sudo docker inspect listnun_listnun`

Docker logs to `/dev/stdout` and `/dev/stderr`. The logs are collected by the docker daemon and stored in your node's host path (by default). The same can be configured (/etc/docker/daemon.json) in your docker daemon settings to setup other logging drivers, logrotate policy and more, which you can read about [here](https://docs.docker.com/config/containers/logging/configure/).

### Binary

listnun logs to `stdout`, which is usually not saved to any file. To save listnun logs to a file use `./listnun > listnun.log`.

**Settings › Logs** in the admin app is each workspace's own activity log (settings changes, campaign and import events, webhook failures), kept in the database for 30 days. It isn't the server log.

For a systemd service, you can use `ExecStart=/bin/bash -ce "exec /usr/bin/listnun --config /etc/listnun/config.toml --static-dir /etc/listnun/static >>/etc/listnun/listnun.log 2>&1"` to create a log file that persists after restarts. [More info](https://github.com/knadh/listmonk/issues/1462#issuecomment-1868501606).


## Time zone

To change listnun's time zone (logs, etc.) edit `docker-compose.yml`:
```
environment:
    - TZ=Etc/UTC
```
with any Timezone listed [here](https://en.wikipedia.org/wiki/List_of_tz_database_time_zones). Then run `sudo docker-compose stop ; sudo docker-compose up` after making changes.

## SMTP

### Retries
**Settings › SMTP › Retries** denotes the number of times a message that fails at the moment of sending is retried silently using different connections from the SMTP pool. The messages that fail even after retries are the ones that are logged as errors and ignored.

## SMTP ports
Some server hosts block outgoing SMTP ports (25, 465). You may have to contact your host to unblock them before being able to send e-mails. Eg: [Hetzner](https://docs.hetzner.com/cloud/servers/faq/#why-can-i-not-send-any-mails-from-my-server).


## Performance

### Batch size

The batch size parameter is useful when working with very large lists with millions of subscribers for maximising throughput. It is the number of subscribers that are fetched from the database sequentially in a single cycle (~5 seconds) when a campaign is running. Increasing the batch size uses more memory, but reduces the round trip to the database.
