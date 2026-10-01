# Quarterdeck

[![Tests](https://github.com/rotationalio/quarterdeck/actions/workflows/tests.yaml/badge.svg)](https://github.com/rotationalio/quarterdeck/actions/workflows/tests.yaml)

**Distributed authentication and authorization service for Rotational applications.**

Quarterdeck is a JWT issuer that also provides JWKS to verify that keys have been issued from Quarterdeck. Quarterdeck provides middleware for other Go applications to easily authenticate JWT claims provided in requests and the claims themselves provide authorization scope for access. Quarterdeck also provides user and api key management via an API and a simple user interface.

Projects using Quarterdeck:

- [Endeavor](https://github.com/rotationalio/endeavor)
- [HonuDB](https://github.com/rotationalio/honu)

## CSRF Protection

Quarterdeck applies Gimlet's `csrf/secfetch` middleware globally to application routes (health probes remain outside application middleware). No client or frontend CSRF management or tokens need to be tracked or implemented, however CSRF errors will still be captured and displayed to the user as a toast. `QD_CSRF_COOKIE_DOMAIN`, `QD_CSRF_COOKIE_TTL`, and `QD_CSRF_SECRET` have been removed; authentication cookies are unchanged.

Verified bearer tokens are accepted only through Gimlet's `WithFallback` check, not as an unconditional bypass. The fallback runs only when `Sec-Fetch-Site`, `Origin`, and `Referer` are all absent and configured fetch mode/destination checks have passed. It verifies the token without cookie refresh; header presence alone is insufficient. Protected routes still perform their normal authentication and authorization.

Defaults allow `GET`, `HEAD`, and `OPTIONS`, and allow same-origin mutations. Same-site mutations require an exact trusted Origin. Missing or unknown Fetch Metadata requires a trusted Origin/Referer; requests with no site, origin, or referer can instead pass verified bearer fallback. Cross-site mutations are rejected regardless of bearer authentication, and `none` mutations are rejected by default. Bearer fallback cannot override other metadata or origin rejections. Proxies must forward Fetch Metadata headers unchanged; browser clients should use HTTPS (or localhost).

Configure comma-separated exact browser-facing origins with `QD_CSRF_EXPECTED_ORIGINS`, e.g. `https://app.example.com,https://auth.example.com`. This trust list is separate from `QD_ALLOW_ORIGINS` (CORS); cross-origin browser clients may need both configured.

| Environment variable                 | Default                                                      |
| ------------------------------------ | ------------------------------------------------------------ |
| `QD_CSRF_DISABLED`                   | `false`; `true` logs would-be rejections without blocking    |
| `QD_CSRF_NAMESPACE`                  | `quarterdeck` (`X-Quarterdeck-CSRF-Error`)                   |
| `QD_CSRF_SAFE_HTTP_METHODS`          | `GET,HEAD,OPTIONS` (only these safe methods can be exempted) |
| `QD_CSRF_EXPECTED_ORIGINS`           | Empty                                                        |
| `QD_CSRF_ALLOW_MISSING_METADATA`     | `false`                                                      |
| `QD_CSRF_ALLOW_UNKNOWN_SITE`         | `false`                                                      |
| `QD_CSRF_ALLOW_SITE_NONE`            | `false`                                                      |
| `QD_CSRF_ALLOWED_FETCH_MODES`        | Empty (unrestricted)                                         |
| `QD_CSRF_ALLOWED_FETCH_DESTINATIONS` | Empty (unrestricted)                                         |
| `QD_CSRF_REQUIRE_FETCH_MODE`         | `false`                                                      |
| `QD_CSRF_REQUIRE_FETCH_DESTINATION`  | `false`                                                      |

Leave compatibility relaxations disabled unless needed. Rejections return HTTP 403 and the namespaced CSRF error header. The UI shows a toast asking the user to fully reload the page and contact support if the failure persists; it does not retry automatically.

## Testing (Postgres)

Some `pkg/store/v2` tests require a Postgres database and will fail with
"postgres not configured" unless a database URL is provided.

Start a local Postgres container (same defaults used in tidal):

```bash
docker run -d --name quarterdeck-postgres -e POSTGRES_USER=rotational -e POSTGRES_PASSWORD=theeaglefliesatdawn -e POSTGRES_DB=postgres -p 5432:5432 postgres:18
```

Run store tests with `POSTGRES_DATABASE_URL` set:

```bash
export POSTGRES_DATABASE_URL="postgres://rotational:theeaglefliesatdawn@localhost:5432/postgres?sslmode=disable"
go test ./pkg/store/v2/...
```

or, without export:

```bash
POSTGRES_DATABASE_URL="postgres://rotational:theeaglefliesatdawn@localhost:5432/postgres?sslmode=disable" go test ./pkg/store/v2/...
```

Stop and remove the container when finished:

```bash
docker stop quarterdeck-postgres && docker rm quarterdeck-postgres
```

## License

This project is licensed under the BSD 3-Clause License. See [`LICENSE.txt`](./LICENSE.txt) for details. Please feel free to use Quarterdeck in your own projects and applications.

## About Rotational Labs

Quarterdeck is developed by [Rotational Labs](https://rotational.io), a team of engineers and scientists building AI infrastructure for serious work.
