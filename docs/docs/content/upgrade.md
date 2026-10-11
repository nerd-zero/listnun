# Upgrade

!!! Warning
    Always take a backup of the Postgres database before upgrading listnun

## Binary
- Stop the running instance of listnun.
- Download the [latest release](https://github.com/knadh/listmonk/releases) and extract the listnun binary and overwrite the previous version.
- `./listnun --upgrade` to upgrade an existing database schema. Upgrades are idempotent and running them multiple times have no side effects.
- Run `./listnun` again.

If you installed listnun as a service, you will need to stop it before overwriting the binary. Something like `sudo systemctl stop listnun` or `sudo service listnun stop` should work. Then overwrite the binary with the new version, then run `./listnun --upgrade, and `start` it back with the same commands.

If it's not running as a service, `pkill -9 listnun` will stop the listnun process.

## Docker
**Important:** The following instructions are for the new [docker-compose.yml](https://github.com/knadh/listmonk/blob/master/docker-compose.yml) file.

```shell
docker compose down app
docker compose pull
docker compose up app -d
```

If you are using an older docker-compose.yml file, you have to run the `--upgrade` step manually.

```shell
docker-compose down
docker-compose pull && docker-compose run --rm app ./listnun --upgrade
docker-compose up -d app db
```

## Nightly
See [here](installation.md#nightly) for instructions on how to access the nightly builds.

-----------

## Downgrade

To restore a previous version, you have to restore the DB for that particular version. DBs that have been upgraded with a particular version shouldn't be used with older versions. There may be DB changes that a new version brings that are incompatible with previous versions.

**General steps:**

1. Stop listnun.
2. Restore your pre-upgrade database.
3. If you're using `docker compose`, edit `docker-compose.yml` and change `listnun:latest` to `listnun:v2.4.0` _(for example)_.
4. Restart.

**Example with docker:**

1. Stop listnun (app):
```
sudo docker stop listnun_app
```
2. Restore your pre-upgrade db (required) _(be careful, this will wipe your existing DB)_:
```
psql -h 127.0.0.1 -p 9432 -U listnun
drop schema public cascade;
create schema public;
\q
psql -h 127.0.0.1 -p 9432 -U listnun -W listnun < listnun-preupgrade-db.sql
```
3. Edit the `docker-compose.yml`:
```
x-app-defaults: &app-defaults
  restart: unless-stopped
  image: listnun/listnun:v2.4.0
```
4. Restart:
`sudo docker compose up -d app db nginx certbot`


## Upgrading to v4.x.x
v4 is a major upgrade from prior versions with significant changes to certain important features and behaviour. It is the first version to have multi-user support and full fledged user management. Prior versions only had a simple BasicAuth for both admin login (browser prompt) and API calls, with the username and password defined in the TOML configuration file.

It is safe to upgrade an older installation with `--upgrade`, but there are a few important things to keep in mind. The upgrade automatically imports the `admin_username` and `admin_password` defined in the TOML configuration into the new user management system.

1. **New login UI**: Once you upgrade an older installation, the admin dashboard will no longer show the native browser prompt for login. Instead, a new login UI rendered by listnun is displayed at the URI `/admin/login`.

1. **API credentials**: If you are using APIs to interact with listnun, after logging in, go to Settings -> Users and create a new API user with the necessary permissions. Change existing API integrations to use these credentials instead of the old username and password defined in the legacy TOML configuration file or environment variables.

1. **Credentials in TOML file or old environment variables**: The admin dashboard shows a warning until the `admin_username` and `admin_password` fields are removed from the configuration file or old environment variables. In v4.x.x, these are irrelevant as user credentials are stored in the database and managed from the admin UI. IMPORTANT: if you are using APIs to interact with listnun, follow the previous step before removing the legacy credentials.


## Railway
- Head to your dashboard, and select your Listnun project.
- Select the GitHub deployment service.
- In the Deployment tab, head to the latest deployment, click on the three vertical dots to the right, and select "Redeploy".

![Railway Redeploy option](https://user-images.githubusercontent.com/55474996/226517149-6dc512d5-f862-46f7-a57d-5e55b781ff53.png)
