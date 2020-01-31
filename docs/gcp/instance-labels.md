# Instance metadata using necogcp

`necogcp` uses some metadata labels to control instances.


| Metadata Key     | Description                                                                                                                                          |
| ---------------- | ------------------------------------------------------                                                                                               |
| `shutdown-at`    | The timestamp that the instance should be deleted                                                                                                    |
| `extend`         | The timestamp that make the instance extend of life. The time of shutdown is `extend` + `expiration`. `expiration` is defined in configuration file. |

The time format is `time.RFC3339`.
