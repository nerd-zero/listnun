# API / Campaigns

| Method | Endpoint                                                                    | Description                               |
| :----- | :-------------------------------------------------------------------------- | :---------------------------------------- |
| GET    | [/api/campaigns](#get-apicampaigns)                                         | Retrieve all campaigns.                   |
| GET    | [/api/campaigns/{campaign_id}](#get-apicampaignscampaign_id)                | Retrieve a specific campaign.             |
| GET    | [/api/campaigns/{campaign_id}/preview](#get-apicampaignscampaign_idpreview) | Retrieve preview of a campaign.           |
| GET    | [/api/campaigns/running/stats](#get-apicampaignsrunningstats)               | Retrieve stats of specified campaigns.    |
| GET    | [/api/campaigns/analytics/engagement](#get-apicampaignsanalyticsengagement) | Retrieve campaigns' results, all time.    |
| GET    | [/api/campaigns/analytics/{type}](#get-apicampaignsanalyticstype)           | Retrieve view counts for a  campaign.     |
| GET    | [/api/campaigns/overview](#get-apicampaignsoverview)                        | What's sending, scheduled, drafted and recently sent. |
| GET    | [/api/campaigns/{campaign_id}/inbox-check](#get-apicampaignscampaign_idinbox-check) | Run the inbox check on a campaign.  |
| GET    | [/api/campaigns/{campaign_id}/replies](#get-apicampaignscampaign_idreplies) | Retrieve a campaign's tracked replies.    |
| POST   | [/api/campaigns](#post-apicampaigns)                                        | Create a new campaign.                    |
| POST   | [/api/campaigns/{campaign_id}/test](#post-apicampaignscampaign_idtest)      | Test campaign with arbitrary subscribers. |
| PUT    | [/api/campaigns/{campaign_id}](#put-apicampaignscampaign_id)                | Update a campaign.                        |
| PUT    | [/api/campaigns/{campaign_id}/status](#put-apicampaignscampaign_idstatus)   | Change status of a campaign.              |
| PUT    | [/api/campaigns/{campaign_id}/archive](#put-apicampaignscampaign_idarchive) | Publish campaign to public archive.       |
| DELETE | [/api/campaigns/{campaign_id}](#delete-apicampaignscampaign_id)             | Delete a campaign.                        |
| DELETE | [/api/campaigns](#delete-apicampaigns)                                      | Delete multiple campaigns.                |

With an [API key](../developers/api-keys.md), reading campaigns, previews, stats and analytics needs the `campaigns:read` scope. Creating, changing, testing and deleting campaigns, and starting, scheduling, pausing or cancelling them, needs `campaigns:manage`. A key with `campaigns:manage` sends real email through the workspace's sending server, just like starting it in the app.

____________________________________________________________________________________________________________________________________

#### GET /api/campaigns

Retrieve all campaigns.

##### Example Request

```shell
 curl -u "api_user:token" -X GET 'http://localhost:9000/api/campaigns?page=1&per_page=100'
```

##### Parameters

| Name     | Type     | Required | Description                                                              |
| :------- | :------- | :------- | :----------------------------------------------------------------------- |
| order    | string   |          | Sorting order: ASC for ascending, DESC for descending.                   |
| order_by | string   |          | Result sorting field. Options: name, status, created_at, updated_at.     |
| query    | string   |          | String to filtter by campaign name and subject (fulltext and substring). |
| status   | []string |          | Status to filter campaigns. Repeat in the query for multiple values.     |
| tags     | []string |          | Tags to filter campaigns. Repeat in the query for multiple values.       |
| page     | number   |          | Page number for paginated results.                                       |
| per_page | number   |          | Results per page. Set as 'all' for all results.                          |
| no_body  | boolean  |          | When set to true, returns response without body content.                 |

##### Example Response

```json
{
    "data": {
        "results": [
            {
                "id": 1,
                "created_at": "2020-03-14T17:36:41.29451+01:00",
                "updated_at": "2020-03-14T17:36:41.29451+01:00",
                "views": 0,
                "clicks": 0,
                "lists": [
                    {
                        "id": 1,
                        "name": "Default list"
                    }
                ],
                "started_at": null,
                "to_send": 0,
                "sent": 0,
                "uuid": "57702beb-6fae-4355-a324-c2fd5b59a549",
                "type": "regular",
                "name": "Test campaign",
                "subject": "Welcome to listnun",
                "from_email": "No Reply <noreply@yoursite.com>",
                "body": "<h3>Hi {{ .Subscriber.FirstName }}!</h3>\n\t\t\tThis is a test e-mail campaign. Your second name is {{ .Subscriber.LastName }} and you are from {{ .Subscriber.Attribs.city }}.",
                "body_source": null,
                "send_at": "2020-03-15T17:36:41.293233+01:00",
                "status": "draft",
                "content_type": "richtext",
                "tags": [
                    "test-campaign"
                ],
                "template_id": 1,
                "messenger": "email"
            }
        ],
        "query": "",
        "total": 1,
        "per_page": 20,
        "page": 1
    }
}
```

______________________________________________________________________

#### GET /api/campaigns/{campaign_id}

Retrieve a specific campaign.

##### Parameters

| Name        | Type    | Required | Description                                              |
| :---------- | :------ | :------- | :------------------------------------------------------- |
| campaign_id | number  | Yes      | Campaign ID.                                             |
| no_body     | boolean |          | When set to true, returns response without body content. |

##### Example Request

```shell
curl -u "api_user:token" -X GET 'http://localhost:9000/api/campaigns/1'
```

##### Example Response

```json
{
    "data": {
        "id": 1,
        "created_at": "2020-03-14T17:36:41.29451+01:00",
        "updated_at": "2020-03-14T17:36:41.29451+01:00",
        "views": 0,
        "clicks": 0,
        "lists": [
            {
                "id": 1,
                "name": "Default list"
            }
        ],
        "started_at": null,
        "to_send": 0,
        "sent": 0,
        "uuid": "57702beb-6fae-4355-a324-c2fd5b59a549",
        "type": "regular",
        "name": "Test campaign",
        "subject": "Welcome to listnun",
        "from_email": "No Reply <noreply@yoursite.com>",
        "body": "<h3>Hi {{ .Subscriber.FirstName }}!</h3>\n\t\t\tThis is a test e-mail campaign. Your second name is {{ .Subscriber.LastName }} and you are from {{ .Subscriber.Attribs.city }}.",
        "body_source": null,
        "send_at": "2020-03-15T17:36:41.293233+01:00",
        "status": "draft",
        "content_type": "richtext",
        "tags": [
            "test-campaign"
        ],
        "template_id": 1,
        "messenger": "email"
    }
}
```

______________________________________________________________________

#### GET /api/campaigns/{campaign_id}/preview

Preview a specific campaign.

##### Parameters

| Name        | Type   | Required | Description             |
| :---------- | :----- | :------- | :---------------------- |
| campaign_id | number | Yes      | Campaign ID to preview. |

##### Example Request

```shell
curl -u "api_user:token" -X GET 'http://localhost:9000/api/campaigns/1/preview'
```

##### Example Response

```html
<h3>Hi John!</h3>
This is a test e-mail campaign. Your second name is Doe and you are from Bengaluru.
```

______________________________________________________________________

#### GET /api/campaigns/running/stats

Retrieve stats of specified campaigns.

##### Parameters

| Name        | Type   | Required | Description                    |
| :---------- | :----- | :------- | :----------------------------- |
| campaign_id | number | Yes      | Campaign IDs to get stats for. |

##### Example Request

```shell
curl -u "api_user:token" -X GET 'http://localhost:9000/api/campaigns/running/stats?campaign_id=1'
```

##### Example Response

```json
{
    "data": []
}
```

______________________________________________________________________

#### GET /api/campaigns/analytics/engagement

Retrieve campaigns' results, all time, counted the same way as on the Campaigns overview and the Analytics page.

- `delivered` is sent less `bounced` (people with a bounce), and is what open and click rates are a share of.
- `opened` is people who viewed or clicked. A click counts as an open, because the email was opened even when images were blocked and the view pixel never loaded.
- `clicked` is people who clicked.
- `anonymous` is views and clicks recorded without a subscriber: with **Individual subscriber tracking** off, or for subscribers since deleted. Each of those counts once in `opened` and `clicked`, so with it above zero they may count a person more than once.

##### Parameters

| Name | Type       | Required | Description                     |
| :--- | :--------- | :------- | :------------------------------ |
| id   | number\[\] | Yes      | Campaign IDs to get results for. |

##### Example Request

```shell
curl -u "api_user:token" -X GET 'http://localhost:9000/api/campaigns/analytics/engagement?id=1'
```

##### Example Response

```json
{
  "data": [
    {
      "campaign_id": 1,
      "bounced": 12,
      "delivered": 29968,
      "opened": 6120,
      "clicked": 1544,
      "anonymous": 0
    }
  ]
}
```

______________________________________________________________________

#### GET /api/campaigns/analytics/{type}

Retrieve stats of specified campaigns. `views` and `clicks` count people: each subscriber once per campaign, on the day of their first view or click in the range. `links` counts the people who clicked each link.

##### Parameters

| Name | Type       | Required | Description                                   |
| :--- | :--------- | :------- | :-------------------------------------------- |
| id   | number\[\] | Yes      | Campaign IDs to get stats for.                |
| type | string     | Yes      | Analytics type: views, links, clicks, bounces |
| from | string     | Yes      | Start of the date range: a date (`2024-08-04`) or an ISO 8601 timestamp. |
| to   | string     | Yes      | End of the date range. A date includes that whole day. |


##### Example Request

```shell
curl -u "api_user:token" -X GET 'http://localhost:9000/api/campaigns/analytics/views?id=1&from=2024-08-04&to=2024-08-12'
```

##### Example Response

```json
{
  "data": [
    {
      "campaign_id": 1,
      "count": 10,
      "timestamp": "2024-08-04T00:00:00Z"
    },
    {
      "campaign_id": 1,
      "count": 14,
      "timestamp": "2024-08-08T00:00:00Z"
    },
    {
      "campaign_id": 1,
      "count": 20,
      "timestamp": "2024-08-09T00:00:00Z"
    },
    {
      "campaign_id": 1,
      "count": 21,
      "timestamp": "2024-08-10T00:00:00Z"
    },
    {
      "campaign_id": 1,
      "count": 21,
      "timestamp": "2024-08-11T00:00:00Z"
    }
  ]
}
```

##### Example Request

```shell
curl -u "api_user:token" -X GET 'http://localhost:9000/api/campaigns/analytics/links?id=1&from=2024-08-04T18%3A30%3A00.624Z&to=2024-08-12T18%3A29%3A00.624Z'
```

##### Example Response

```json
{
  "data": [
    {
      "url": "https://freethebears.org",
      "count": 294
    },
    {
      "url": "https://calmcode.io",
      "count": 278
    },
    {
      "url": "https://climate.nasa.gov",
      "count": 261
    },
    {
      "url": "https://www.storybreathing.com",
      "count": 260
    }
  ]
}
```

______________________________________________________________________

#### POST /api/campaigns

Create a new campaign.

##### Parameters

| Name         | Type       | Required | Description                                                                                                            |
| :----------- | :--------- | :------- | :--------------------------------------------------------------------------------------------------------------------- |
| name         | string     | Yes      | Campaign name.                                                                                                         |
| subject      | string     | Yes      | Campaign email subject.                                                                                                |
| lists        | number\[\] | Yes      | List IDs to send campaign to.                                                                                          |
| from_email   | string     |          | 'From' email in campaign emails. Defaults to value from settings if not provided.                                      |
| type         | string     | Yes      | Campaign type: 'regular' or 'optin'.                                                                                   |
| content_type | string     | Yes      | Content type: 'richtext', 'html', 'markdown', 'plain', 'visual'.                                                       |
| body         | string     | Yes      | Content body of campaign.                                                                                              |
| body_source  | string     |          | If content_type is `visual`, the JSON block source of the body.                                                        |
| altbody      | string     |          | Alternate plain text body for HTML (and richtext) emails.                                                              |
| send_at      | string     |          | Timestamp to schedule campaign. Format: 'YYYY-MM-DDTHH:MM:SSZ'.                                                        |
| messenger    | string     |          | 'email' or a custom messenger defined in settings. Defaults to 'email' if not provided.                                |
| template_id  | number     |          | Template ID to use. Defaults to default template if not provided.                                                      |
| tags         | string\[\] |          | Tags to mark campaign.                                                                                                 |
| headers      | JSON       |          | Key-value pairs to send as SMTP headers. Supports template expressions (e.g., `{{ .Subscriber.UUID }}`). Example: \[{"x-custom-header": "value"}, {"x-subscriber": "{{ .Subscriber.UUID }}"}\]. |
| attribs      | JSON       |          | Optional JSON object attributes that can be used in the campaign message template. Example `{"location": "Somewhere"}` |
| tracking     | JSON       |          | Click tracking and tracking pixels, for HTML formats: `{"links": true, "pixels": ["https://px.example.com/p.gif"]}`. `links` tracks every web link; `pixels` are up to 10 http(s) image URLs. Omitted means both off. See [Tracking](../guides/tracking.md). |

##### Example request

```shell
curl -u "api_user:token" 'http://localhost:9000/api/campaigns' -X POST -H 'Content-Type: application/json;charset=utf-8' --data-raw '{"name":"Test campaign","subject":"Hello, world","lists":[1],"from_email":"listnun <noreply@listnun.yoursite.com>","content_type":"richtext","messenger":"email","type":"regular","tags":["test"],"template_id":1}'
```

##### Example response

```json
{
    "data": {
        "id": 1,
        "created_at": "2021-12-27T11:50:23.333485Z",
        "updated_at": "2021-12-27T11:50:23.333485Z",
        "views": 0,
        "clicks": 0,
        "bounces": 0,
        "lists": [{
            "id": 1,
            "name": "Default list"
        }],
        "started_at": null,
        "to_send": 1,
        "sent": 0,
        "uuid": "90c889cc-3728-4064-bbcb-5c1c446633b3",
        "type": "regular",
        "name": "Test campaign",
        "subject": "Hello, world",
        "from_email": "listnun \u003cnoreply@listnun.yoursite.com\u003e",
        "body": "",
        "body_source": null,
        "altbody": null,
        "send_at": null,
        "status": "draft",
        "content_type": "richtext",
        "tags": ["test"],
        "template_id": 1,
        "messenger": "email",
        "headers": {},
        "attribs": {}
    }
}
```

______________________________________________________________________

#### POST /api/campaigns/{campaign_id}/test

Test campaign with arbitrary subscribers.

Use the same parameters in [POST /api/campaigns](#post-apicampaigns) in addition to the below parameters.

##### Parameters

| Name        | Type       | Required | Description                                        |
| :---------- | :--------- | :------- | :------------------------------------------------- |
| subscribers | string\[\] | Yes      | List of subscriber e-mails to send the message to. |

______________________________________________________________________

#### PUT /api/campaigns/{campaign_id}

Update a campaign.

> Refer to parameters from [POST /api/campaigns](#post-apicampaigns)

______________________________________________________________________

#### PUT /api/campaigns/{campaign_id}

Update a specific campaign.

> Refer to parameters from [POST /api/campaigns](#post-apicampaigns)

______________________________________________________________________

#### PUT /api/campaigns/{campaign_id}/status

Change status of a campaign.

##### Parameters

| Name        | Type   | Required | Description                                                             |
| :---------- | :----- | :------- | :---------------------------------------------------------------------- |
| campaign_id | number | Yes      | Campaign ID to change status.                                           |
| status      | string | Yes      | New status for campaign: 'scheduled', 'running', 'paused', 'cancelled'. |

##### Note

> - Only 'scheduled' campaigns can change status to 'draft'.
> - Only 'draft' campaigns can change status to 'scheduled'.
> - Only 'paused' and 'draft' campaigns can start ('running' status).
> - Only 'running' campaigns can change status to 'cancelled' and 'paused'.

##### Example Request

```shell
curl -u "api_user:token" -X PUT 'http://localhost:9000/api/campaigns/1/status' \
--header 'Content-Type: application/json' \
--data-raw '{"status":"scheduled"}'
```

##### Example Response

```json
{
    "data": {
        "id": 1,
        "created_at": "2020-03-14T17:36:41.29451+01:00",
        "updated_at": "2020-04-08T19:35:17.331867+01:00",
        "views": 0,
        "clicks": 0,
        "lists": [
            {
                "id": 1,
                "name": "Default list"
            }
        ],
        "started_at": null,
        "to_send": 0,
        "sent": 0,
        "uuid": "57702beb-6fae-4355-a324-c2fd5b59a549",
        "type": "regular",
        "name": "Test campaign",
        "subject": "Welcome to listnun",
        "from_email": "No Reply <noreply@yoursite.com>",
        "body": "<h3>Hi {{ .Subscriber.FirstName }}!</h3>\n\t\t\tThis is a test e-mail campaign. Your second name is {{ .Subscriber.LastName }} and you are from {{ .Subscriber.Attribs.city }}.",
        "send_at": "2020-03-15T17:36:41.293233+01:00",
        "status": "scheduled",
        "content_type": "richtext",
        "tags": [
            "test-campaign"
        ],
        "template_id": 1,
        "messenger": "email"
    }
}
```

______________________________________________________________________

#### PUT /api/campaigns/{campaign_id}/archive

Publish campaign to public archive.

##### Parameters

| Name                | Type        | Required | Description                                                               |
| :------------------ | :---------- | :------- | :------------------------------------------------------------------------ |
| campaign_id         | number      | Yes      | Campaign ID to publish to public archive.                                 |
| archive             | bool        | Yes      | State of the public archive.                                              |
| archive_template_id | number      | No       | Archive template id. Defaults to 0.                                       |
| archive_meta        | JSON string | No       | Optional Metadata to use in campaign message or template.Eg: name, email. |
| archive_slug        | string      | No       | Name for page to be used in public archive URL                            |


##### Example Request

```shell

curl -u "api_user:token" -X PUT 'http://localhost:8080/api/campaigns/33/archive' 
--header 'Content-Type: application/json' 
--data-raw '{"archive":true,"archive_template_id":1,"archive_meta":{},"archive_slug":"my-newsletter-old-edition"}'
```

##### Example Response

```json
{
  "data": {
    "archive": true,
    "archive_template_id": 1,
    "archive_meta": {},
    "archive_slug": "my-newsletter-old-edition"
  }
}
```

______________________________________________________________________

#### DELETE /api/campaigns/{campaign_id}

Delete a campaign.

##### Parameters

| Name        | Type   | Required | Description            |
| :---------- | :----- | :------- | :--------------------- |
| campaign_id | number | Yes      | Campaign ID to delete. |

##### Example Request

```shell
curl -u "api_user:token" -X DELETE 'http://localhost:9000/api/campaigns/34'
```

##### Example Response

```json
{
    "data": true
}
```

______________________________________________________________________

#### DELETE /api/campaigns

Delete multiple campaigns by IDs or by a search query.

##### Parameters

| Name  | Type       | Required                      | Description                                                                 |
| :---- | :--------- | :---------------------------- | :-------------------------------------------------------------------------- |
| id    | number\[\] | Yes (if `query` not provided) | Onr or more campaign IDs to delete.                                         |
| query | string     | Yes (if `id` not provided)    | Fulltext search query to filter campaigns for deletion (same as GET query). |

##### Example Request (by IDs)

```shell
curl -u "api_user:token" -X DELETE 'http://localhost:9000/api/campaigns?id=10&id=11&id=12'
```

##### Example Request (by search query)

```shell
curl -u "api_user:token" -X DELETE 'http://localhost:9000/api/campaigns?query=test%20campaign'
```

##### Example Response

```json
{
    "data": true
}
```

______________________________________________________________________

#### GET /api/campaigns/{campaign_id}/inbox-check

Run the [inbox check](../guides/campaigns.md#inbox-check) on the campaign as saved. Returns every check in display order, passing or not. `id` names the check, `group` is `content`, `subject`, `sender` or `audience`, and `params` holds the figures the check's message uses, such as the email's size or the share of inactive subscribers. The checks never block sending.

##### Example Response

```json
{
  "data": [
    {"id": "plainText", "group": "content", "pass": true},
    {"id": "size", "group": "content", "pass": false, "params": {"kb": "131"}},
    {"id": "subjectLength", "group": "subject", "pass": true},
    {"id": "fromFreeMail", "group": "sender", "pass": true},
    {"id": "inactive", "group": "audience", "pass": true}
  ]
}
```

______________________________________________________________________

#### GET /api/campaigns/{campaign_id}/replies

Retrieve the replies to a campaign that has [reply tracking](../guides/campaigns.md#replies) on, newest first.

##### Parameters

| Name     | Type   | Required | Description                 |
| :------- | :----- | :------- | :-------------------------- |
| page     | number |          | Page number for pagination. |
| per_page | number |          | Results per page.           |

##### Example Response

```json
{
  "data": {
    "results": [
      {
        "id": 12,
        "uuid": "3f0e…",
        "campaign_id": 7,
        "subscriber_id": 1042,
        "subscriber_email": "ada@example.com",
        "subscriber_name": "Ada",
        "from_email": "ada@example.com",
        "from_name": "Ada",
        "subject": "Re: Spring sale",
        "text_body": "Do you ship to Bulawayo?\n\nOn Tue, … wrote: …",
        "html_body": "",
        "stripped_text": "Do you ship to Bulawayo?",
        "is_auto": false,
        "forwarded_to": "support@acme.com",
        "forwarded_at": "2026-10-20T09:15:03Z",
        "received_at": "2026-10-20T09:15:02Z"
      }
    ],
    "counts": {"replies": 31, "repliers": 28, "auto": 4},
    "total": 35,
    "page": 1,
    "per_page": 20
  }
}
```

`counts.replies` and `counts.repliers` leave out automatic replies (`is_auto`), which are counted in `counts.auto`. `stripped_text` is the reply without the quoted original, when the mail provider could separate it.

______________________________________________________________________

#### GET /api/campaigns/overview

The data behind the Campaigns page: every running and paused campaign, the next 10 scheduled with their audience, the 3 latest drafts and the 4 latest sent with bounces, openers and clickers, plus how many campaigns are in each status. Limited to the lists you can see.

##### Example Response

```json
{
  "data": {
    "campaigns": [
      {
        "id": 7, "uuid": "…", "name": "Spring sale", "status": "running",
        "lists": [{"id": 3, "name": "Newsletter"}],
        "audience": 5210, "to_send": 5210, "sent": 1800, "opened": 412, "clicked": 37, "bounced": 4,
        "send_at": null, "started_at": "2026-10-20T09:00:00Z", "pause_reason": "",
        "updated_at": "2026-10-20T08:55:00Z", "updated_by_name": "Ada"
      }
    ],
    "counts": [{"status": "draft", "count": 5}, {"status": "finished", "count": 41}]
  }
}
```

`updated_by_name` is who last created or saved the campaign.
