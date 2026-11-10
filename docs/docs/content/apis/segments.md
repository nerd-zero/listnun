# API / Segments

A segment is a group of subscribers described by conditions instead of SQL. See [Segments](../guides/subscribers.md#segments) for how they work in the app.

| Method | Endpoint                                        | Description                                |
| :----- | :---------------------------------------------- | :----------------------------------------- |
| GET    | [/api/segments](#get-apisegments)               | Retrieve the workspace's saved segments.   |
| POST   | [/api/segments](#post-apisegments)              | Save a segment.                            |
| PUT    | [/api/segments/{id}](#put-apisegmentsid)        | Rename a saved segment or change its conditions. |
| DELETE | [/api/segments/{id}](#delete-apisegmentsid)     | Delete a saved segment.                    |
| POST   | [/api/segments/count](#post-apisegmentscount)   | Count the people in one or more segments.  |
| POST   | [/api/segments/sql](#post-apisegmentssql)       | Get the SQL condition a segment compiles to. |

Reading and counting segments needs `subscribers:get_all` or `subscribers:get`. An [API key](../developers/api-keys.md) with the `subscribers:read` scope has this. Saving, changing and deleting them needs `subscribers:manage`, which a key gets from the `subscribers:manage` scope.

______________________________________________________________________

## The segment object

```json
{
  "match": "all",
  "conditions": [
    {"field": "attribute", "key": "city", "op": "is", "value": "Harare"},
    {"field": "joined", "op": "in_last", "value": "90"}
  ]
}
```

| Name       | Type   | Description |
| :--------- | :----- | :---------- |
| match      | string | `all`: people must meet every condition. `any`: one is enough. Defaults to `all`. |
| conditions | array  | Up to 20 conditions. An empty array means everyone. |

Each condition has a `field`, an `op`, and, depending on the op, a `value`:

| field          | ops                                              | value |
| :------------- | :----------------------------------------------- | :---- |
| `attribute`    | `is`, `is_not`, `contains`, `exists`             | Text. Also needs `key`, the attribute's name. |
| `email`        | `is`, `is_not`, `contains`, `exists`             | Text. |
| `name`         | `is`, `is_not`, `contains`, `exists`             | Text. |
| `joined`       | `in_last`, `not_in_last`, `before`, `after`      | Days (1–3650) for `in_last`/`not_in_last`, a `YYYY-MM-DD` date for `before`/`after`. |
| `list`         | `is`, `is_not`                                   | A list ID. Unsubscribed people don't count as on the list. |
| `status`       | `is`, `is_not`                                   | `enabled` or `blocklisted`. |
| `unconfirmed`  | `is`                                             | None. Matches people with a double opt-in subscription waiting for confirmation. |
| `last_opened`  | `in_last`, `not_in_last`, `before`, `after`, `never` | As for `joined`; none for `never`. |
| `last_clicked` | `in_last`, `not_in_last`, `before`, `after`, `never` | As for `joined`; none for `never`. |

`contains` is case-insensitive. `exists` matches a non-empty value. An invalid segment is rejected with HTTP 400 and a message saying which condition is wrong.

## Querying subscribers by segment

These endpoints take an optional `segment` parameter, a segment object as JSON. It narrows the people they act on and, unlike the SQL `query` parameter, needs no `subscribers:sql_query` permission:

| Endpoint | How to pass it |
| :------- | :------------- |
| `GET /api/subscribers` | `segment` query parameter (URL-encoded JSON). |
| `GET /api/subscribers/export` | `segment` query parameter (URL-encoded JSON). |
| `POST /api/subscribers/query/delete`, `PUT /api/subscribers/query/blocklist`, `PUT /api/subscribers/query/lists` | `segment` field in the JSON body. |

If you also pass `query`, people must match both.

```shell
curl -u "my-service:$LISTNUN_API_KEY" -G 'https://yourcompany.listnun.app/api/subscribers' \
  --data-urlencode 'segment={"match":"all","conditions":[{"field":"last_opened","op":"in_last","value":"90"}]}'
```

______________________________________________________________________

#### GET /api/segments

Retrieve the workspace's saved segments.

##### Example Response

```json
{
  "data": [
    {
      "id": 1,
      "uuid": "6c1e7f0a-0b7a-4d43-9a8e-3f2f2b9c1d11",
      "name": "Harare, joined recently",
      "match": "all",
      "conditions": [
        {"field": "attribute", "key": "city", "op": "is", "value": "Harare"},
        {"field": "joined", "op": "in_last", "value": "90"}
      ],
      "created_by_name": "Ada",
      "created_at": "2026-10-02T09:14:00Z",
      "updated_at": "2026-10-02T09:14:00Z"
    }
  ]
}
```

______________________________________________________________________

#### POST /api/segments

Save a segment. The body is a segment object plus a `name`. Returns the saved segment.

##### Example Request

```shell
curl -u "username:token" -X POST 'https://yourcompany.listnun.app/api/segments' \
  -H 'Content-Type: application/json' \
  -d '{
    "name": "Harare, joined recently",
    "match": "all",
    "conditions": [
      {"field": "attribute", "key": "city", "op": "is", "value": "Harare"},
      {"field": "joined", "op": "in_last", "value": "90"}
    ]
  }'
```

______________________________________________________________________

#### PUT /api/segments/{id}

Rename a saved segment and replace its conditions. Takes the same body as `POST /api/segments` and returns the updated segment.

______________________________________________________________________

#### DELETE /api/segments/{id}

Delete a saved segment. The people in it aren't affected.

______________________________________________________________________

#### POST /api/segments/count

Count the people in up to 30 segments at once, among the lists you can see. Returns the counts in the order given.

##### Example Request

```shell
curl -u "my-service:$LISTNUN_API_KEY" -X POST 'https://yourcompany.listnun.app/api/segments/count' \
  -H 'Content-Type: application/json' \
  -d '{"segments": [
    {"match": "all", "conditions": []},
    {"match": "all", "conditions": [{"field": "last_opened", "op": "never"}]}
  ]}'
```

##### Example Response

```json
{"data": [5210, 812]}
```

______________________________________________________________________

#### POST /api/segments/sql

Return the SQL condition a segment compiles to, for use as the `query` parameter of the subscriber endpoints. A segment with no conditions compiles to `TRUE`.

##### Example Request

```shell
curl -u "my-service:$LISTNUN_API_KEY" -X POST 'https://yourcompany.listnun.app/api/segments/sql' \
  -H 'Content-Type: application/json' \
  -d '{"match": "all", "conditions": [{"field": "email", "op": "contains", "value": "@example.com"}]}'
```

##### Example Response

```json
{"data": {"sql": "((subscribers.email ILIKE '%@example.com%'))"}}
```
