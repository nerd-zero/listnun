// Code snippets for the Developers quickstart: the same POST /api/tx call
// in each language, using the workspace's own URL, key and template.

export type SnippetLang = 'curl' | 'node' | 'python' | 'go';

export const SNIPPET_LANGS: SnippetLang[] = ['curl', 'node', 'python', 'go'];

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
