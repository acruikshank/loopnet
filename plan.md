* Start up a node, it outputs it's address
* Connect another node to the first
* The nodes add each other to their peerstores
* Upon receiving a connection, a node will gossip out the state of the new node including its address
* When a node receives a gossip message containing a new peer, it will add it to its peerstore
* Each node can change it's current state by muting, unmuting, or changing pitch
* The consensus state is a map of all peer addresses to their current note state
* Each node plays an appegiator by sorting each node's pitch and playing them lowest to highest and back down.
* We can use communication failures (or successes) to track presence. Presence will be gossiped out as well
