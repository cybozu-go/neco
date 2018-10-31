CI/CD
=====

Continuous Integration in **Neco** is served by CircleCI workflow.

`neco-updater` service works as Continuous Delivery to update components in the boot server automatically.

Architecture
------------

![architecture](http://www.plantuml.com/plantuml/png/ZPJFRXen4CRlVeeHfqJAKY0tL4LfSsWhHQtKN7Cnza1SlFRMVqAYYdVlxDhh3R1fkI3Z-StdRyRZlVM1kn1hpHoAmCr68nWKTYfn1NyOmB0zQVSdm7q7Y5gUHglOI1xGTLHUZr0xwxOPIajYBiY6MdCH_7HZBzjGsJXKsA11Hy9LYNT2UTiwjcTCQ1ibJBwey3MkKbY5fvWgCbQc6ljItiGElUO-b9ffJSo-ry0WPCEztyaM6FwzfpSGHNNOwhNtlVdVaRzELMfukpw-3M5DuCgWyt3X-OflkRa2iSKhyEZMq-dqiajLDT-W7sJlhCCV3t3NPyEzCl6bGmM5h3yw9_5jKz-S_SseeBW6eQFlhhlqTVBPsW0Fw9v9UjR9hcXdbhiXUI2hlkjTwTfihF7GSB4bwu-66mcU19L0_sYEQXqQM2eMjcvcXwVb78hsonROZvgU5zFpqolqlCPYffmsVrTiKSGMvuejyXYSYbqNiMiIenifCm-Lj3jJtGnlPWavYAmTdWAKb2Kv2KxXCm9fUsKDotDx3ff7vHoKfISELMmE3JfSeojOfh9YkiGbI6oqVMLfY4imiHIblzfooTZ1EtgVoyDVERLv2cF0aOqoBKk8JGi0znw3lteq99DSfG5Lc_P9MECPor-Aef6_WoEjoP52TezX2P-aX8yDjKUzt7mGqJaq8HixHdSyYATJcEKlTC7ReI6SlCS6xdz-9zUA3AVVIbl1zMZE_2JdN_JY_xJ6TSZqzH9-MMKoh94_OxjcjsXahFLV)
<!-- go to http://www.plantuml.com/plantuml/ and enter the above URL to edit the diagram. -->

CI flow
-------

CI is running as nightly job.

1. Checkout `master` branch then merge changes into `release` branch.
1. Run `generate-artifacts` to retrieve latest version of components, then generate `artifacts_release.go`.
1. Build neco binaries with `artifacts_release.go`.
1. Run [dctest](../dctest/) with these binaries.
1. If dctest is passed, push `release` branch to remote branch in [github.com/cybozu-go/neco](https://github.com/cybozu-go/neco).
1. When above job pushes changes to `release` branch, build neco deb package, and release it to the GitHub Release as pre-release.

When administrator confirms version of the pre-release, publish it to be deployed by CD flow.

CD flow
-------

1. A back ground service `neco-updater` detect a new release of neco repository.
1. If new release exists, `neco-updater` add information to the etcd key `<prefix>/current`.
1. `neco-worker` to update `neco` package, then restart `neco-worker` service.
1. `neco-worker` installs/updates container images, and sabakan contents.
