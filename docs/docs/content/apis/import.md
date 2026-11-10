# API / Import

Method   | Endpoint                                        | Description
---------|-------------------------------------------------|------------------------------------------------
GET      | [/api/import/subscribers](#get-apiimportsubscribers) | Retrieve import statistics.
GET      | [/api/import/subscribers/logs](#get-apiimportsubscriberslogs) | Retrieve import logs.
POST     | [/api/import/subscribers](#post-apiimportsubscribers) | Upload a file for bulk subscriber import.
DELETE   | [/api/import/subscribers](#delete-apiimportsubscribers) | Stop and remove an import.
POST     | [/api/import/subscribers/file](#post-apiimportsubscribersfile) | Upload a file and get its columns, to import later.
POST     | [/api/import/subscribers/check](#post-apiimportsubscriberscheck) | Dry-run an import of an uploaded file.
GET      | [/api/import/subscribers/problems](#get-apiimportsubscribersproblems) | Rows the current or last import left out.
GET      | [/api/import/subscribers/history](#get-apiimportsubscribershistory) | Recent finished imports.

The app imports in two stages: it uploads the file first (`/file`), matches its columns and dry-runs the import (`/check`), then starts it with `POST /api/import/subscribers`, passing `file_id` and `columns` instead of a file. Sending a file straight to `POST /api/import/subscribers` still works as before. All of these need the `subscribers:import` permission; an [API key](../developers/api-keys.md) needs the `subscribers:import` scope, which brings `lists:manage` with it so the import can add people to lists.

______________________________________________________________________

#### GET /api/import/subscribers

Retrieve the status of an ongoing import.

##### Example Request

```shell
curl -u "api_user:token" -X GET 'http://localhost:9000/api/import/subscribers'
```

##### Example Response

```json
{
    "data": {
        "name": "",
        "total": 0,
        "imported": 0,
        "status": "none"
    }
}
```

______________________________________________________________________

#### GET /api/import/subscribers/logs

Retrieve logs from an ongoing import.

##### Example Request

```shell
curl -u "api_user:token" -X GET 'http://localhost:9000/api/import/subscribers/logs'
```

##### Example Response

```json
{
    "data": "2020/04/08 21:55:20 processing 'import.csv'\n2020/04/08 21:55:21 imported finished\n"
}
```

______________________________________________________________________

#### POST /api/import/subscribers

Send a CSV (optionally ZIP compressed) file to import subscribers. Use a multipart form POST.

##### Parameters

| Name   | Type        | Required | Description                              |
|:-------|:------------|:---------|:-----------------------------------------|
| params | JSON string | Yes      | Stringified JSON with import parameters. |
| file   | file        | Yes      | File for upload.                         |


#### `params` (JSON string)
| Name      | Type     | Required | Description                                                                                                                        |
|:----------|:---------|:---------|:-----------------------------------------------------------------------------------------------------------------------------------|
| mode      | string   | Yes      | `subscribe` or `blocklist`                                                                                                         |
| delim     | string   | Yes      | Single character indicating delimiter used in the CSV file, eg: `,`                                                                |
| lists     | []number |          | Array of list IDs to subscribe to.                                                                                                 |
| overwrite | bool     |          | Whether to overwrite the subscriber parameters including subscriptions or ignore records that are already present in the database. |
| file_id   | string   |          | A file uploaded earlier with [`POST /api/import/subscribers/file`](#post-apiimportsubscribersfile), imported instead of a `file` sent with this request. |
| columns   | []object |          | What each column is saved as: `{"index": 0, "as": "email"}`. `as` is `email`, `name`, `attribute` (with `key`, the attribute name), `attributes` (a JSON object column) or `skip`. Without it, columns named `email`, `name` and `attributes` are imported and the rest ignored. |

##### Example Request

```shell
curl -u "api_user:token" -X POST 'http://localhost:9000/api/import/subscribers' \
  -F 'params={"mode":"subscribe", "subscription_status":"confirmed", "delim":",", "lists":[1, 2], "overwrite": true}' \
  -F "file=@/path/to/subs.csv"
```

##### Example Response

```json
    {
        "mode": "subscribe", // subscribe or blocklist
        "delim": ",",        // delimiter in the uploaded file
        "lists":[1],         // array of list IDs to import into
        "overwrite": true    // overwrite existing entries or skip them?
    }
```

______________________________________________________________________

#### DELETE /api/import/subscribers

Stop and delete an ongoing import.

##### Example Request

```shell
curl -u "api_user:token" -X DELETE 'http://localhost:9000/api/import/subscribers' 
```

##### Example Response

```json
{
    "data": {
        "name": "",
        "total": 0,
        "imported": 0,
        "status": "none"
    }
}
```

______________________________________________________________________

#### POST /api/import/subscribers/file

Upload a CSV, or a ZIP with one CSV inside (up to 50 MB), as the multipart field `file`. The file is kept for two hours. Returns its ID, the detected delimiter, the row count, and each column with its first values and a guess at what it holds.

##### Example Response

```json
{
  "data": {
    "file_id": "a1b2c3…",
    "name": "customers.csv",
    "size": 18231,
    "rows": 412,
    "delim": ",",
    "uploaded_at": "2026-10-22T18:06:41Z",
    "columns": [
      {"index": 0, "header": "Email", "samples": ["ada@example.com", "alan@example.com"], "as": "email"},
      {"index": 1, "header": "City", "samples": ["London", ""], "as": "attribute", "key": "city"}
    ]
  }
}
```


______________________________________________________________________

#### POST /api/import/subscribers/check

Check an uploaded file before importing it. Nothing is written.

##### Example Request

```shell
curl -u "api_user:token" -X POST 'http://localhost:9000/api/import/subscribers/check' \
  -H 'Content-Type: application/json' \
  -d '{"file_id": "a1b2c3…", "columns": [{"index": 0, "as": "email"}, {"index": 1, "as": "attribute", "key": "city"}]}'
```

##### Example Response

```json
{
  "data": {
    "rows": 412,
    "valid": 409,
    "existing": 57,
    "num_problems": 3,
    "problems": [
      {"row": 18, "value": "ada@@example", "reason": "invalid_email"}
    ]
  }
}
```

`existing` is how many valid rows are already subscribers. `row` is the row number in the spreadsheet. `reason` is `no_email`, `invalid_email`, `blocked_domain` or `columns` (wrong number of columns).


______________________________________________________________________

#### GET /api/import/subscribers/problems

The rows the current or last import left out, as `[{"row", "value", "reason"}]` like the check above.


______________________________________________________________________

#### GET /api/import/subscribers/history

Recent finished imports, newest first.

##### Example Response

```json
{
  "data": [
    {
      "id": 9,
      "name": "customers.csv",
      "mode": "subscribe",
      "status": "finished",
      "total": 412,
      "imported": 409,
      "added": 352,
      "skipped": 3,
      "failed": 0,
      "created_at": "2026-10-22T18:06:41Z"
    }
  ]
}
```
