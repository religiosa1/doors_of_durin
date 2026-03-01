# Simple auth server

A simple golang auth server, intended to be used with nginx [auth_request](https://nginx.org/en/docs/http/ngx_http_auth_request_module.html)
to cover with basic-like authorization some of the self-hosted services with a
small and manually controlled list of users -- basically your family members.

If you're looking for a full-featured, production scale one look at [Authelia](https://www.authelia.com/).

## How it works

Stores a list of users and their passwords in a local
sqlite database. Authorization is implemented as session headers, sessions
are stored in the same database.

The main endpoint is `/verify`. It's intended to be used as the destination of
nginx `auth_request` -- it receives headers from request to upstream and
returns either 200 or 401 status based on the presence and validity of the
session id cookie. On a 200 response it sets an `X-Auth-User` header containing
the authenticated username, which can be forwarded to the upstream application.

On 401 responses, users may be redirected in nginx to the `/login` page,
which will allow user to enter their login/password and will set the session id
cookie.

`POST /logout` deletes the session from the database and clears the session
cookie. It accepts an optional `back_url` query parameter to redirect the user
after logout; if omitted, the user is redirected to `/`. Returns 401 if no
session cookie is present in the request.

`/users` series of pages provides basic CRUD for admins, to manage users roles
and permissions.

No Web-UI for registration or password restore as it's assumed list of users
is small enough for manual management.

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

Run "auth_server <command> --help" for more information on a command.
```

### To Do in future iterations:

Perform some basic monitoring and alerting if the traffic to the services
exceeds some predefined values.

## What to forward:

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
  Default: X-Forwarded-For
  Fallback: TCP source IP
```

## Nginx configuration example

```
  server {
      listen 80;

      # Forwarding headers sent to all upstream requests
      # this is required for correct identification in auth server
      proxy_set_header X-Forwarded-URI $request_uri;
      proxy_set_header X-Forwarded-Proto $scheme;
      proxy_set_header X-Forwarded-Host $http_host;
      proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;

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

See the example nginx configuration file in [nginx/nginx.conf](./nginx/nginx.conf)

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

auth_server is MIT licensed
