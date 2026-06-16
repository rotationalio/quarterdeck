# Quarterdeck

[![Tests](https://github.com/rotationalio/quarterdeck/actions/workflows/tests.yaml/badge.svg)](https://github.com/rotationalio/quarterdeck/actions/workflows/tests.yaml)

**Distributed authentication and authorization service for Rotational applications.**

Quarterdeck is a JWT issuer that also provides JWKS to verify that keys have been issued from Quarterdeck. Quarterdeck provides middleware for other Go applications to easily authenticate JWT claims provided in requests and the claims themselves provide authorization scope for access. Quarterdeck also provides user and api key management via an API and a simple user interface.

Projects using Quarterdeck:

- [Endeavor](https://github.com/rotationalio/endeavor)
- [HonuDB](https://github.com/rotationalio/honu)

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
