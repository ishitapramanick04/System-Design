package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

var ctx = context.Background()

type DistributedLock struct {
	client     *redis.Client
	key        string
	identifier string
	ttl        time.Duration
}

func NewLock(client *redis.Client, resource string, ttl time.Duration) *DistributedLock {
	return &DistributedLock{
		client:     client,
		key:        fmt.Sprintf("lock:%s", resource),
		identifier: uuid.New().String(),
		ttl:        ttl,
	}
}

func (l *DistributedLock) Acquire() (bool, error) {
	return l.client.SetNX(ctx, l.key, l.identifier, l.ttl).Result()
}

func (l *DistributedLock) AcquireWithRetry(retries int, delay time.Duration) (bool, error) {
	for i := 0; i < retries; i++ {
		acquired, err := l.Acquire()
		if err != nil {
			return false, err
		}
		if acquired {
			return true, nil
		}
		fmt.Printf("    [%s] waiting for [%s] - Lock busy, retrying... (%d/%d)\n", l.identifier[:8], l.key, i+1, retries)
		time.Sleep(delay)
	}
	return false, nil
}

func (l *DistributedLock) Release() error {
	script := redis.NewScript(`
        if redis.call("GET", KEYS[1]) == ARGV[1] then
            return redis.call("DEL", KEYS[1])
        else
            return 0
        end
    `)
	_, err := script.Run(ctx, l.client, []string{l.key}, l.identifier).Result()
	return err
}

// Shared resources
var (
	inventoryCount int // Resource 1: Inventory
	orderCount     int // Resource 2: Orders
)

func inventoryWorker(id int, rdb *redis.Client, wg *sync.WaitGroup, startGate <-chan struct{}) {
	defer wg.Done()

	// Wait for start signal (all workers start simultaneously)
	<-startGate

	lock := NewLock(rdb, "inventory", 10*time.Second)

	fmt.Printf("📦 Inventory Worker %d: Trying to acquire lock [%s] with ID [%s]\n", id, lock.key, lock.identifier[:8])

	acquired, err := lock.AcquireWithRetry(15, 500*time.Millisecond)
	if err != nil {
		fmt.Printf("📦 Inventory Worker %d: Error: %v\n", id, err)
		return
	}

	if !acquired {
		fmt.Printf("📦 Inventory Worker %d: ❌ Could not acquire lock [%s]\n", id, lock.key)
		return
	}

	fmt.Printf("📦 Inventory Worker %d: ✅ Lock acquired! [%s] owned by [%s]\n", id, lock.key, lock.identifier[:8])

	// === CRITICAL SECTION ===
	current := inventoryCount
	fmt.Printf("📦 Inventory Worker %d: Read inventory = %d\n", id, current)
	time.Sleep(2 * time.Second) // Simulate processing
	inventoryCount = current + 10
	fmt.Printf("📦 Inventory Worker %d: Updated inventory to %d\n", id, inventoryCount)
	// === END CRITICAL SECTION ===

	lock.Release()
	fmt.Printf("📦 Inventory Worker %d: 🔓 Lock released [%s] by [%s]\n", id, lock.key, lock.identifier[:8])
}

func orderWorker(id int, rdb *redis.Client, wg *sync.WaitGroup, startGate <-chan struct{}) {
	defer wg.Done()

	// Wait for start signal (all workers start simultaneously)
	<-startGate

	lock := NewLock(rdb, "orders", 10*time.Second)

	fmt.Printf("🛒 Order Worker %d: Trying to acquire lock [%s] with ID [%s]\n", id, lock.key, lock.identifier[:8])

	acquired, err := lock.AcquireWithRetry(15, 500*time.Millisecond)
	if err != nil {
		fmt.Printf("🛒 Order Worker %d: Error: %v\n", id, err)
		return
	}

	if !acquired {
		fmt.Printf("🛒 Order Worker %d: ❌ Could not acquire lock [%s]\n", id, lock.key)
		return
	}

	fmt.Printf("🛒 Order Worker %d: ✅ Lock acquired! [%s] owned by [%s]\n", id, lock.key, lock.identifier[:8])

	// === CRITICAL SECTION ===
	current := orderCount
	fmt.Printf("🛒 Order Worker %d: Read orders = %d\n", id, current)
	time.Sleep(2 * time.Second) // Simulate processing
	orderCount = current + 1
	fmt.Printf("🛒 Order Worker %d: Updated orders to %d\n", id, orderCount)
	// === END CRITICAL SECTION ===

	lock.Release()
	fmt.Printf("🛒 Order Worker %d: 🔓 Lock released [%s] by [%s]\n", id, lock.key, lock.identifier[:8])
}

func main() {
	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	// Test connection
	_, err := rdb.Ping(ctx).Result()
	if err != nil {
		fmt.Println("Could not connect to Redis:", err)
		return
	}

	// Clear any existing locks
	rdb.Del(ctx, "lock:inventory", "lock:orders")

	// Reset counters
	inventoryCount = 0
	orderCount = 0

	fmt.Println("================================================================")
	fmt.Println("  Distributed Lock Demo: 2 Resources, 5 Workers")
	fmt.Println("  📦 Inventory Lock: 3 workers competing")
	fmt.Println("  🛒 Orders Lock: 2 workers competing")
	fmt.Println("================================================================\n")

	var wg sync.WaitGroup
	startGate := make(chan struct{}) // All workers wait for this signal

	// 3 workers competing for INVENTORY lock
	for i := 1; i <= 3; i++ {
		wg.Add(1)
		go inventoryWorker(i, rdb, &wg, startGate)
	}

	// 2 workers competing for ORDERS lock
	for i := 4; i <= 5; i++ {
		wg.Add(1)
		go orderWorker(i, rdb, &wg, startGate)
	}

	// Release all workers at once!
	fmt.Println("🚦 Starting all workers simultaneously...")
	close(startGate)

	wg.Wait()

	fmt.Println("\n================================================================")
	fmt.Println("  RESULTS")
	fmt.Println("================================================================")
	fmt.Printf("  📦 Final Inventory: %d (expected: 30 = 3 workers × 10)\n", inventoryCount)
	fmt.Printf("  🛒 Final Orders:    %d (expected: 2 = 2 workers × 1)\n", orderCount)
	fmt.Println("================================================================")
}
