GAE app REST API
================

- [POST /shutdown](#shutdown)

Failure response format
-----------------------

Failure response body is a JSON object with these fields:

- `status`: HTTP status code
- `error`: Error message

<a name="shutdown" />`POST /shutdown`
-------------------------------------

Shutdown all instances and delete target instances in `neco-gcp.yml`.

### Successful response

- HTTP status code: 200 OK
- HTTP response header: Content-Type: application/json
- HTTP response body: list of stopped and deleted instances.

```json
{
  "stopped": ["docker-test"],
  "deleted": ["host-vm"],
  "status": 200
}
```

### Failure responses

- 400 Bad Request: missing or wrong configuration file.
- 500 Internal Server Error: other error.
