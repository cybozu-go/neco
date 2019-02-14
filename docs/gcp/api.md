GAE app REST API
================

- [POST /shutdown](#shutdown)
- [POST /extend](#extend)

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

<a name="extend" />`POST /extend`
---------------------------------

Extend 1 hour the given instance. A user can specify the following URL queries.

| Query                 | Description                             |
| --------------------- | --------------------------------------- |
| `instance=<instance>` | Instance name in the neco-test project. |

### Successful response

- HTTP status code: 200 OK
- HTTP response header: Content-Type: application/json
- HTTP response body: list of stopped and deleted instances.

```json
{
  "extended": "neco-1234",
  "time": 1550109818,
  "availableUntil": "Thu Feb 14 11:03:17 UTC 2019",
  "status": 200
}
```

### Failure responses

- 400 Bad Request: missing or wrong configuration file or wrong query parameter.
- 500 Internal Server Error: other errors.
