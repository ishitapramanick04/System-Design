# When to Use Redis Pub/Sub over Kafka or Other Brokers

Use Redis Pub/Sub when **all** of the following are true.

---

## 1. You Need Persistent TCP Connections

Redis maintains long-lived TCP connections.
Subscribers stay connected. Messages are pushed, not polled.
This removes reconnect churn and eliminates consumer lag mechanics.

Kafka consumers also keep TCP connections, but they **poll**. Redis Pub/Sub is true push.

This makes Redis ideal for:

* WebSockets
* Live dashboards
* Presence systems
* Real-time notifications

---

## 2. Message Loss is Acceptable

Redis Pub/Sub has:

* No persistence
* No replay
* No acknowledgements

If a subscriber disconnects, messages during that window are gone.
If this is unacceptable, Redis Pub/Sub is the wrong tool.

---

## 3. You Care About Latency More Than Guarantees

Redis Pub/Sub delivers messages in microseconds.
Kafka optimizes for throughput and durability, not immediacy.

Use Redis when:

* Humans are waiting
* UI must update instantly
* Cache invalidation must propagate immediately

---

## 4. Fan-out is the Dominant Pattern

Redis is excellent at:

* One → many delivery
* Broadcasting state changes
* Signaling events

Kafka excels at:

* Durable event logs
* Ordered replay
* Independent consumer progress

If consumers are observers rather than processors, Redis wins.

---

## 5. You Want Minimal Infrastructure

**Redis:**

* Single process
* Single port
* No partitions
* No rebalancing
* No coordinators

**Kafka:**

* Brokers
* Partitions
* Consumer groups
* Rebalancing
* Operational overhead

If the system must be explainable in one diagram, Redis fits.

---

## 6. Message Semantics are Simple

**Redis Pub/Sub guarantees:**

* Best-effort delivery
* In-order per connection

**Kafka guarantees:**

* Durable ordering
* At-least-once / exactly-once semantics

If messages are signals, not records, Redis is sufficient.

---

## When NOT to Use Redis Pub/Sub

Do not use Redis Pub/Sub when:

* Messages must survive crashes
* Consumers may be offline
* You need replay or backfill
* You need auditability
* You need exactly-once processing

**That is Kafka territory.**

---

## Practical Rule (The Real Takeaway)

> **Redis Pub/Sub** is for **state change notification**.
> **Kafka** is for **state change history**.

| | Redis Pub/Sub | Kafka |
|---|---|---|
| Answers | *"Something just happened."* | *"Everything that has ever happened."* |

---

## Final Summary

Redis Pub/Sub advantages explicitly include:

* ✅ Persistent TCP connections
* ✅ Server-push delivery
* ✅ No polling
* ✅ Extremely low latency

Those are core differentiators, not footnotes.

> 🧠 **Use Redis Pub/Sub as a nervous system.**
> 📚 **Use Kafka as a memory.**
