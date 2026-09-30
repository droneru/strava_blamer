# strava_blamer
CLI tool (Go) that renames default runs and swims on Strava based on workout content

> **Status: on hold.** Strava has made API access subscriber-only, and the tool cannot work without the API.

## Prerequisites

- **Paid Strava subscription.** Strava API access is subscriber-only. Without a subscription, every API call
  fails with `403 Forbidden (Application Status: Inactive)`, even though OAuth token refresh still succeeds.
- Go (see `go.mod` for the version).

## Build

```sh
make build   # bin/strava_blamer
make test
```

## Strava API access

The tool needs a Strava API application and a refresh token with the
`activity:read_all` and `activity:write` scopes. This is a one-time manual setup.

### 1. Create an API application

1. Open <https://www.strava.com/settings/api> and create an application.
2. Set **Authorization Callback Domain** to `localhost`. Other fields can be anything.
3. Note the **Client ID** and **Client Secret**.

### 2. Authorize the application

Open this URL in a browser, replacing `CLIENT_ID`:

```
https://www.strava.com/oauth/authorize?client_id=CLIENT_ID&response_type=code&redirect_uri=http://localhost/exchange_token&approval_prompt=force&scope=activity:read_all,activity:write
```

Click **Authorize** and leave both activity checkboxes enabled. The browser is redirected to
a page that does not load, for example:

```
http://localhost/exchange_token?state=&code=AUTH_CODE&scope=read,activity:write,activity:read_all
```

Check that `scope` contains both `activity:write` and `activity:read_all`, then copy the
`code` value. The code is single-use and expires quickly, so do the next step right away.

### 3. Exchange the code for a refresh token

```sh
curl -X POST https://www.strava.com/oauth/token \
  -F client_id=CLIENT_ID \
  -F client_secret=CLIENT_SECRET \
  -F code=AUTH_CODE \
  -F grant_type=authorization_code
```

Copy `refresh_token` from the JSON response.

### 4. Configure the tool

```sh
cp .env.example .env   # fill in the values
set -a; source .env; set +a
bin/strava_blamer get https://www.strava.com/activities/1234567890
```

`.env` is ignored by git and is also used by `docker run --env-file`.

On the first run the refresh token is copied into the token file (`--token-file`,
default `token.json`; `/data/token.json` in Docker). Strava issues a new refresh token
on every refresh and immediately invalidates the old one, so from then on the token file
is the only source of truth and `STRAVA_REFRESH_TOKEN` is ignored while the file exists.

If the token file is lost or the token stops working, repeat steps 2–3, delete the token
file and run again with the new `STRAVA_REFRESH_TOKEN`.

## Commands

| Command | Description |
|---|---|
| `get <id\|URL>` | Show activity info: current name, `sport_type`, distance, start time, laps for swims. Changes nothing |
