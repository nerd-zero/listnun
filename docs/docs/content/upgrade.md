# Upgrade

!!! warning
    Always back up the Postgres database before upgrading.

An upgrade is a new binary plus `--upgrade`, which brings the database schema up to date. Upgrades are idempotent: running `--upgrade` again does nothing once the database is current.

## Binary

1. Build the new version: `git pull && make dist`. See [Installation](installation.md#build-the-binary).
2. Stop listnun, for example `sudo systemctl stop listnun`.
3. Replace the old binary with the new one.
4. Run `./listnun --upgrade`. It runs each pending migration in order and says which.
5. Start listnun again.

## Docker

Rebuild the image from the new code, then restart the app container. The container runs `--upgrade` on start.

```shell
git pull && make dist
docker build -t listnun/listnun:latest .
docker compose up -d app
```

If your `docker-compose.yml` doesn't run `--upgrade` in its `command:`, run it once yourself:

```shell
docker compose run --rm app ./listnun --upgrade
```

## Upgrading to the redesign (v6.20 to v6.33)

The redesign adds 14 migrations, which `--upgrade` applies in order. Nothing needs doing by hand, but some behaviour changes:

| Migration | Change |
| :-------- | :----- |
| v6.20 | Usage metering for billing (`usage_counters`). |
| v6.21–v6.25 | The Developers area: API keys, the transactional message log, transactional templates, email branding and webhooks. |
| v6.26 | Deleting a list role no longer deletes the users who had it; they just lose its list access. |
| v6.27 | Cloudflare Turnstile and the sign-in captcha setting. |
| v6.28 | Media file sizes, for storage reporting. Existing files are measured in the background after the upgrade. |
| v6.29 | Each API user becomes an API key. Old `username:token` pairs keep working until you disable them. See [API keys](developers/api-keys.md#keys-migrated-from-api-users). |
| v6.30 | The "require two-step verification" and maximum upload size settings. |
| v6.31 | The per-workspace activity log (**Settings › Logs**). |
| v6.32 | Inactive cleanup. It starts **off** in every workspace. |
| v6.33 | Campaign tracking settings. Existing campaigns start with click tracking off, so they send as before. New campaigns have it on. |

## Downgrade

A database upgraded by a newer version shouldn't be used with an older one. To go back:

1. Stop listnun.
2. Restore the database backup you took before upgrading.
3. Run the older binary or image.
