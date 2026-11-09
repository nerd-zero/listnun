# Quickstart: send your first campaign

This walks you from an empty workspace to a sent campaign in about ten minutes. You'll need a CSV file of subscribers, or you can add a few by hand.

## 1. Create a list

1. In the sidebar, open **Lists › All lists** and click **New list**.
2. Give it a name, for example *Newsletter*.
3. Choose who can see it:
    - **Public** lists appear on your subscription forms, so people can join them.
    - **Private** lists are only for you; people can't sign up to them.
4. Choose the opt-in:
    - **Single opt-in** adds people straight away.
    - **Double opt-in** emails each new subscriber a confirmation link first, and campaigns only go to people who confirmed.
5. Click **Save**.

## 2. Add subscribers

To bring in many at once, open **Subscribers › Import**:

1. Drop your CSV file on the upload area (a ZIP containing a CSV also works).
2. Under **Options**, choose **Subscribe**, pick your list, and set the subscription status.
3. Click **Import**. A progress bar shows the import as it runs, then how many were imported, skipped and failed.

Your CSV needs an `email` column. A `name` column and an `attributes` column (JSON) are optional. See [Importing subscribers](../guides/import.md) for the full format.

To add one person, open **Subscribers › All subscribers** and click **New subscriber**.

## 3. Write the campaign

Open **Campaigns › Create new**. The editor has four steps along the top.

1. **Settings**: give the campaign a name (only you see it), the subject line, the From address, and the lists to send to. Click **Continue**.
2. **Content**: pick a format and write the email. **Visual** lets you build it from blocks; **Rich text** is a normal word-processor editor. **Track link clicks** is on by default, so you'll see who clicks each link. See [Writing campaigns](../guides/campaigns.md#content).
3. **Schedule**: send as soon as you start it, or pick **Schedule for later** and a date and time.
4. **Review**: a checklist of everything that matters before sending.

## 4. Send yourself a test

On the **Review** step, the **Test email** check stays amber until you send a test of the current version. Click **Send test**, enter your own email address, and check the email in your inbox: the layout, the links and how it looks on your phone.

If you change the subject or content after testing, the check turns amber again so you test the new version.

## 5. Start it

Click **Start campaign** (or **Schedule** if you chose a time). listnun confirms how many subscribers it goes to, then starts sending. You can pause a running campaign from the campaigns list.

## 6. See the results

Open **Campaigns › Analytics**, or the campaign in **All campaigns**, to see opens, clicks, bounces and which links people clicked.

## Next

- [A tour of your workspace](workspace.md)
- [Lists and forms](../guides/lists-and-forms.md): put a signup form on your website
- [Tracking opens and clicks](../guides/tracking.md)
