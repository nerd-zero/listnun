# Your team

Open **Workspace › Users** to manage who can use your workspace and what they can do. It has four tabs: **Users**, **User roles**, **List roles** and **API users**. The last only lists API users made before API keys existed; new integrations use [API keys](../developers/api-keys.md).

## Users

**Add user** creates a teammate:

| Field | Notes |
| :---- | :---- |
| **Username** | What they sign in with, when using a password. |
| **Name** and **email** | Shown in the app and used for notifications. |
| **Enable password login** | Off means they sign in with single sign-on only (see [OIDC SSO](../oidc.md)). |
| **User role** | What they can do across the workspace. |
| **List role** | Optionally, which lists they can see and manage. |
| **Status** | **Disabled** users can't sign in. |

Open a user to change any of this later.

To connect another system to listnun, don't create a user. Create an [API key](../developers/api-keys.md) instead.

## User roles

A user role is a set of permissions, such as "can send campaigns" or "can manage settings". The built-in **Super Admin** role can do everything. Create roles for the jobs people do, such as an *Editor* who writes campaigns but can't change settings. See [Roles and permissions](../roles-and-permissions.md) for the full list of permissions.

## List roles

A list role limits a user to particular lists: for each list, whether they can see its subscribers and whether they can manage them. Give it to users who should only work with some of your lists. A user role with **lists:get_all** or **lists:manage_all** overrides it.

Deleting a list role doesn't delete the users who had it; they just lose its list access.

## Your profile and two-step verification

Click your initials at the top right and choose **Profile**.

- **Details**: change your name and email. If you sign in with single sign-on, your email comes from your provider and can't be changed here.
- **Password**: enter your current password, then the new one twice (at least 8 characters). Changing it signs you out everywhere else.
- **Two-factor authentication**: click **Turn on**, scan the QR code with an authenticator app (such as Ente Auth or Google Authenticator), and enter the 6-digit code it shows. From then on you'll be asked for a code each time you sign in. To turn it off, enter your password.

### Requiring it for everyone

An admin can turn on **Settings › Security › Require two-step verification**. People who sign in with a password then have to set it up before they can use the workspace: they're taken to their profile with a note explaining why. Single sign-on users and API keys aren't affected.
