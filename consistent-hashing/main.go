package main

import (
	"fmt"
	"hash/crc32"
	"sort"
	"strconv"
)

type ConsistentHashing struct {
	ring    []uint32
	hashMap map[uint32]string
	nodes   map[string]map[string]string
	vnodes  int
}

func NewConsistentHashing(vnodes int) *ConsistentHashing {
	return &ConsistentHashing{
		ring:    make([]uint32, 0),
		hashMap: make(map[uint32]string),
		nodes:   make(map[string]map[string]string),
		vnodes:  vnodes,
	}
}

// hashKey generates a uint32 hash for a string key.
func hashKey(key string) uint32 {
	return crc32.ChecksumIEEE([]byte(key))
}


// GetNode finds the node responsible for a data key.
func (ch *ConsistentHashing) GetNode(key string) (string, error) {
	if len(ch.ring) == 0 {
		return "", fmt.Errorf("no nodes in the ring")
	}

	keyHash := hashKey(key)

	// Find the first node in the ring whose hash is >= the key hash.
	idx := sort.Search(len(ch.ring), func(i int) bool {
		return ch.ring[i] >= keyHash
	})

	// If the key hash is greater than all node hashes,
	// it "wraps around" the ring and belongs to the first node.
	if idx == len(ch.ring) {
		idx = 0
	}

	nodeHash := ch.ring[idx]
	return ch.hashMap[nodeHash], nil
}

// AddNode adds a node and redistributes data from other nodes to it.
func (ch *ConsistentHashing) AddNode(nodeName string) {
	if _, exists := ch.nodes[nodeName]; exists {
		fmt.Printf("! Node '%s' already exists.\n", nodeName)
		return
	}

	fmt.Printf("\n✨ Adding node '%s' and redistributing data...\n", nodeName)

	// 1. Add the new node and its VNodes to the ring first.
	// This updates the state so that GetNode works correctly for redistribution.
	ch.nodes[nodeName] = make(map[string]string)
	for i := 0; i < ch.vnodes; i++ {
		vnodeKey := fmt.Sprintf("%s#%d", nodeName, i)
		hash := hashKey(vnodeKey)
		ch.ring = append(ch.ring, hash)
		ch.hashMap[hash] = nodeName
	}
	sort.Slice(ch.ring, func(i, j int) bool { return ch.ring[i] < ch.ring[j] })

	// 2. Find and move the data that now belongs to the new node.
	keysMoved := 0
	movesBySource := make(map[string]int)

	// To avoid modifying maps during iteration, we first identify
	// all the keys to be moved.
	keysToMove := make(map[string][]string) // Map of: sourceNode -> [keys]

	for sourceNode, data := range ch.nodes {
		if sourceNode == nodeName {
			continue
		}
		for key := range data {
			targetNode, _ := ch.GetNode(key)
			if targetNode == nodeName {
				keysToMove[sourceNode] = append(keysToMove[sourceNode], key)
			}
		}
	}

	// Now, we actually move the keys.
	for sourceNode, keys := range keysToMove {
		for _, key := range keys {
			value := ch.nodes[sourceNode][key]
			ch.nodes[nodeName][key] = value
			delete(ch.nodes[sourceNode], key)
			movesBySource[sourceNode]++
			keysMoved++
		}
	}

	fmt.Printf("✅ %d records were moved to the new node '%s'.\n", keysMoved, nodeName)
	if len(movesBySource) > 0 {
		for sourceNode, count := range movesBySource {
			fmt.Printf("  -> From '%s': %d records\n", sourceNode, count)
		}
	}
}

// RemoveNode removes a node and redistributes its data to other nodes.
func (ch *ConsistentHashing) RemoveNode(nodeName string) error {
	if _, exists := ch.nodes[nodeName]; !exists {
		return fmt.Errorf("node '%s' not found", nodeName)
	}

	fmt.Printf("\nRemoving node '%s' and redistributing its data...\n", nodeName)

	// 1. Save the data to be moved BEFORE changing the ring.
	dataToMove := ch.nodes[nodeName]

	// 2. Remove all VNodes from the ring.
	hashesToRemove := make(map[uint32]bool)
	for i := 0; i < ch.vnodes; i++ {
		vnodeKey := fmt.Sprintf("%s#%d", nodeName, i)
		hash := hashKey(vnodeKey)
		hashesToRemove[hash] = true
		delete(ch.hashMap, hash)
	}
	newRing := make([]uint32, 0, len(ch.ring))
	for _, hash := range ch.ring {
		if !hashesToRemove[hash] {
			newRing = append(newRing, hash)
		}
	}
	ch.ring = newRing

	// 3. Delete the node from the storage map. The data map is still in 'dataToMove'.
	delete(ch.nodes, nodeName)

	// 4. Redistribute the data to their new destination nodes.
	movesByDest := make(map[string]int)
	for key, value := range dataToMove {
		newNode, _ := ch.GetNode(key)
		ch.nodes[newNode][key] = value
		movesByDest[newNode]++
	}

	fmt.Printf("✅ %d records were moved from node '%s'.\n", len(dataToMove), nodeName)
	if len(movesByDest) > 0 {
		for destNode, count := range movesByDest {
			fmt.Printf("  -> To '%s': %d records\n", destNode, count)
		}
	}
	return nil
}

func (ch *ConsistentHashing) printNodeStats() {
	fmt.Println("\n--- Current Node Status ---")
	total := 0
	nodeNames := make([]string, 0, len(ch.nodes))
	for name := range ch.nodes {
		nodeNames = append(nodeNames, name)
	}
	sort.Strings(nodeNames)

	for _, name := range nodeNames {
		count := len(ch.nodes[name])
		fmt.Printf("Node %-8s: %d records\n", name, count)
		total += count
	}
	fmt.Printf("----------------------------\n")
	fmt.Printf("Total Records: %d\n", total)
	fmt.Printf("----------------------------\n")
}

func verifyKeys(ch *ConsistentHashing, users map[string]string) {
	fmt.Println("\n🔎 Verifying the location of all keys...")
	
	correct := 0
	incorrect := 0
	
	actualLocations := make(map[string]string)
	for nodeName, data := range ch.nodes {
		for key := range data {
			actualLocations[key] = nodeName
		}
	}

	for key := range users {
		expectedNode, _ := ch.GetNode(key)
		actualNode, found := actualLocations[key]

		if !found {
			incorrect++
			fmt.Printf("  -> FATAL ERROR! Key '%s' was LOST and not found on any node.\n", key)
		} else if expectedNode == actualNode {
			correct++
		} else {
			incorrect++
			fmt.Printf("  -> Error! Key '%s' should be on '%s', but is on '%s'.\n", key, expectedNode, actualNode)
		}
	}
	
	fmt.Printf("----------------------------\n")
	fmt.Printf("Verification Complete: %d correct keys, %d incorrect keys.\n", correct, incorrect)
	fmt.Printf("----------------------------\n")
}

func main() {
	const numUsers = 1000000
	const initialNodes = 10
	const numVNodes = 1000

	fmt.Printf("📝 Creating %d user records...\n", numUsers)
	users := make(map[string]string)
	for i := 0; i < numUsers; i++ {
		key := "user_" + strconv.Itoa(i)
		users[key] = "data_for_" + key
	}

	ch := NewConsistentHashing(numVNodes)

	fmt.Printf("⚙️  Adding %d initial nodes to the ring (with %d VNodes each)...\n", initialNodes, numVNodes)
	for i := 0; i < initialNodes; i++ {
		nodeName := "node-" + strconv.Itoa(i)
		ch.nodes[nodeName] = make(map[string]string)
		for j := 0; j < ch.vnodes; j++ {
			vnodeKey := fmt.Sprintf("%s#%d", nodeName, j)
			hash := hashKey(vnodeKey)
			ch.ring = append(ch.ring, hash)
			ch.hashMap[hash] = nodeName
		}
	}
	sort.Slice(ch.ring, func(i, j int) bool { return ch.ring[i] < ch.ring[j] })
	fmt.Println("Nodes added.")

	fmt.Println("\n🗺️  Distributing initial records to nodes...")
	for key, value := range users {
		node, _ := ch.GetNode(key)
		ch.nodes[node][key] = value
	}
	ch.printNodeStats()

	ch.RemoveNode("node-4")
	ch.printNodeStats()

	ch.AddNode("node-10")
	ch.printNodeStats()

	verifyKeys(ch, users)
}