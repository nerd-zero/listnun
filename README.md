[![listnun-logo](https://user-images.githubusercontent.com/547147/231084896-835dba66-2dfe-497c-ba0f-787564c0819e.png)](https://github.com/nerd-zero/listnun)

listnun is a standalone, self-hosted, newsletter and mailing list manager. It is fast, feature-rich, and packed into a single binary. It uses a PostgreSQL database as its data store.

listnun is a fork of [listmonk](https://github.com/knadh/listmonk) that adds multi-tenancy and other changes for running it as a hosted service.

[![listnun-dashboard](https://github.com/user-attachments/assets/689b5fbb-dd25-4956-a36f-e3226a65f9c4)](https://github.com/nerd-zero/listnun)

Read the [documentation](https://nerd-zero.github.io/listnun/docs/) for more info.

## Installation

### Docker

The latest image is available on the GitHub Container Registry at `ghcr.io/nerd-zero/listnun:latest`.
Download and use the sample [docker-compose.yml](https://github.com/nerd-zero/listnun/blob/master/docker-compose.yml).


```shell
# Download the compose file to the current directory.
curl -LO https://github.com/nerd-zero/listnun/raw/master/docker-compose.yml

# Run the services in the background.
docker compose up -d
```
Visit `http://localhost:9000`

See [installation docs](https://nerd-zero.github.io/listnun/docs/installation)

__________________

### Binary
- Download the [latest release](https://github.com/nerd-zero/listnun/releases) and extract the listnun binary.
- `./listnun --new-config` to generate config.toml. Edit it.
- `./listnun --install` to setup the Postgres DB (or `--upgrade` to upgrade an existing DB. Upgrades are idempotent and running them multiple times have no side effects).
- Run `./listnun` and visit `http://localhost:9000`

See [installation docs](https://nerd-zero.github.io/listnun/docs/installation)
__________________


## Developers
listnun is free and open source software licensed under AGPLv3. If you are interested in contributing, refer to the [developer setup](https://nerd-zero.github.io/listnun/docs/developer-setup). The backend is written in Go and the frontend is Vue 3 with PrimeVue for UI. 


## License
listnun is licensed under the AGPL v3 license.
