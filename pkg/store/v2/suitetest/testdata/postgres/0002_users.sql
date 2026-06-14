INSERT INTO users (id, name, email, password, last_login, created, modified) VALUES
    (decode('0195254846f950b31ba321d125d52df2', 'hex'), 'Keyholder User', 'keyholder@example.com', '$argon2id$v=19$m=65536,t=1,p=2$6wYd2N9TUl8TzcxywXtj+Q==$sQSXLgnOfZprJkV0xlgcF9iMLgXlqEkZwZSbeM9NFmQ=', '2025-04-09T03:19:55Z', '2025-02-20T21:34:08Z', '2025-04-09T03:19:55Z'),
    (decode('019545eb8b6e4c28bc6d4c684b20e9fd', 'hex'), 'Admin User', 'admin@example.com', '$argon2id$v=19$m=65536,t=1,p=2$9ihQHJnCW+bojgqoUWYc/A==$GBaUbq36VeFsoqpHfDZXSzUu+1JUXjO2ein7Bis2r4I=', '2025-05-06T09:51:19Z', '2025-02-27T05:40:19Z', '2025-05-06T09:51:19Z'),
    (decode('0195bd8afa8e8d5df66412f742cb14ea', 'hex'), 'Gary Redfield', 'gary@example.com', '$argon2id$v=19$m=65536,t=1,p=2$gSf2/qcUTOI0wPMtb/PbRQ==$BwsN4Ditjj96S2GF1bqC2qE0BaknigvtXpDsiCJBN+I=', '2025-04-28T18:44:16Z', '2025-03-22T11:09:16Z', '2025-04-28T18:44:16Z'),
    (decode('0195eb6b859180cd9d9eb8bcf5f58818', 'hex'), 'Editor User', 'editor@example.com', '$argon2id$v=19$m=65536,t=1,p=2$oPREW7ztC12IG7EVldbneA==$K/4cNUUt661D30ufLmTTN/bZD0WSig/FrbqOmkOoX9I=', '2025-04-29T15:02:51Z', '2025-03-31T08:57:27Z', '2025-04-29T15:02:51Z'),
    (decode('0196f8f5b7abac0d2adfe334c4a46343', 'hex'), 'Viewer User', 'viewer@example.com', '$argon2id$v=19$m=65536,t=1,p=2$shIXJGz48Q4DeqIG7/G9AQ==$3/3aNigBqUmSqRNJ/wahDYGkDGd3bbTA2fOoh6MXJas=', '2025-05-27T22:56:13Z', '2025-05-22T17:06:15Z', '2025-05-27T22:56:13Z')
;

INSERT INTO user_roles (user_id, role_id, created) VALUES
    (decode('0195254846f950b31ba321d125d52df2', 'hex'), 4, '2025-02-20T21:34:08Z'),
    (decode('019545eb8b6e4c28bc6d4c684b20e9fd', 'hex'), 1, '2025-02-27T05:40:19Z'),
    (decode('0195bd8afa8e8d5df66412f742cb14ea', 'hex'), 1, '2025-03-22T11:09:16Z'),
    (decode('0195eb6b859180cd9d9eb8bcf5f58818', 'hex'), 2, '2025-03-31T08:57:27Z'),
    (decode('0196f8f5b7abac0d2adfe334c4a46343', 'hex'), 3, '2025-05-22T17:06:15Z')
;

INSERT INTO vero_tokens (id, token_type, resource_id, email, expiration, signature, sent_on, created, modified) VALUES
    (decode('0197750cbf0c4222af236138d2737d2d', 'hex'), 'reset_password', decode('018f2ee1d49935bf09d5913b8c13d51a', 'hex'), 'observer@example.com', '2024-11-16T17:43:53Z',
     decode('0197750cbf0c4222af236138d2737d2db0ccb8e986f8a8c93097ffe008098fb5a6f3d5b5844b140accf8033974223d6f390fb4fdd3afe5f991a1c6ba56395cd93013783b0c5174a3362c22e0fa1f9f40d23b4abf4405cd24b60eacf0ef001a3abc0c9e803118ee98bb7ffbd563cd021c95bde00a88f26b4a55', 'hex'),
     '2024-11-16T17:28:45Z', '2024-11-16T17:28:57Z', '2024-11-16T17:28:57Z')
;
