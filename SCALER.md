# Brixa Scaler

Network routing layer for the Brixa distributed system.

## Works With

**Brixa Node Engine** - Task execution layer

The Scaler and Node Engine communicate via a strict API contract.

## API Contract

### POST /task
Request: {"task_id": "uuid", "type": "dm|world|quest|ai", "payload": {}}
Response: {"task_id", "status", "result", "node_id", "execution_time_ms", "proof_hash"}

### GET /health - Check node is alive
### GET /capabilities - What tasks can node handle?
### GET /metrics - Node statistics
