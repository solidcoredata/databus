# DATABUS

See [system.go](bus/system.go) for details.

How does this tool work?

 * Establish a folder with schema and app definitions.
 * Run a tool over it.
 * The tool can run other tools to process the definitions and
   return additional components.
 * There is also a stateful schema folder that annotates the various version
   schemas.
 * In addition to knowing what schema changed, the other definitions
   may be hashed so if the hashes changed, downstream components may be
   updated on deploy, without needing to re-deploy everything.

The tool can be divided up into different parts:
 1. A stateful schema tool (bus commit) that manages logical changes.
 2. A tool (bus generate) that generates code or other data structures from
    the schema and delta schema.
 3. A tool (bus deploy) that deploys a commit to servers.
 4. A tool (bus ui) that provides a visual development experience
    to define the schema.

The bus commit tool seems easy enough to define: definition files are
created on the local file system, you can check them in to modify the
schema delta.

The bus generate tool could be tricky if it is extensible. Right now
each configuration declares a type. Each project declares a list of
extensions. When loaded, each extension declares what types it should
be sent. That part is tricky, but already done. What is not done is
what to do with the results of the generation. A few thoughts:

 * I want to make it easy for tools to only update the parts of an application
   that need updating. This also makes it safer to hot-load non-framework
   code but still requires a full-load for framework changes.
 * The code should be deployed from a single root.
   `root://<extension-type>/...` note this is the extension type. Because
   a single extension controls each extension-type root, it is expected
   to manage everything under that path itself.
 * During a deploy, each extension also manages the deploy, so it knows
   how to read the extension managed root.

