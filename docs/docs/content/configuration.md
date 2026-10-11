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


### Customizing system templates
See [system templates](templating.md#system-templates).


### HTTP routes
When configuring auth proxies and web application firewalls, use this table.

#### Private admin endpoints.

| Methods | Route              | Description             |
| ------- | ------------------ | ----------------------- |
| `*`     | `/api/*`           | Admin APIs              |
| `GET`   | `/admin/*`         | Admin UI and HTML pages |
| `POST`  | `/webhooks/bounce` | Admin bounce webhook    |


#### Public endpoints to expose to the internet.

| Methods     | Route                 | Description                                   |
| ----------- | --------------------- | --------------------------------------------- |
| `GET, POST` | `/subscription/*`     | HTML subscription pages                       |
| `GET, `     | `/link/*`             | Tracked link redirection                      |
| `GET`       | `/campaign/*`         | Pixel tracking image                          |
| `GET`       | `/public/*`           | Static files for HTML subscription pages      |
| `POST`      | `/webhooks/service/*` | Bounce webhook endpoints for SES, Azure ACS, Sendgrid, and other supported providers |
| `GET`       | `/uploads/*`          | The file upload path configured in media settings |


## Media uploads

#### Using filesystem

When configuring `docker` volume mounts for using filesystem media uploads, you can follow either of two approaches. [The second option may be necessary if](https://github.com/knadh/listmonk/issues/1169#issuecomment-1674475945) your setup requires you to use `sudo` for docker commands. 

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

Settings -> Logs in admin shows the last 1000 lines of the standard log output but gets erased when listnun is restarted.

For the [service file](https://github.com/knadh/listmonk/blob/master/listmonk%40.service), you can use `ExecStart=/bin/bash -ce "exec /usr/bin/listnun --config /etc/listnun/config.toml --static-dir /etc/listnun/static >>/etc/listnun/listnun.log 2>&1"` to create a log file that persists after restarts. [More info](https://github.com/knadh/listmonk/issues/1462#issuecomment-1868501606).


## Time zone

To change listnun's time zone (logs, etc.) edit `docker-compose.yml`:
```
environment:
    - TZ=Etc/UTC
```
with any Timezone listed [here](https://en.wikipedia.org/wiki/List_of_tz_database_time_zones). Then run `sudo docker-compose stop ; sudo docker-compose up` after making changes.

## SMTP

### Retries
The `Settings -> SMTP -> Retries` denotes the number of times a message that fails at the moment of sending is retried silently using different connections from the SMTP pool. The messages that fail even after retries are the ones that are logged as errors and ignored.

## SMTP ports
Some server hosts block outgoing SMTP ports (25, 465). You may have to contact your host to unblock them before being able to send e-mails. Eg: [Hetzner](https://docs.hetzner.com/cloud/servers/faq/#why-can-i-not-send-any-mails-from-my-server).


## Performance

### Batch size

The batch size parameter is useful when working with very large lists with millions of subscribers for maximising throughput. It is the number of subscribers that are fetched from the database sequentially in a single cycle (~5 seconds) when a campaign is running. Increasing the batch size uses more memory, but reduces the round trip to the database.
