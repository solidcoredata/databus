# Data Bus

Let's take a small side track from coding the basic poc.
How should the data bus be modeled / represented?

 * Node
   - Role
     - Field


Not only that, but some Roles are directional.
Also, Some nodes may be "adapters" and have to "sides".
Maybe each Role could specify "side: left|right|both".

A node should not implement any functionality.
Each node should have a type that goes with it.

There must be a named node type view, that projects the bound
roles and fields on a single representation. This is the view that
are consumed by controls.

Lastly, a node's definition should also containe the other bound nodes that
project onto this node. This is used for validation / refactoring /
and projecting properties. Binding a UI to a query should be easy.
I'm not sure how to represent a query binding to N number of tables.

