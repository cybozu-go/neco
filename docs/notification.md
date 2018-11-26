Notification
============

`neco-updater` will post notification to Slack when update is started or finished.

Start update
------------

When `neco-updater` starts updating, it notifies the following message.

* Update: Start updating since the new neco package is released.
* Reconfigure: Start reconfiguration since the set of boot servers is changed.

Finish update
-------------

When `neco-updater` finish updating, it notifies the following message.

* Succeeded: The update was succeeded.
* Aborted: The update was aborted due to an error on any server.
* Timeout: The update was aborted because boot servers did not return a response in time. 
