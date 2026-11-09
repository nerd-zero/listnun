# APIs

All features that are available on the listnun dashboard are also available as REST-like HTTP APIs that can be interacted with directly. Request and response bodies are JSON. This allows easy scripting of listnun and integration with other systems, for instance, synchronisation with external subscriber databases.

!!! note
    If you come across API calls that are yet to be documented, please consider contributing to docs.


## Base URL

Every workspace has its own address, and API calls go to it: `https://yourcompany.listnun.app/api/…`. The examples on these pages use `http://localhost:9000`; replace it with your workspace's address.

## Auth

Authenticate with an [API key](../developers/api-keys.md): its name and secret in the `Authorization` header.

```shell
curl -H "Authorization: token my-service:lnk_live_…" https://yourcompany.listnun.app/api/developers/messages
```

HTTP Basic auth with the same pair also works:

```shell
curl -u "my-service:lnk_live_…" https://yourcompany.listnun.app/api/developers/messages
```

A key belongs to one workspace and only works on that workspace's address.

## Permissions

A key can only call the endpoints its scopes allow:

| Scope | Endpoints |
| :---- | :-------- |
| `tx:send` | `POST /api/tx`, resending logged messages |
| `tx:logs` | `GET /api/developers/messages…`, `GET /api/developers/stats` |
| `subscribers:read` | Reading subscribers (`GET /api/subscribers…`) |
| `keys:manage` | `/api/developers/keys…` |
| `webhooks:manage` | `/api/developers/webhooks…` |

The other endpoints on these pages (lists, writing subscribers, imports, campaigns, templates, media, bounces) need a user's permissions, set by their [user role and list role](../roles-and-permissions.md). Today they're available to people signed in to the app, and to API users created before API keys existed, which keep working with their `username:token` pair. New API users can't be created, so new integrations are limited to the key scopes above.

______________________________________________________________________

## Response structure

### Successful request

```http
HTTP/1.1 200 OK
Content-Type: application/json

{
    "data": {}
}
```

All responses from the API server are JSON with the content-type application/json unless explicitly stated otherwise. A successful 200 OK response always has a JSON response body with a status key with the value success. The data key contains the full response payload.

### Failed request

```http
HTTP/1.1 500 Server error
Content-Type: application/json

{
    "message": "Error message"
}
```

A failure response is preceded by the corresponding 40x or 50x HTTP header. There may be an optional `data` key with additional payload.

### Timestamps

All timestamp fields are in the format `2019-01-01T09:00:00.000000+05:30`. The seconds component is suffixed by the milliseconds, followed by the `+` and the timezone offset.

### Common HTTP error codes

| Code  |                                                                             |
| ----- | ----------------------------------------------------------------------------|
|  400  | Missing or bad request parameters or values                                 |
|  403  | Session expired or invalidate. Must relogin                                 |
|  404  | Request resource was not found                                              |
|  405  | Request method (GET, POST etc.) is not allowed on the requested endpoint    |
|  410  | The requested resource is gone permanently                                  |
|  422  | Unprocessable entity. Unable to process request as it contains invalid data |
|  429  | Too many requests to the API (rate limiting)                                |
|  500  | Something unexpected went wrong                                             |
|  502  | The backend OMS is down and the API is unable to communicate with it        |
|  503  | Service unavailable; the API is down                                        |
|  504  | Gateway timeout; the API is unreachable                                     |


## OpenAPI (Swagger) spec

Every listnun instance serves its own OpenAPI (Swagger) spec, generated from the code, at `/api/swagger/index.html`, for example `https://yourcompany.listnun.app/api/swagger/index.html`. It covers every endpoint, including the Developers and operator APIs.

