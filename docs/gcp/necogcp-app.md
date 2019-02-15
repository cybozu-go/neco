necogcp-app
===========

`necogcp-app` is REST API server running on the Google App Engine. It is deployed as Go 1.11 of the GAE standard.
Cron Job on the App Engine calls REST API itself, and control running GCE instances.
Configuration file would be deployed together for loading parameter.

API
---

See [api.md](api.md)

Configuration file
------------------

See [config.md](config.md)

CronJob
-------

If you want to change shutdown schedule, please edit [../../pkg/necogcp-app/cron.yaml](../../pkg/necogcp-app/cron.yaml)
then run `make -f Makefile.gcp deploy`.
