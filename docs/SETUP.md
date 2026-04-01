# Setting Up sheets-mcp

This guide walks you through the complete setup of `sheets-mcp`, from creating a
Google Cloud project to your first conversation with your agent of choice. Total
time: about 15 minutes.

---

## Prerequisites

- Go installed (current stable version)
- `sheets-mcp` binary built and available (via `make install` or `make build`)
- A Google account (personal Gmail is fine — Google Workspace is not required)
- Claude Desktop installed and working

---

## Step 1: Create a Google Cloud Project

1. Go to the [Google Cloud Console](https://console.cloud.google.com/).
2. Sign in with the Google account whose spreadsheets you want to access.
3. Click the project dropdown at the top of the page (it may say "My First
   Project" or another existing project name).
4. Click **New project** (in the top-right portion of the dialog box).
5. Enter a project name — `sheets-mcp` works fine. Leave the organization and
   location at their defaults.
6. Click **Create**.
7. Make sure the new project is selected in the project dropdown before
   continuing.

---

## Step 2: Enable the Required APIs

You need two APIs enabled: Google Sheets and Google Drive.

### Enable the Google Sheets API

1. Go to the [API Library](https://console.cloud.google.com/apis/library).
2. Search for **Google Sheets API**.
3. Click it, then click **Enable**.

### Enable the Google Drive API

1. Still in the API Library, search for **Google Drive API**.
2. Click it, then click **Enable**.

---

## Step 3: Configure the OAuth Consent Screen

Before creating credentials, Google requires you to set up a consent screen —
this is what you'll see in the browser when you authorize `sheets-mcp`.

1. In the left sidebar/navigation menu, click **View all products**.
2. Navigate to **Google Auth Platform** → **Branding**. If you see a message
   saying "Google Auth Platform not configured yet", click **Get started**.
3. **App name**: Enter `sheets-mcp`.
4. **User support email**: Select your email address from the dropdown.
5. Click **Next**.
6. **Audience**: Select **External**. (This is the correct choice for a personal
   Google account. "Internal" is only available for Google Workspace
   organizations.)
7. Click **Next**.
8. **Contact information**: Enter your email address.
9. Click **Next**.
10. **Finish**: Check the "I agree" box for the Google API Services User Data
    Policy.
11. Click **Continue**.
12. Click **Create**.

### Add your email as a test user

Apps in "Testing" mode only work for users explicitly listed as testers. You
need to add yourself.

1. Navigate to **Google Auth Platform** → **Audience**.
2. Under **Test users**, click **Add users**.
3. Enter your Google account email address.
4. Click **Save**.

### Add the required scopes

1. Navigate to **Google Auth Platform** → **Data Access**.
2. Click **Add or remove scopes**.
3. In the filter/search box, search for and select these two scopes:
   - `https://www.googleapis.com/auth/spreadsheets` — allows reading and writing
     spreadsheet data
   - `https://www.googleapis.com/auth/drive.metadata.readonly` — allows
     searching for spreadsheets by name (metadata only, no file content access)
4. Click **Update** to save.

These scopes control what the consent screen shows the user. The
`drive.metadata.readonly` scope lets `sheets-mcp` find your spreadsheets by
name — it cannot read or modify file contents through Drive.

---

## Step 4: Create an OAuth Client ID

1. Navigate to **Google Auth Platform** → **Clients**.
2. Click **Create client**.
3. **Application type**: Select **Desktop app**.
4. **Name**: Enter `sheets-mcp` (this name is just for your reference in the
   console).
5. Click **Create**.

### Copy your credentials immediately

After creation, Google will show you the **Client ID** and **Client Secret**.

**Important:** As of mid-2025, Google only shows the client secret at the time
of creation. Once you close this dialog, the secret is hashed and cannot be
retrieved. Copy both values now or download the JSON file.

You need two values:
- **Client ID** — looks like `123456789-abcdefg.apps.googleusercontent.com`
- **Client secret** — looks like `GOCSPX-...`

Copy them somewhere safe. You'll paste them into the config file in the next
step.

---

## Step 5: Create the sheets-mcp Config File

Create the config directory and file:

```bash
mkdir -p ~/.config/sheets-mcp
```

Create `~/.config/sheets-mcp/config.json` with your credentials:

```json
{
  "client_id": "YOUR_CLIENT_ID_HERE",
  "client_secret": "YOUR_CLIENT_SECRET_HERE"
}
```

Replace the placeholder values with the Client ID and Client Secret you copied
in Step 4.

### Optional config fields

You can also add these optional fields:

```json
{
  "client_id": "YOUR_CLIENT_ID_HERE",
  "client_secret": "YOUR_CLIENT_SECRET_HERE",
  "default_spreadsheet": "YOUR_SPREADSHEET_ID",
  "allowed_spreadsheets": []
}
```

- **`default_spreadsheet`**: A spreadsheet ID that's used when Claude doesn't
  specify one. Find it in the spreadsheet URL:
  `https://docs.google.com/spreadsheets/d/THIS_PART_IS_THE_ID/edit`
- **`allowed_spreadsheets`**: An array of spreadsheet IDs to restrict access.
  When empty or omitted, all spreadsheets are accessible.

---

## Step 6: Authenticate

Run the OAuth flow:

```bash
sheets-mcp auth
```

This will:
1. Open your default browser to Google's consent screen.
2. You'll see a warning that "Google hasn't verified this app" — this is
   expected for personal projects. Click **Continue**.
3. Grant the requested permissions (spreadsheet access and Drive metadata).
4. You'll see "Authentication successful! You can close this tab." in the
   browser.
5. Back in your terminal, you should see a confirmation with the token expiry
   time.

Verify it worked:

```bash
sheets-mcp auth --status
```

You should see output like:

```
Authentication successful! Token expires 2026-03-29 18:27:15.
```

### Token expiry note

Because your app is in "Testing" mode (not verified by Google), refresh tokens
expire after 7 days. If you go more than a week without using `sheets-mcp`,
you'll need to re-run `sheets-mcp auth`. This is a Google-imposed limitation for
unverified apps. During normal use, the token refreshes automatically and you
won't notice.

---

## Step 7: Configure Claude Desktop

Add the `sheets-mcp` server to your Claude Desktop config.

Edit your Claude Desktop config file. On Linux, this is typically at
`~/.config/Claude/claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "sheets": {
      "command": "/home/YOUR_USERNAME/go/bin/sheets-mcp",
      "args": ["serve"]
    }
  }
}
```

Replace `/home/YOUR_USERNAME/go/bin/sheets-mcp` with the actual path to your
`sheets-mcp` binary. If you used `make install`, it's in `$GOBIN` or
`$GOPATH/bin` (usually `~/go/bin/`). If you used `make build`, it's at
`./bin/sheets-mcp` in the repo.

If you already have other MCP servers configured, add the `sheets` entry
alongside them:

```json
{
  "mcpServers": {
    "xmind": {
      "command": "/home/YOUR_USERNAME/go/bin/xmind-mcp"
    },
    "sheets": {
      "command": "/home/YOUR_USERNAME/go/bin/sheets-mcp",
      "args": ["serve"]
    }
  }
}
```

Restart Claude Desktop for the changes to take effect.

---

## Step 8: Test It

Open a conversation in Claude Desktop and try:

> "Find my spreadsheets"

Claude should call the `sheets_find` tool and list your Google Sheets. Then try:

> "Read the first sheet in [spreadsheet name]"

If everything is working, you'll see your spreadsheet data in the conversation.

---

## Troubleshooting

### "Missing client_id and client_secret in config"

The config file at `~/.config/sheets-mcp/config.json` is missing or doesn't have
both `client_id` and `client_secret`. Double-check the file exists and the JSON
is valid.

### "No stored token found"

You haven't run `sheets-mcp auth` yet, or the token file was deleted. Run
`sheets-mcp auth` to authenticate.

### "Authentication timed out"

You have 2 minutes to complete the consent flow in your browser. If the browser
didn't open automatically, check the terminal output for the consent URL and
open it manually.

### "Google hasn't verified this app" warning

This is expected. Your project is in "Testing" mode, which is fine for personal
use. Click **Continue** to proceed.

### "Access blocked: This app's request is invalid" or redirect URI errors

This usually means the OAuth client type is wrong. Make sure you created a
**Desktop app** client, not a "Web application" client. Desktop app clients
don't require redirect URIs to be configured in the console — `sheets-mcp`
handles this automatically.

### Token expired after 7 days of inactivity

Apps in Testing mode have refresh tokens that expire after 7 days. Run
`sheets-mcp auth` again to re-authenticate. During active use, tokens refresh
automatically.

### "insufficient_scope" or permission errors

Make sure both the Google Sheets API and Google Drive API are enabled in your
project (Step 2), and that the required scopes are added on the Data Access page
(Step 3).

### Claude Desktop doesn't show sheets tools

Make sure you restarted Claude Desktop after editing the config. Verify the
binary path in `claude_desktop_config.json` is correct and the binary has
execute permissions.

---

## Revoking Access

To revoke `sheets-mcp`'s access to your Google account and delete the stored
token:

```bash
sheets-mcp auth --revoke
```

This revokes the token with Google and deletes
`~/.config/sheets-mcp/token.json`.

To fully clean up, you can also delete the OAuth client in the
[Google Cloud Console](https://console.cloud.google.com/) under
**Google Auth platform** → **Clients**.
