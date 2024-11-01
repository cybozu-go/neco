CI/CD
=====

Continuous Integration in **Neco** is served by CircleCI workflow.

`neco-updater` service works as Continuous Delivery to update components in the boot server automatically.

Architecture
------------

![architecture](http://www.plantuml.com/plantuml/svg/ZPJFRXen4CRlVeeHfqJAKY0tL4LfSsWhHQtKN7Cnza1SlFRMVqAYYdVlxDhh3R1fkI3Z-StdRyRZlVM1kn1hpHoAmCr68nWKTYfn1NyOmB0zQVSdm7q7Y5gUHglOI1xGTLHUZr0xwxOPIajYBiY6MdCH_7HZBzjGsJXKsA11Hy9LYNT2UTiwjcTCQ1ibJBwey3MkKbY5fvWgCbQc6ljItiGElUO-b9ffJSo-ry0WPCEztyaM6FwzfpSGHNNOwhNtlVdVaRzELMfukpw-3M5DuCgWyt3X-OflkRa2iSKhyEZMq-dqiajLDT-W7sJlhCCV3t3NPyEzCl6bGmM5h3yw9_5jKz-S_SseeBW6eQFlhhlqTVBPsW0Fw9v9UjR9hcXdbhiXUI2hlkjTwTfihF7GSB4bwu-66mcU19L0_sYEQXqQM2eMjcvcXwVb78hsonROZvgU5zFpqolqlCPYffmsVrTiKSGMvuejyXYSYbqNiMiIenifCm-Lj3jJtGnlPWavYAmTdWAKb2Kv2KxXCm9fUsKDotDx3ff7vHoKfISELMmE3JfSeojOfh9YkiGbI6oqVMLfY4imiHIblzfooTZ1EtgVoyDVERLv2cF0aOqoBKk8JGi0znw3lteq99DSfG5Lc_P9MECPor-Aef6_WoEjoP52TezX2P-aX8yDjKUzt7mGqJaq8HixHdSyYATJcEKlTC7ReI6SlCS6xdz-9zUA3AVVIbl1zMZE_2JdN_JY_xJ6TSZqzH9-MMKoh94_OxjcjsXahFLV)
<!-- go to http://www.plantuml.com/plantuml/ and enter the above URL to edit the diagram. -->

CI flow
-------

### After `main` merge

1. Check out `main` branch then merge changes into `release` branch.
2. Run [dctest][] but fewer test cases from regular dctest.
3. If dctest is passed, [cybozu-neco][] pushes changes to `release` branch to remote branch in [github.com/cybozu-go/neco](https://github.com/cybozu-go/neco).
4. Also, [cybozu-neco][] also applies and pushes a tag `test-YYYY.MM.DD-UNIQUE_ID` to the remote.

Regular test cases of dctest are also run in parallel.

### Nightly workflow

Same workflow process but it works on `release` branch. This is run as stability aspects.

### Are you ready to deploy for the staging data center?

1. Choose a tag in [release page](https://github.com/cybozu-go/neco/releases) which starts with `test-` of what you want.
2. Create a new tag which has `release-` prefix. Note that date and UNIQUE_ID are same as before.
    ```console
    $ git tag release-YYYY.MM.DD-UNIQUE_ID test-YYYY.MM.DD-UNIQUE_ID
    $ git push origin --tags
    ```
3. CI workflow builds a debian package `neco-YYYY.MM.DD-UNIQUE_ID.deb` and uploads it to GitHub Release.
4. `neco-updater` on staging starts CD flow described below.

### Are you ready to deploy for production data center?

1. Choose a pre-release tag in [release page](https://github.com/cybozu-go/neco/releases) which starts with `release-` of what you want.
2. Uncheck `This is a pre-release`, then click `Publish release`.
3. `neco-updater` on production starts CD flow described below.

CD flow
-------

1. A service `neco-updater` detects a new release of GitHub Release on neco repository.
2. If new release exists, `neco-updater` add information to the etcd key `<prefix>/current`.
3. `neco-worker` to update `neco` package, then restart `neco-worker` service.
4. `neco-worker` installs/updates container images, and sabakan contents.

Glossary
--------

- tag: `test-YYYY.MM.DD-UNIQUE_ID`

    It is a candidate release version which is passed [dctest].

- tag: `release-YYYY.MM.DD-UNIQUE_ID`

    It is an already released version with `neco` Debian package.

    -  `neco-updater` on **staging** finds a new **pre-release** of them.
    -  `neco-updater` on **production** finds a new **release** of them.

- [cybozu-neco][] 🐈

    A bot GitHub Account for handling CI jobs using GitHub.

[dctest]: ../dctest/
[cybozu-neco]: https://github.com/cybozu-neco
