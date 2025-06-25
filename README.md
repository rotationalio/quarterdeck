# Quarterdeck

[![Tests](https://github.com/rotationalio/quarterdeck/actions/workflows/tests.yaml/badge.svg)](https://github.com/rotationalio/quarterdeck/actions/workflows/tests.yaml)

**Distributed authentication and authorization service for Rotational applications.**

Quarterdeck is a JWT issuer that also provides JWKS to verify that keys have been issued from Quarterdeck. Quarterdeck provides middleware for other Go applications to easily authenticate JWT claims provided in requests and the claims themselves provide authorization scope for access. Quarterdeck also provides user and api key management via an API and a simple user interface.

Projects using Quarterdeck:

- [Endeavor](https://github.com/rotationalio/endeavor)
- [HonuDB](https://github.com/rotationalio/honu)

## License

This project is licensed under the BSD 3-Clause License. See [`LICENSE.txt`](./LICENSE.txt) for details. Please feel free to use Quarterdeck in your own projects and applications.

## About Rotational Labs

Quarterdeck is developed by [Rotational Labs](https://rotational.io), a team of engineers and scientists building AI infrastructure for serious work.