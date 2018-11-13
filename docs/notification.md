Notification
============

`neco-updater` will post following notifications to Slack.

* StartRequest
  * Update: Start updating by the new release.
  * Reconfigure: Start reconfiguration by the new set of boot servers.
* Succeeded: The update request was succeeded.
* Failure
  * Aborted: The update request was aborted due to an error on any server.
  * Timeout: The update request was aborted because boot servers did not return a responce in the specified time. 

