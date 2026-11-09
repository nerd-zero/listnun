# API / Settings

The workspace's settings themselves are read and written with `GET` and `PUT /api/settings`. These endpoints sit alongside them.

| Method | Endpoint | Description |
| :----- | :------- | :---------- |
| GET    | [/api/settings/logs](#get-apisettingslogs) | The workspace's activity log. |
| GET    | [/api/settings/inactive-cleanup](#get-apisettingsinactive-cleanup) | Inactive cleanup's status. |
| POST   | [/api/settings/inactive-cleanup/preview](#post-apisettingsinactive-cleanuppreview) | Count who some cleanup rules would act on. |
| POST   | [/api/settings/scrub/test](#post-apisettingsscrubtest) | Check an email validation (Scrub) URL and key. |

Reading needs `settings:get`; the Scrub test needs `settings:manage`.


______________________________________________________________________

#### GET /api/settings/logs

The workspace's [activity log](../guides/settings.md#activity-log), newest first. `limit` sets how many lines to return, up to 500. Lines are kept for 30 days.

##### Example Response

```json
{
  "data": [
    {
      "id": 812,
      "created_at": "2026-10-22T18:10:03Z",
      "level": "error",
      "source": "sending",
      "message": "The mail server rejected the login. …"
    }
  ]
}
```

`level` is `info`, `warn` or `error`. `source` says what wrote the line, such as `sending` for send failures.


______________________________________________________________________

#### GET /api/settings/inactive-cleanup

The state of [inactive cleanup](../guides/settings.md#inactive-subscribers).

| Field | Description |
| :---- | :---------- |
| managed | listnun's managed rules, as used in Managed mode. |
| matching | How many subscribers the saved rules would act on now. |
| tracking | `false` when individual subscriber tracking is off, which stops cleanup from running. |
| next_run | When it runs next. |
| last_run | The last run: when, its mode and action, how many matched and were done, and any error. |


______________________________________________________________________

#### POST /api/settings/inactive-cleanup/preview

Count how many subscribers some rules would act on, without saving them. Returns a number.

##### Example Request

```shell
curl -u "api_user:token" -X POST 'http://localhost:9000/api/settings/inactive-cleanup/preview' \
  -H 'Content-Type: application/json' \
  -d '{"days": 180, "activity": "either", "action": "unsubscribe", "list_ids": [], "schedule": "0 2 * * 0"}'
```

| Field | Description |
| :---- | :---------- |
| days | No activity for this many days. |
| activity | What counts as activity: `opened`, `clicked` or `either`. |
| action | `unsubscribe`, `blocklist` or `delete`. |
| list_ids | Only subscribers of these lists (and, for `unsubscribe`, only from these lists). Empty means all lists. |
| schedule | When it runs, as a 5-field cron expression. |

##### Example Response

```json
{"data": 640}
```


______________________________________________________________________

#### POST /api/settings/scrub/test

Check that a Scrub URL and API key work before saving them. Returns HTTP 200 if they do, or 400 with the reason.

```shell
curl -u "api_user:token" -X POST 'http://localhost:9000/api/settings/scrub/test' \
  -H 'Content-Type: application/json' \
  -d '{"url": "https://scrub.example.com", "api_key": "…"}'
```
