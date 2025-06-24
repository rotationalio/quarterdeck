-- Initial schema for Quarterdeck authentication service.
-- NOTE: all primary keys are ULIDs but rather than using the 16 byte blob version of
-- the ULIDs we're using the string representation to make database queries easier and
-- because use of the sqlite3 storage backend isn't considered to be performance
-- intensive. NOTE: the oklog/v2 ulid package provides Scan for both []byte and string.
BEGIN;

COMMIT;