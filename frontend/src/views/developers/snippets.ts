// Code snippets for the Developers quickstart: the same POST /api/tx call
// in each language, using the workspace's own URL, key and template. 'llm'
// is a prompt to paste into an AI coding assistant instead of code, for
// users who'd rather have it write the integration in their own stack.

export type SnippetLang = 'curl' | 'node' | 'python' | 'go' | 'llm';

export const SNIPPET_LANGS: SnippetLang[] = ['curl', 'node', 'python', 'go', 'llm'];

export interface SnippetInput {
  rootUrl: string;
  keyName: string;
  templateId: number;
  templateName: string;
  recipient: string;
  data: Record<string, unknown>;
}

function body(i: SnippetInput, indent: string): string {
  return JSON.stringify({ subscriber_email: i.recipient, template_id: i.templateId, data: i.data }, null, 2)
    .split('\n')
    .map((l, n) => (n === 0 ? l : indent + l))
    .join('\n');
}

function llmPrompt(i: SnippetInput, url: string): string {
  const vars = Object.keys(i.data);
  return [
    `Add transactional email sending to my app using the listnun API. ${i.templateName ? `The first email is "${i.templateName}".` : ''}`.trim(),
    '',
    '## Endpoint',
    `POST ${url}`,
    'Content-Type: application/json',
    '',
    '## Auth',
    `HTTP Basic auth. Username is the API key name "${i.keyName}", password is the API key secret.`,
    'Read the secret from the LISTNUN_API_KEY environment variable. Never hardcode it or send it to the browser:',
    'call this API from server-side code only.',
    '',
    '## Request body',
    body(i, ''),
    '',
    `- template_id: ${i.templateId} is the transactional template to render.`,
    vars.length
      ? `- data: values the template uses (${vars.join(', ')}). Fill them from my app's own data, not hardcoded samples.`
      : '- data: values the template uses. Fill them from my app\'s own data.',
    '- subscriber_email: the recipient. By default they must already be a subscriber in the workspace, or the',
    '  send fails with HTTP 400. To email anyone (e.g. order receipts to guests), add "subscriber_mode": "external";',
    '  "subscriber_mode": "fallback" uses the subscriber record when it exists and sends anyway when it doesn\'t.',
    '- Optional: "subject" overrides the template subject, "from_email" overrides the sender,',
    '  "headers" is a list of {"Header-Name": "value"} objects.',
    '',
    '## Responses',
    '- 200 {"data": true}: accepted for delivery.',
    '- 400 {"message": "..."}: bad request (unknown recipient, missing field, template error). Don\'t retry; log the message.',
    '- 401/403: wrong key name or secret, or the key lacks the tx:send scope.',
    '- 429 or 5xx: retry with exponential backoff, a few attempts at most.',
    '',
    '## What I want',
    '- A small, reusable sendTransactionalEmail(to, templateId, data) function in my project\'s language and HTTP',
    '  client, with a 10s timeout and the error handling above.',
    '- Call it where this email should go out, and document LISTNUN_API_KEY in my env example/config.',
    '- Ask me which language/framework to use if it isn\'t clear from the project.',
  ].join('\n');
}

export function snippet(lang: SnippetLang, i: SnippetInput): string {
  const url = `${i.rootUrl.replace(/\/$/, '')}/api/tx`;
  const comment = `Send ${i.templateName || 'a transactional email'}`;

  switch (lang) {
    case 'node':
      return [
        `// ${comment}`,
        'const auth = Buffer.from(',
        `  \`${i.keyName}:\${process.env.LISTNUN_API_KEY}\`,`,
        ').toString("base64");',
        '',
        `await fetch("${url}", {`,
        '  method: "POST",',
        '  headers: {',
        // eslint-disable-next-line no-template-curly-in-string -- the snippet's own template literal
        '    "Authorization": `Basic ${auth}`,',
        '    "Content-Type": "application/json",',
        '  },',
        `  body: JSON.stringify(${body(i, '  ')}),`,
        '});',
      ].join('\n');

    case 'python':
      return [
        `# ${comment}`,
        'import os, requests',
        '',
        'requests.post(',
        `    "${url}",`,
        `    auth=("${i.keyName}", os.environ["LISTNUN_API_KEY"]),`,
        `    json=${body(i, '    ')},`,
        '    timeout=10,',
        ').raise_for_status()',
      ].join('\n');

    case 'go':
      return [
        `// ${comment}`,
        `body := strings.NewReader(\`${body(i, '')}\`)`,
        `req, _ := http.NewRequest("POST", "${url}", body)`,
        `req.SetBasicAuth("${i.keyName}", os.Getenv("LISTNUN_API_KEY"))`,
        'req.Header.Set("Content-Type", "application/json")',
        'resp, err := http.DefaultClient.Do(req)',
      ].join('\n');

    case 'llm':
      return llmPrompt(i, url);

    default:
      return [
        `# ${comment}`,
        `curl -X POST ${url} \\`,
        `  -u "${i.keyName}:$LISTNUN_API_KEY" \\`,
        '  -H "Content-Type: application/json" \\',
        `  -d '${body(i, '  ').replace(/'/g, "'\\''")}'`,
      ].join('\n');
  }
}
