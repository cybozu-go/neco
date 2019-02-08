GAE app
=======

GAE app is REST API server running on the Google App Engine. It is deployed as Go 1.11 of the GAE standard.
Cron Job on the App Engine calls REST API itself, and control running GCE instances.
Configuration file would be deployed together for loading parameter.

API
---

See [api.md](api.md)

Configuration file
------------------

See [config.md](config.md)
