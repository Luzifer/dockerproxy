
1.11.0 / 2016-04-16
==================

  * Added support for new container label configuration

1.10.0 / 2016-04-16
==================

  * Use second level domain as CN
  * Fetch one certificate per second level domain

1.9.2 / 2016-03-27
==================

  * Fix: LetsEncrypt switched to X3 intermediate

1.9.1 / 2016-01-25
==================

  * Fix: We're using `user` module, so we need CGO

1.9.0 / 2016-01-25
==================

  * Added LetsEncrypt support

1.8.0 / 2015-08-29
==================

  * Fixed some linter errors
  * Rewrite redirects without HTTPs if SSL is enforced

1.7.1 / 2015-08-29
==================

  * Vendored deps
  * Documented authentication provider

1.7.0 / 2015-05-17
==================

  * Added authentication

1.6.1 / 2015-05-16
==================

  * Fixed missing colon in example config

1.6.0 / 2015-05-16
==================

  * Changed example config to yaml format
  * Refactored code; added yaml support for config

1.5.0 / 2015-02-08
==================

  * Force secure connections to use TLS to prevent SSL attacks
  * Made docker container usable without building an own container

1.4.1 / 2014-10-28
==================

  * Critical Fix: Removed securiy hole which could be used to attack various sites using post requests

1.4.0 / 2014-10-16
==================

  * Added basic HTTP logging
  * Fixed bug in port export in docker readme
  * Fixed copy and paste bug in readme

1.3.0 / 2014-10-05
==================

  * Added X-Forwarded-For header
  * Added support for pseudo load balanced containers
  * Documented SSL certificate ordering

1.2.0 / 2014-10-04
==================

  * Disabled proxy functions as we are not a proxy

1.1.0 / 2014-10-04
==================

  * Added force_ssl flag

1.0.0 / 2014-10-04
==================

  * Initial running version

