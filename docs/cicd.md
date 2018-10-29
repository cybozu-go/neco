CI/CD
=====

Continuous Integration in **Neco** is served by CircleCI workflow.

`neco-updater` service works as Continuous Delivery to update components in the boot server automatically.

Architecture
------------

![architecture](http://www.plantuml.com/plantuml/png/ZPInZjim38PtFGNXAG5YFq26uj0Rsg50Xrkxo1BZ2d4aLvBlUZZatIl2jAG4QIwRHNxyHT8KdqAKFiwdKNXKKTfXHB2e77m8W4rocODKCNI3su8Ca0t9MmAQCFVgfFTWR98RnntCavOHs_JTK1ZRqTyEOph8NXBEPqzdSQuI6z2Y9pAdGJHRdOSFfaiPBKiHH-VrIAHGevirDDzC_3xt3I63YR_ddce7wpHtWaChtqKHvEiq9W46qtTYpgi6HgKd6SAR9g2S_gTNYAnQJAlsUKt-popVE-C80_eclLfDEHkbiUW3RAYVHsbte8wuWu3-q7NTTlb19pbWABBFpkFF5tXUe-67iVDVGa4bbmjNzomyDPLgXkQhSn5UqB-Yfo3ewVnmQZcjWbo6fZR09DMHaePDQKyLEXq72j8o9kc0m3UGIrCFDrkW3b3APO1QxTvi-wMC-JxFdA3kPY27xC5Zz0PV4LAjmJWh-CS-Wd8l7q55VaEGXhhEaU03-amMa7LB6nEhSHhT-ms86fRTopnaNwRtG9RHIIqkXl8koSDq3n7La_-ilWhDchgdBK9Ia5B267QGRkGgfDLW1cjYYWw2_Zgq7EDnC269BK_Ls8ExBhswxMP9To31so1ZrGQgiCfAfJEtRaLOzmlyik1dkooSUi6A9xGwRV1_)
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
1. If new release exists, download `neco` deb package, and update `neco` in the boot server.
1. Invoke `neco update-all` to update `neco`, container images, and sabakan contents.
