# Store v2 test fixtures

Shared seed data for `pkg/store/v2` integration and conformance tests. Loaded by
[`suitetest.LoadFixtures`](../suite.go) at the start of each test from
`testdata/<provider>/` (`postgres/` or `sqlite3/`).

**All store v2 test seed data should live here.** Add new `.sql` files with a
numeric prefix so load order is explicit (e.g. `0005_my_table.sql`).

## Load order

| File | Tables |
|------|--------|
| `0001_permissions.sql` | `roles`, `permissions`, `role_permissions` |
| `0002_users.sql` | `users`, `user_roles`, `vero_tokens` |
| `0003_apikeys.sql` | `api_keys`, `api_key_permissions` |
| `0004_oidc_clients.sql` | `oidc_clients` |

Postgres and SQLite carry the same logical data; syntax differs (`decode(…, 'hex')`
vs `x'…'` for binary IDs).

## Roles (integer IDs)

| ID | Title | Notes |
|----|-------|-------|
| 1 | `admin` | All permissions |
| 2 | `editor` | Default role (`is_default=true`) |
| 3 | `viewer` | Read-only |
| 4 | `keyholder` | API key management |

## Permissions (integer IDs)

| ID | Title |
|----|-------|
| 1 | `content:modify` |
| 2 | `content:view` |
| 3 | `content:delete` |
| 4 | `users:view` |
| 5 | `users:invite` |
| 6 | `users:delete` |
| 7 | `users:modify` |
| 8 | `keys:create` |
| 9 | `keys:revoke` |
| 10 | `keys:view` |

## Users

Password for each user is `supersecret-<role>` (e.g. `supersecret-admin`).

| Name | Email | ULID | Role |
|------|-------|------|------|
| Keyholder User | `keyholder@example.com` | `01JMJMGHQSA2SHQ8S1T4JXABFJ` | keyholder |
| Admin User | `admin@example.com` | `01JN2YQ2VE9GMBRVACD15J1TFX` | admin |
| Gary Redfield | `gary@example.com` | `01JPYRNYMEHNEZCS0JYX1CP57A` | admin |
| Editor User | `editor@example.com` | `01JQNPQ1CHG36SV7NRQKTZB20R` | editor |
| Viewer User | `viewer@example.com` | `01JVWFBDXBNG6JNQZ36K2A8RT3` | viewer |

`01JN2YQ2VE9GMBRVACD15J1TFX` (Admin User) is the usual `created_by` parent for
OIDC clients and model conformance tests.

## API keys

All keys are created by Keyholder User (`01JMJMGHQSA2SHQ8S1T4JXABFJ`). List
queries return only non-revoked keys (3 of 5).

| Description | ULID | Client ID | Revoked | Permissions |
|-------------|------|-----------|---------|-------------|
| Read/view only keys | `01JNH8ZKWFJ2Z8E3GJTQTFPQCT` | `TPAkoalHEorqAENISHvxYY` | no | 2, 4, 10 |
| Full permission keys | `01JP72M6KXSFM1EQGKVDN2STAA` | `ISoIuDiGkpVpAyCrLGYrKU` | no | 1–10 |
| Revoked keys | `01JM6AHREW1YN8CMN1B4ZQCG5Z` | `yfoPxjgVyleDkpOPnNfsBG` | yes | none |
| Never used keys | `01JX2EX9XHAR5XHRWVZFCGAYK1` | `HcSloDQOcmfmExFvwdCMek` | no | 2, 4 |
| Revoked without use | `01JMS02ECAEK96CS5XA48WETZ9` | `jgSQoHTwJznURdRNBqbNOh` | yes (never seen) | none |

## OIDC clients

Both created by Admin User (`01JN2YQ2VE9GMBRVACD15J1TFX`).

| Name | Client ID | ULID |
|------|-----------|------|
| Full Metadata OIDC Client | `OidcClient1FullMetadata` | `01K80020000000000000000001` |
| Minimal Metadata OIDC Client | `OidcClient2Minimal` | `01K80040000000000000000002` |

## Vero tokens

| Token type | Email | Token ULID | Resource ULID |
|------------|-------|------------|---------------|
| `reset_password` | `observer@example.com` | `01JXTGSFRC88HAY8V173976Z9D` | `01HWQE3N4S6PZGKNCH7E617N8T` |

The fixture token is already sent (`sent_on` set) and includes a valid signature
blob for retrieve/scan tests.
