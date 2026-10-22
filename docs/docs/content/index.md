# listnun documentation

listnun is a newsletter and mailing list platform. You keep your subscribers in lists, write campaigns, send them, and see who opened and clicked. Developers can also send one-off transactional emails, such as receipts and password resets, through an API.

These docs are in three parts. Start with the one that fits you.

<div class="grid cards" markdown>

-   **Using listnun**

    ---

    For anyone working in a workspace: building lists, importing subscribers, writing and sending campaigns, and reading the results.

    [Quickstart: send your first campaign](start/quickstart.md)

-   **Developers**

    ---

    For connecting your own systems: API keys, the REST API, transactional email, webhooks and template syntax.

    [Developers overview](developers/index.md)

-   **Running listnun**

    ---

    For people hosting listnun: installation, configuration, upgrades, workspaces and storage.

    [Installation](installation.md)

</div>

## Words you'll see

| Term | Meaning |
| :--- | :------ |
| **Workspace** | Your own space in listnun, with its own subscribers, campaigns, team and settings. It has its own address, like `yourcompany.listnun.app`. |
| **List** | A group of subscribers you send to. A subscriber can be on many lists. |
| **Subscriber** | One person: an email address, a name, and any extra details you keep about them (attributes). |
| **Campaign** | An email you send to one or more lists. |
| **Template** | The layout a campaign sits in: header, footer and styling. |
| **Transactional email** | A single email sent to one person when something happens, such as an order receipt. Sent through the API, not as a campaign. |

See [Concepts](concepts.md) for the details behind each of these.

listnun is built on [listmonk](https://listmonk.app), the open-source newsletter manager, and keeps its templating language and most of its API. The source is on [GitHub](https://github.com/nerd-zero/listnun); see [Developer setup](developer-setup.md) to work on it.
