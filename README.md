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
returns either 201 or 401 status based on the presence and validity of the
session id cookie.

On 401 responses, users may be redirected in nginx to the `/login` page,
which will allow user to enter their login/password and will set the session id
cookie.

`/users` series of pages provides basic CRUD for admins, to manage users roles
and permissions.

No Web-UI for registration or password restore as it's assumed list of users
is small enough for manual management.

### To Do in future iterations:

Perform some basic monitoring and alerting if the traffic to the services
exceeds some predefined values.

## What to forward:

```
Scheme Detection:
  Default: X-Forwarded-Proto (header)
  Fallback: TLS (listening socket state)

Host Detection:
  Default: X-Forwarded-Host (header)
  Fallback: Host (header)

Path Detection:
  Default: X-Forwarded-URI (header)
  Fallback: Start Line Request Target (start line)

Remote IP:
  Default: X-Forwarded-For
  Fallback: TCP source IP
```

## Nginx configuration example

```
server {
    listen 80;
    server_name example.com;

    proxy_set_header X-Original-URL $scheme://$http_host$request_uri;
    proxy_set_header X-Forwarded-URI $request_uri;
    proxy_set_header X-Forwarded-Proto $scheme;
    proxy_set_header X-Original-Method $request_method;
    proxy_set_header X-Forwarded-Host $http_host;
    proxy_set_header X-Forwarded-Ssl on;
    proxy_set_header X-Forwarded-For $remote_addr;
    proxy_set_header X-Real-IP $remote_addr;

    # The main app — protected by auth_request
    location / {
        auth_request /auth;
        auth_request_set $auth_status $upstream_status;

        # On auth failure, show the login page
        error_page 401 = @login;

        proxy_pass http://some-backend-app:3000;
    }

    # Internal subrequest to auth service
    location = /auth {
        internal;
        proxy_pass http://auth-service:4000/verify;

        proxy_pass_request_body off;
        proxy_set_header Content-Length "";
    }

    # Named location: shows the login form on 401
    location @login {
        proxy_pass http://auth-service:4000/login;
    }

    # Direct access to auth endpoints (login form submission, logout, etc.)
    location /auth/ {
        proxy_pass http://auth-service:4000/;
    }
}
```

## Working with the source-code locally.

This project uses [mise-en-place](https://mise.jdx.dev/) for tooling  management and build tasks.

Once you have mise installed, run:


```sh
mise install # only once, after the initial clone
```

For ease of development, there's a dev server with hot reload via 
[air](https://github.com/air-verse/air)

```
mise run dev # to run with hot reload with air
```

## License

auth_server is MIT licensed
