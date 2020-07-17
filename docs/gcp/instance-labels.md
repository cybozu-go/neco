# Instance metadata using necogcp

`necogcp` uses some metadata labels to control instances.


| Metadata Key  | Description                                       |
| ------------- | ------------------------------------------------- |
| `shutdown-at` | The timestamp that the instance should be deleted |

The time format is `time.RFC3339`.
