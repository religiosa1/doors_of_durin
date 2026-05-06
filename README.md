# Doors of Durin

A simple golang auth server, intended to be used with nginx [auth_request](https://nginx.org/en/docs/http/ngx_http_auth_request_module.html)
to cover with basic-like authorization some of the self-hosted services with a
small and manually controlled list of users -- basically your family members.

<img width="1780" height="1473" alt="localhost_4000_login" src="https://github.com/user-attachments/assets/fdf08379-c041-42b5-a1f6-473764da667e" />

_Speak, friend, and enter._

If you're looking for a full-featured, production scale one look at [Authelia](https://www.authelia.com/) or
[Authentik](https://goauthentik.io/).

Can also work behind [traefik](https://doc.traefik.io/traefik/reference/routing-configuration/http/middlewares/forwardauth/) 
or [caddy](https://caddyserver.com/docs/caddyfile/directives/forward_auth).

## Features

- lightweight, minimal dependency, single-binary distribution
- ULID-based session authentication
- session cookie with Secure and HttpOnly attributes on the cookies to prevent sniffing
- optional basic-auth support for API access
- small sleep on failed login attempts to make targeted users attacks more difficult
- rate-limiting by IP for the login endpoint against brute-force attacks
- CLI for user and session management
- full structured logging for everything
- CSRF protection on the login endpoint
- WAL-mode sqlite with hashed passwords -- once written, cannot be read or leaked

## How it works

Stores a list of users and their passwords in a local
sqlite database. Authorization is implemented as session headers, sessions
are stored in the same database.

The main endpoint is `GET /verify`. It's intended to be used as the destination
of nginx `auth_request` -- it receives headers from request to upstream and
returns either 200 or 401 status based on the presence and validity of the
session id cookie. On a 200 response it sets an `X-Auth-User` header containing
the authenticated username, which can be forwarded to the upstream application.

Please notice, that cookies are set with Secure and HttpOnly 
[attributes](https://developer.mozilla.org/en-US/docs/Web/HTTP/Guides/Cookies#security).
That means they will not be acessible on plain HTTP connection, only HTTPS.
It's there just to make sure, your session cookie won't be sniffed on a initial 
non HTTP connection if you have a redirect from HTTP to HTTPS.

On 401 responses, users should be redirected by nginx to the `/login` page,
which will allow user to enter their login/password and will set the session id
cookie. Supplying back_url query param will allow users to be redirected back
to the target page after a login (see [nginx config example](#nginx-configuration-example)).

`POST /logout` deletes the session from the database and clears the session
cookie. It accepts an optional `back_url` query parameter to redirect the user
after logout; if omitted, the user is redirected to `/`. Returns 401 if no
session cookie is present in the request.

No Web-UI for registration or password restore as it's assumed list of users
is small enough for manual management.

### BasicAuth

If config field `EnableBasicAuth` is set to true, `/verify` endpoint will also
allow basic-auth authentication. Make sure, your server is behind SSL in order
to use it, otherwise login-password pair can be trivially sniffed from the
request.

Failed basic auth requests will trigger a rate-limiting, which will prevent any
basic-auth or session-based logins from the IP-address (but it won't invalidate
current active sessions).

## CLI usage:

```
Commands:
  user add [<username>] [flags]
    Add a new user

  user delete [<username>] [flags]
    Delete a user

  user rename [<username>] [flags]
    Rename a user

  user list [flags]
    List all users

  session add [<username>] [flags]
    Create a session (login)

  session delete [flags]
    Delete sessions

  session list [flags]
    List all sessions

  serve [flags]
    Run HTTP server

Run "durin <command> --help" for more information on a command.
```

### To Do in future iterations:

User list web CRUD, currently managed only through the CLI.

Perform some basic monitoring and alerting if the traffic to the services
exceeds some predefined values.

## Configuration

Service can be configured with a yml config file or through env variables.
When resolving config file it picks the first one found in the following order:

- `${XDG_CONFIG_HOME}/durin/config.yml`
- `/etc/durin.yml`

Configuration file can also be specified either through the `DURIN_CONFIG_PATH`
env variable, or with `-c` cli flag.

For an example of configuration file please see [conf/config.example.yml](conf/config.example.yml)

On windows the following paths will be checked instead:

- `${APPDATA}\\durin\\config.yml`
- `${PROGRAMDATA}\\durin\\config.yml`

You can override `UserConfigPath` and `GlobalConfigPath` at build time with ldflags
if you want to modify this location, e.g.:

```sh
go build `-X 'github.com/religiosa1/doors_of_durin/internal/config.GlobalConfigPath=/something/else` .
```

## What to forward from HTTP:

```
Scheme Detection:
  Default: X-Forwarded-Proto (header)
  Fallback: TLS (listening socket state)

Path Detection:
  Default: X-Forwarded-URI (header)
  Fallback: Start Line Request Target (start line)

Host Detection:
  Default: X-Forwarded-Host (header)
  Fallback: Host (header)

Remote IP:
  Default: X-Real-IP (header)
  Fallback: X-Forwarded-For (first entry)
  Last resort: TCP source IP
```

## Nginx configuration example

```
  server {
      listen 80;

      # Forwarding headers sent to all upstream requests
      # this is required for correct identification in auth server
      proxy_set_header X-Forwarded-Proto $scheme;
      proxy_set_header X-Forwarded-Host $http_host;
      proxy_set_header X-Forwarded-URI $request_uri;
      proxy_set_header X-Real-IP $remote_addr;

      # Internal auth subrequest
      location = /auth-check {
          internal;
          proxy_pass http://auth:4000/verify;
          proxy_pass_request_body off;
          proxy_set_header Content-Length "";
      }

      # Redirect unauthenticated requests to the auth server login page
      location @login {
          # Scheme and host part are required if you're serving on a non-standard port
          return 302 $scheme://$http_host/auth/login?redirect_to=$request_uri;
      }

      # All auth server endpoints: login, logout, static assets
      # This must be supplied in auth server configuration, e.g. as an env:
      # URL_PREFIX: "/auth"
      location /auth/ {
          proxy_pass http://auth:4000/;
      }

      # Protected reverse-proxied backend app
      location /app/ {
          auth_request /auth-check;
          error_page 401 = @login;
          auth_request_set $auth_user $upstream_http_x_auth_user;
          proxy_set_header X-Auth-User $auth_user;
          proxy_pass http://backend/;
      }
  }
```

See a more detailed example of nginx configuration file in [conf/nginx.conf](./conf/nginx.conf)

## Working with the source-code locally.

This project uses [mise-en-place](https://mise.jdx.dev/) for tooling management and build tasks.

Once you have mise installed, run:

```sh
mise trust
mise install # only once, after the initial clone
```

For ease of development, there's a dev server with hot reload via
[air](https://github.com/air-verse/air)

```
mise run dev # to run with hot reload with air
```

## License

doors_of_durin is MIT licensed
