# Your team

Open **Workspace › Team** to manage who can use your workspace and what they can do. It has three tabs: **Members**, **Roles** and **List access**.

To connect another system to listnun, don't add a member. Create an [API key](../developers/api-keys.md) instead. API users made before API keys existed are listed, and can be migrated, under **Developers › API keys**.

## Members

The table shows each member's role, whether they use two-step verification, and when they were last active. Members who sign in with single sign-on show **SSO** under two-step, since their identity provider handles it.

**Add member** creates a teammate:

| Field | Notes |
| :---- | :---- |
| **Username** | What they sign in with, when using a password. |
| **Name** and **email** | Shown in the app and used for notifications. |
| **Enable password login** | Off means they sign in with single sign-on only (see [OIDC SSO](../oidc.md)). |
| **User role** | What they can do across the workspace. |
| **List role** | Optionally, which lists they can see and manage. |
| **Status** | **Deactivated** members can't sign in. |

To change someone's role, pick a new one from the picker on their row. Making someone a **Super Admin** asks you to confirm first. You can't change your own role, or a deactivated member's. Open a member to change anything else, or use their row's menu to **Deactivate** them.

Deactivated members are folded away under **N deactivated accounts** below the table. Searching still finds them.

If any active members sign in with just a password and no two-step verification, a banner says how many and links to **Require two-step for everyone**. See [Requiring it for everyone](#requiring-it-for-everyone).

## Roles

A user role is a set of permissions, such as "can send campaigns" or "can manage settings". The built-in **Super Admin** role can do everything. Create roles for the jobs people do, such as an *Editor* who writes campaigns but can't change settings. See [Roles and permissions](../roles-and-permissions.md) for the full list of permissions.

## List access

A list role limits a user to particular lists: for each list, whether they can see its subscribers and whether they can manage them. Give it to users who should only work with some of your lists. A user role with **lists:get_all** or **lists:manage_all** overrides it.

Deleting a list role doesn't delete the users who had it; they just lose its list access.

## Your profile and two-step verification

Click your initials at the top right and choose **Profile**.

- **Details**: change your name and email. If you sign in with single sign-on, your email comes from your provider and can't be changed here.
- **Password**: enter your current password, then the new one twice (at least 8 characters). Changing it signs you out everywhere else.
- **Two-factor authentication**: click **Turn on**, scan the QR code with an authenticator app (such as Ente Auth or Google Authenticator), and enter the 6-digit code it shows. From then on you'll be asked for a code each time you sign in. To turn it off, enter your password.

### Requiring it for everyone

An admin can turn on **Settings › Sign-in and security › Require two-step verification**. People who sign in with a password then have to set it up before they can use the workspace: they're taken to their profile with a note explaining why. Single sign-on users and API keys aren't affected.
