#!/bin/sh
# Import a mailbox by its ID. The password and the Microsoft client secret are
# write-only: set them in the configuration, the next apply sends them again.
terraform import youtrack_mailbox.support 254-0
