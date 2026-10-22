# Importing subscribers

**Subscribers › Import** adds many subscribers at once from a CSV file. It takes three steps.

## 1. Upload file

Drop a `.csv` file on the upload area, or click to choose one. You can also upload a `.zip` containing a CSV.

The first row must be the column names:

| Column | Required | Notes |
| :----- | :------- | :---- |
| `email` | Yes | One address per row. |
| `name` | No | The subscriber's name. |
| `attributes` | No | Extra data as a JSON object, e.g. `{"city": "Harare", "plan": "pro"}`. |

```csv
email,name,attributes
ada@example.com,Ada Lovelace,"{""city"": ""London""}"
alan@example.com,Alan Turing,
```

Commas are the default separator. If your file uses another one, such as `;`, set it on the next step.

## 2. Options

| Option | What it does |
| :----- | :----------- |
| **Mode** | **Subscribe** adds the people to lists. **Blocklist** blocklists every address in the file instead, so they never get email. |
| **Lists** | The lists to add subscribers to (subscribe mode). |
| **Subscription status** | **Unconfirmed** (they'll need to confirm, on double opt-in lists) or **Confirmed** (you already have their consent). |
| **Overwrite user info** | When an email already exists, replace their name and attributes with the file's. Off keeps what you already have. |
| **Overwrite subscription status** | When someone is already on a chosen list, set their subscription to the status above. This can resubscribe people who unsubscribed, so listnun asks you to confirm. |
| **CSV delimiter** | The column separator, if it isn't a comma. |

## 3. Import

The import runs on the server, so you can leave the page and come back. While it runs you see a progress bar, the number written so far, and a live log. When it finishes you see:

- **Imported**: subscribers added or updated.
- **Skipped**: rows left out because they were malformed, such as a bad email address or the wrong number of columns. Each one is in the log with the reason.
- **Errors**: valid rows the database refused.

Click **Download log** to keep a copy. **Done** clears the import so you can start another.

Only one import runs at a time per workspace.

## Tips

- Large files are fine. Rows are written in batches, and the first ones show up in your subscriber list within a second or two.
- Importing someone who's already on a list doesn't create a duplicate. An email address is one subscriber.
- If your workspace uses [Mail Validation](settings.md#mail-validation), imported addresses are checked in the background after the import.
