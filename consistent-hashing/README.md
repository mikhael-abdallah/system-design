# Consistent Hashing in Go

This project provides a robust, simplified implementation of the Consistent Hashing algorithm in Go. It is designed to be a clear and practical demonstration of how consistent hashing works, including the crucial concept of Virtual Nodes (VNodes) to ensure uniform data distribution.

The accompanying simulation demonstrates the algorithm's primary benefit: minimizing data migration when nodes are added to or removed from a cluster.

## Table of Contents

- [Core Concepts](#core-concepts)
  - [The Problem with Modulo Hashing](#the-problem-with-modulo-hashing)
  - [The Solution: The Hash Ring](#the-solution-the-hash-ring)
  - [The Refinement: Virtual Nodes (VNodes)](#the-refinement-virtual-nodes-vnodes)
- [How It Works: Code Implementation](#how-it-works-code-implementation)
  - [The ConsistentHashing Struct](#the-consistenthashing-struct)
  - [Key Operations: AddNode and RemoveNode](#key-operations-addnode-and-removenode)
  - [A Note on Performance and Real-World Optimizations](#a-note-on-performance-and-real-world-optimizations)
- [Key Features](#key-features)
- [How to Run the Simulation](#how-to-run-the-simulation)
- [Understanding the Output](#understanding-the-output)
- [Configuration](#configuration)

## Core Concepts

### The Problem with Modulo Hashing

A naive approach to distributing data across N servers is to use a modulo operation: `server_index = hash(key) % N`. This system is extremely fragile. If you add or remove a single server, N changes, and nearly every key will map to a new server. This causes a "catastrophic reshuffle," where most of your data needs to be moved, crushing your system's performance and availability.

### The Solution: The Hash Ring

Consistent Hashing solves this by mapping servers and keys to an abstract circle, or hash ring.

- **The Ring**: Imagine a circle representing all possible hash values (e.g., from 0 to 2³² − 1).

- **Placing Nodes**: Each server (node) is assigned a position on the ring by hashing its name (e.g., `hash("node-0")`).

- **Placing Keys**: Each data key is also placed on the ring by hashing it (e.g., `hash("user-123")`).

- **Ownership Rule**: To determine which node owns a key, we start at the key's position on the ring and move clockwise until we find the first node. That node is the owner.

When a node is removed or added, only the keys in the arc immediately preceding it are affected. The vast majority of keys remain on their existing nodes.

### The Refinement: Virtual Nodes (VNodes)

Placing only one point per node on the ring can lead to non-uniform data distribution; by random chance, one node might get a huge arc while another gets a tiny one.

VNodes solve this. Instead of mapping each physical node to a single point, we map it to hundreds of "virtual" points on the ring (e.g., `hash("node-0#0")`, `hash("node-0#1")`, ...).

This has two major benefits:

- **Uniform Distribution**: It spreads each physical node's presence across the ring, making it statistically very likely that each node will be responsible for a similar amount of the hash space (and thus, a similar amount of data).

- **Smoother Rebalancing**: When a node leaves, its load is distributed among many neighbors (the nodes clockwise to each of its VNodes) instead of overwhelming a single node.

## How It Works: Code Implementation

The logic is encapsulated within the `ConsistentHashing` struct and its methods.

### The ConsistentHashing Struct

```go
type ConsistentHashing struct {
    ring    []uint32                       // The sorted Hash Ring of VNodes
    hashMap map[uint32]string              // Maps a VNode hash to its physical node's name
    nodes   map[string]map[string]string   // The actual data storage
    vnodes  int                            // The number of VNodes per physical node
}
```

- **ring**: A sorted slice of `uint32` values representing the positions of all VNodes on the hash ring. Keeping it sorted allows for efficient lookups using binary search.

- **hashMap**: A lookup table to find the physical node name (e.g., "node-0") from a VNode's hash value.

- **nodes**: A map simulating the actual storage servers, where the user records are stored.

- **vnodes**: A configuration parameter, set to 100 in our simulation, defining how many virtual points each physical node gets.

### Key Operations: AddNode and RemoveNode

The core logic for rebalancing the cluster is encapsulated directly within these methods.

#### `RemoveNode(nodeName string)`

1. **Identify Data**: First, it takes a reference to all data currently stored on the node to be removed.

2. **Update Ring**: It removes all VNodes corresponding to `nodeName` from the ring and the `hashMap`. The ring is now in its final state, without the removed node.

3. **Redistribute Data**: It iterates through the identified data. For each key, it calls `GetNode(key)` to find its new owner on the modified ring. The data is then moved to the new owner's storage.

#### `AddNode(nodeName string)`

1. **Update Ring First**: The VNodes for the new node are added to the ring and `hashMap`. The ring is re-sorted and is now in its final state, including the new node.

2. **Claim Data**: The method then iterates through every other node in the cluster. For each key on those nodes, it checks if its new owner is the node that was just added by calling `GetNode(key)`.

3. **Migrate Data**: If a key's ownership has changed to the new node, it is moved from its old location to the new node's storage.

### A Note on Performance and Real-World Optimizations

The implementation of `AddNode` in this project is designed for clarity and educational purposes. It correctly demonstrates the logic by iterating through every key in the entire cluster to check if its ownership has changed.

This approach is highly inefficient and would not be used in a production system.

Scanning billions of keys to add a single node is not feasible. Real-world systems like Cassandra or DynamoDB use significant optimizations:

- **Hash-Indexed Local Storage**: Nodes don't just store keys in a simple map. They use data structures (like B-Trees) that are indexed or sorted by the hash of the key. This allows a node to perform a very fast local query like, "Give me all keys whose hashes fall within the range (X, Y]."

- **Partition Handoff**: When a new node is added, it doesn't need to check every key. The system calculates exactly which hash ranges the new node is now responsible for. The "old" owner nodes can then efficiently find all keys within those specific ranges (using the optimization above) and transfer them as a single block of data to the new node. This process is often called "partition handoff" and happens in the background without scanning unrelated data.

This project deliberately omits these complex optimizations to keep the core consistent hashing logic front and center.

## Key Features

- **Self-Contained Logic**: Data migration is handled internally by the `AddNode` and `RemoveNode` methods, providing a clean API.

- **VNode Implementation**: Uses virtual nodes for excellent data distribution, avoiding hotspots.

- **Efficient Lookups**: Uses binary search (`sort.Search`) for fast node lookups on the ring.

- **Detailed Simulation**: The main function simulates a real-world scenario:
  - Initial creation of 1,000,000 user records
  - Distribution across 10 nodes
  - Graceful removal of a node and re-distribution of its keys
  - Addition of a new node and migration of keys from existing nodes

- **Verification Step**: After all operations, it verifies that every single key is located on the correct node, proving the algorithm's correctness.

## How to Run the Simulation

### Prerequisites

- Go (version 1.18 or later) must be installed on your system.

### Steps

1. Save the code as a file named `main.go`.

2. Open your terminal and navigate to the directory where you saved the file.

3. Run the simulation with the following command:

```bash
go run main.go
```

## Understanding the Output

When you run the simulation, you will see a detailed log of the operations. Pay attention to:

- **Initial Status**: Observe how the 1,000,000 records are distributed. Thanks to VNodes, the load on each of the 10 nodes should be relatively balanced (around 100,000 records each).

- **Node Removal**: When node-4 is removed, notice that only its ~100,000 records are moved. The records on the other 8 nodes are untouched. The log will show exactly where these records were moved to.

- **Node Addition**: When node-10 is added, it "claims" records from several other nodes—those that now fall into its new hash ranges. Again, only a fraction of the total keys are moved.

- **Final Verification**: The last line of the output is crucial, confirming that despite all the shuffling, the system is in a consistent state and every key is exactly where it should be.

## Configuration

The number of virtual nodes is a key parameter for tuning. It's set as a constant in `main.go`:

```go
const numVNodes = 100
```

- A higher number leads to better data distribution and balance.
- A lower number reduces memory and CPU overhead for managing the ring.
- A value between 100 and 256 is a common industry standard, providing an excellent trade-off between balance and performance.
