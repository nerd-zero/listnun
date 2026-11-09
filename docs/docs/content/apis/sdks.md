# SDKs and client libraries

listnun doesn't have official SDKs yet. For transactional email, the [Developers quickstart](../developers/index.md#quickstart) gives ready-to-paste code for curl, Node, Python and Go, filled in with your workspace's details, or a prompt for an AI assistant to write the integration in your own stack.

## Community listmonk libraries

listnun is forked from [listmonk](https://github.com/knadh/listmonk), and the community libraries below were written for listmonk's API. Much of it is the same, so they may work for subscribers, lists and campaigns, but they haven't been tested against listnun and aren't supported. Before using one, check that:

- it lets you set the base URL to your workspace's own address (`https://yourcompany.listnun.app`); each workspace has its own, and a key only works on its own workspace's address,
- it authenticates with HTTP Basic auth or the `Authorization: token name:secret` header, so it can use an [API key](../developers/api-keys.md),
- the endpoints you need are covered by your key's scopes (see [Permissions](apis.md#permissions)).

Features that only exist in listnun, such as API keys, webhooks, the message log, segments and signup forms, aren't in these libraries.


- [listmonk-ruby](https://github.com/polydice/listmonk-ruby) — Ruby API client
- [WordPress - WooCommerce plugin](https://github.com/post-duif/integration-listmonk-wordpress-plugin) integration for listmonk
- [listmonk](https://github.com/mikeckennedy/listmonk) — Python API client
- [listmonk-api](https://github.com/Knuckles-Team/listmonk-api) — Python API client
- [frappe_listmonk](https://github.com/anandology/frappe_listmonk) — Frappe framework integration for listmonk
- [auto-newsletter-listmonk](https://github.com/chaddyc/auto-newsletter-listmonk) — Ghost CMS integration
- [listmonk-newsletter](https://github.com/iloveitaly/listmonk-newsletter) - RSS to listmonk integration for email newsletters
- [listmonk-crystal](https://github.com/russ/listmonk-crystal) — Crystal lang API client
- [terraform-provider-listmonk](https://github.com/Muravlev/terraform-provider-listmonk) — Manage listmonk templates in Terraform
- [listmonk-php-client](https://github.com/arunnabraham/listmonk-php-client) — PHP API client
- [php-listmonk](https://github.com/junisan/php-listmonk) — PHP API client
- [go-listmonk](https://github.com/EzeXchange-API/go-listmonk) — Go API client
- [listmonk-nodejs-api](https://github.com/mihairaulea/listmonk-nodejs-api) — NodeJS API client
- [@maloma/listmonk](https://codeberg.org/maloma/listmonk-sdk) — NodeJS/Bun Typescript SDK
- [listmonk-laravel](https://github.com/theafolayan/listmonk-laravel) — Laravel API Client
- [nuxt-listmonk](https://github.com/roncallyt/nuxt-listmonk) — listmonk module for Nuxt.js
- [listmonk-japi](https://codeberg.org/hlassiege/listmonk-japi) - listmonk client for Java/Kotlin
- [listmonk-mcp](https://github.com/rhnvrm/listmonk-mcp) — MCP (Model Context Protocol) server for Claude integration
- [N8N Nodes](https://github.com/wiesinghilker/n8n-nodes-listmonk) — Adds listmonk nodes for N8N Automations
