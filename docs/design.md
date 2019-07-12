# Design Notes

For now, have each extension write files to the each extension root.
Don't pass back the files for the bus tool to write.

 1. Write some config.
 2. Run tool.
 3. Tool validates config.
   a. Tool extensions validate config.
 4. Tool generates usable output.
   a. Tool extensions generate usable output.
   b. Extentions return mutliple files to tool.
   c. Tool stores the files under extension+revision namespace.
 5. Tool is used to deploy.
   a. Extensions look for currently deployed version.
   b. Tool determins what updates need to happen and in what order
      based on definition relationships and update order.
   c. Tool calls extensions to deploy updates in turn.

## Storing Versions.

Store version by unique hash. But before it can bump the head version pointer
it must ensure that the changes are compatible. Unsure of many details,
but want to allow for conflict free merges in the normal case, so probably
can't rely on a incrementing number in this case.
