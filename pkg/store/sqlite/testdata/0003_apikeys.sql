-- One API key with complete permissions and one API key with read-only permissions
-- client ID: ISoIuDiGkpVpAyCrLGYrKU    secret: Dah5FqQT8tHtC9UablExfhb2GbmfrJrSiHAXBnDzKI1OQoTa
-- client ID: TPAkoalHEorqAENISHvxYY    secret: HEACkMCWytZquAQQAQoxHKs0LB3h0Mppx93PeSpA5nCVpxYJ
INSERT INTO api_keys (id, description, client_id, secret, created_by, last_seen, revoked, created, modified) VALUES
    (x'0195628fcf8f90be870e12d5f4fb5d9a', 'Read/view only keys', 'TPAkoalHEorqAENISHvxYY', '$argon2id$v=19$m=65536,t=1,p=2$8J11ntVv8i3YBGA74QCS/w==$mOINU411zwT0lNO03UBkMI7l9Mz7rA3XAiQpDIXVVh0=', x'0195254846f950b31ba321d125d52df2', '2025-05-24T18:41:58Z', NULL, '2025-03-04T19:09:06Z', '2025-05-24T18:41:58Z'),
    (x'01958e2a1a7dcbe8175e13db6a2ce94a', 'Full permission keys', 'ISoIuDiGkpVpAyCrLGYrKU', '$argon2id$v=19$m=65536,t=1,p=2$XndK1CI4C1mbOcE25aV8PA==$9NlkyH58LyOmH7oNg38VmB49uoIpa89k7afqABbh+o8=', x'0195254846f950b31ba321d125d52df2', '2025-05-08T22:17:23Z', NULL, '2025-03-13T06:21:18Z', '2025-05-08T22:17:23Z'),
    (x'01950ca8e1dc0faa8652a1593f7640bf', 'Revoked keys', 'yfoPxjgVyleDkpOPnNfsBG', '$argon2id$v=19$m=65536,t=1,p=2$5nlg+OJeE7XVZVPjtd/MXQ==$CCndtaAx1VCSbJSLvQnKaXaNlBm5/5+KlPrcM2kTne0=', x'0195254846f950b31ba321d125d52df2', '2025-02-18T17:40:55Z', '2025-04-10T09:31:40Z', '2025-02-16T02:49:09Z', '2025-04-10T09:31:40Z'),
    (x'019744eea7b1560bd8e39bfbd9057a61', 'Never used keys', 'HcSloDQOcmfmExFvwdCMek', '$argon2id$v=19$m=65536,t=1,p=2$ud0irPXlCt8N2YbIHb2uSQ==$uVMKjyUCdLm4/spu4YnQZ3erkHJFer3W93V1BAavIVc=', x'0195254846f950b31ba321d125d52df2', NULL, NULL, '2025-06-06T11:09:40Z', '2025-06-09T21:54:33Z'),
    (x'01953201398a74d26664bd5111c76be9', 'Revoked without use', 'jgSQoHTwJznURdRNBqbNOh', '$argon2id$v=19$m=65536,t=1,p=2$ExKPQhQKaTiL4Q/F0EZ3HA==$BHJgxhoykEqDtREyNhZnVKQoHyIRMbW1l1M3bZFI8Us=', x'0195254846f950b31ba321d125d52df2', NULL, '2025-05-22T04:55:20Z', '2025-02-23T08:51:35Z', '2025-05-22T04:55:20Z')
;

INSERT INTO api_key_permissions (api_key_id, permission_id, created, modified) VALUES
    (x'0195628fcf8f90be870e12d5f4fb5d9a', 2, '2025-03-04T19:09:06Z', '2025-03-04T19:09:06Z'),
    (x'0195628fcf8f90be870e12d5f4fb5d9a', 4, '2025-03-04T19:09:06Z', '2025-03-04T19:09:06Z'),
    (x'0195628fcf8f90be870e12d5f4fb5d9a', 10, '2025-03-04T19:09:06Z', '2025-03-04T19:09:06Z'),
    (x'01958e2a1a7dcbe8175e13db6a2ce94a', 1, '2025-03-13T06:21:18Z', '2025-03-13T06:21:18Z'),
    (x'01958e2a1a7dcbe8175e13db6a2ce94a', 2, '2025-03-13T06:21:18Z', '2025-03-13T06:21:18Z'),
    (x'01958e2a1a7dcbe8175e13db6a2ce94a', 3, '2025-03-13T06:21:18Z', '2025-03-13T06:21:18Z'),
    (x'01958e2a1a7dcbe8175e13db6a2ce94a', 4, '2025-03-13T06:21:18Z', '2025-03-13T06:21:18Z'),
    (x'01958e2a1a7dcbe8175e13db6a2ce94a', 5, '2025-03-13T06:21:18Z', '2025-03-13T06:21:18Z'),
    (x'01958e2a1a7dcbe8175e13db6a2ce94a', 6, '2025-03-13T06:21:18Z', '2025-03-13T06:21:18Z'),
    (x'01958e2a1a7dcbe8175e13db6a2ce94a', 7, '2025-03-13T06:21:18Z', '2025-03-13T06:21:18Z'),
    (x'01958e2a1a7dcbe8175e13db6a2ce94a', 8, '2025-03-13T06:21:18Z', '2025-03-13T06:21:18Z'),
    (x'01958e2a1a7dcbe8175e13db6a2ce94a', 9, '2025-03-13T06:21:18Z', '2025-03-13T06:21:18Z'),
    (x'01958e2a1a7dcbe8175e13db6a2ce94a', 10, '2025-03-13T06:21:18Z', '2025-03-13T06:21:18Z')
;