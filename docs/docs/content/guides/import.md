# Importing subscribers

**Subscribers › Import** adds many subscribers at once from a spreadsheet. It walks you through four steps, **Upload**, **Match columns**, **Choose options** and **Review**, and nothing is written until you click **Start import** on the last one.

## 1. Upload

Drop a `.csv` file on the upload area, or choose one from your computer. A `.zip` with one CSV inside also works. Files can be up to 50 MB, and they need a column of email addresses. **Download a sample CSV** shows the layout.

Column names can be anything: you match them up on the next step. Commas, semicolons, tabs and other separators are detected automatically.

```csv
Email,First name,City
ada@example.com,Ada,London
alan@example.com,Alan,
```

The uploaded file is kept for two hours while you finish the steps. If it expires, upload it again.

## 2. Match columns

Each column of your file is listed with its first few values and a guess at what it holds. For each, choose what to **Save as**:

| Save as | What happens |
| :------ | :----------- |
| **Email** | The subscriber's address. One column must be this. |
| **Name** | The subscriber's name. |
| **Attribute** | Saved to the subscriber's [attributes](../concepts.md#attributes) under the column's name, e.g. `city`. Any column you keep that isn't email or name is saved as an attribute. |
| **Attributes (JSON)** | A column that already holds a JSON object, such as `{"city": "Harare", "plan": "pro"}`. |
| **Don't import** | The column is ignored. |

## 3. Choose options

| Question | Choices |
| :------- | :------ |
| **What should happen to these people?** | **Add them as subscribers**, or **Block these addresses**: they never get email from this workspace and are unsubscribed from every list. |
| **Add them to** | The lists to add them to. |
| **Did they agree to hear from you?** | Only asked when a chosen list uses double opt-in. **Not sure, ask them**: they join once they click a confirmation email, which you send when the import is done. **Yes, they opted in**: they join straight away. Only pick this if you have their consent. |
| **People already here** | Shown when some addresses in the file are already subscribers. **Leave them as they are** only adds new people. **Update their details** replaces their names and attributes with the file's. |

## 4. Review and import

Before anything is written, the Review step checks the whole file and tells you how many new subscribers there will be, how many people are already here (and whether they'll be updated or left alone), how many will need confirming, and which rows will be skipped and why. Click **Start import** to go ahead.

The import runs on the server, so you can leave the page and come back. While it runs you see its progress and how long is left. When it finishes you see exactly how many people were added and how many were already here.

Rows that weren't imported are listed with their row number in your spreadsheet, the value and the problem:

- No email address
- Not a valid email address
- This email domain isn't allowed (see the domain lists in [Privacy and consent](settings.md#privacy-and-consent))
- Wrong number of columns

**Download these rows** saves them as a CSV, so you can fix them and import just those rows again.

If new people need to confirm a double opt-in list, nothing is sent automatically. The finished import offers **Send confirmation emails**, which sends them the confirmation email. People who were sent one, or added to a list, in the last 24 hours are skipped, so nobody gets two in a row.

Only one import runs at a time per workspace.

## Recent imports

Finished imports are listed under **Recent imports**, with how many people each added or blocked and how many rows it skipped.

## Tips

- Large files are fine. Rows are written in batches, and the first ones show up in your subscriber list within a second or two.
- Importing someone who's already on a list doesn't create a duplicate. An email address is one subscriber.
- If your workspace uses [email validation](settings.md#email-validation), imported addresses are checked in the background after the import.
