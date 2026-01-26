# Failure Modes

This document describes how DocDB handles various failure scenarios.

## Overview

DocDB is designed to survive `kill -9` crashes and maintain data integrity through its Write-Ahead Log (WAL) and append-only storage model.

## Crash Recovery

### Scenario: Process Crashes During Write

**Symptom:**
- Process receives SIGKILL (kill -9)
- WAL record may be partially written

**Detection:**
- CRC32 checksum failure on WAL replay
- Incomplete record length

**Recovery:**
1. Load catalog
2. Sequentially replay WAL from beginning
3. Validate CRC32 for each record
4. Stop at first corrupted record
5. Truncate WAL to last valid offset
6. Rebuild in-memory index from valid records
7. Continue serving

**Result:**
- All complete transactions before crash are recovered
- Partial transaction at crash is discarded (atomicity)

### Scenario: Disk Full During Write

**Symptom:**
- Write operations fail with ENOSPC
- System cannot allocate more space

**Detection:**
- `os.Write` returns error
- `os.Sync` returns error

**Recovery:**
- Return error to client immediately
- Do not mark transaction as committed
- WAL record is incomplete (will be truncated on recovery)

**Result:**
- No data corruption
- Transaction fails atomically
- Client receives clear error

### Scenario: Power Loss During Write

**Symptom:**
- System loses power mid-write
- File system may have partial writes

**Detection:**
- CRC32 checksum failure on WAL replay
- File system journal replay (if applicable)

**Recovery:**
- Same as `kill -9` recovery
- Rely on CRC32 to detect corruption
- Truncate at first error

**Result:**
- Same as crash recovery
- Data integrity maintained

## WAL Corruption

### Scenario: Corrupted WAL Record

**Symptom:**
- CRC32 mismatch on WAL record
- Record length invalid

**Detection:**
- During WAL replay on startup
- CRC32 validation fails

**Recovery:**
1. Process valid records up to corruption
2. Stop at corrupted record
3. Truncate WAL file to last valid offset
4. Continue with recovered state

**Result:**
- All data before corruption is preserved
- Corrupted transaction is discarded
- System continues operating

### Scenario: Truncated WAL File

**Symptom:**
- WAL file ends mid-record
- Incomplete record at end

**Detection:**
- EOF encountered during record read
- Not enough bytes for complete record

**Recovery:**
1. Process all complete records
2. Stop at truncated record
3. WAL is already truncated (no action needed)

**Result:**
- All complete transactions recovered
- Partial transaction discarded

### Scenario: Missing WAL File

**Symptom:**
- WAL file does not exist on startup
- Only data file present

**Detection:**
- File not found error on WAL open

**Recovery:**
1. Load catalog
2. Create new WAL file
3. Data file remains intact
4. Continue with existing data

**Result:**
- Existing documents remain accessible
- WAL resumes fresh

## Data File Corruption

### Scenario: Corrupted Data Record

**Symptom:**
- Read from data file returns corrupted data
- Payload length invalid

**Detection:**
- Mismatch between stored and expected payload length
- CRC32 (if added) failure

**Recovery:**
- This is **NOT** handled in v0
- May result in returning corrupted data to client

**Mitigation:**
- Use filesystem with journaling (ext4, APFS, etc.)
- Regular backups
- Compaction can rewrite clean data

**Future Enhancement:**
- Add CRC32 to data file records
- Compaction validates all records

### Scenario: Missing Data File

**Symptom:**
- Data file does not exist for a database
- WAL references documents not in data file

**Detection:**
- `os.Open` fails on data file

**Recovery:**
- Database cannot be opened
- Return error to client

**Result:**
- Database is inaccessible
- User must restore from backup

## Memory Exhaustion

### Scenario: Global Memory Limit Reached

**Symptom:**
- Global memory capacity exceeded
- New allocations fail

**Detection:**
- `MemoryCaps.TryAllocate` returns false
- Before writing to data file

**Recovery:**
- Return `ErrMemoryLimit` to client
- Transaction fails atomically
- No data written

**Result:**
- No corruption
- Clear error to client
- Server continues operating

### Scenario: Per-DB Memory Limit Reached

**Symptom:**
- Database-specific memory limit exceeded
- Cannot allocate for this database

**Detection:**
- Same as global limit
- Checked per-DB

**Recovery:**
- Same as global limit
- Other databases unaffected

**Result:**
- Failing database limited
- Other databases continue operating

## Compaction Failures

### Scenario: Crash During Compaction

**Symptom:**
- Process dies while writing `.compact` file
- Both old and compact files incomplete

**Detection:**
- `.compact` file present on restart
- Old file may have been unlinked

**Recovery:**
1. Detect `.compact` file exists
2. Delete `.compact` file
3. Continue using old data file
4. Retry compaction later

**Result:**
- No data loss
- Compaction simply failed
- Will succeed next time

### Scenario: Disk Full During Compaction

**Symptom:**
- Cannot write to `.compact` file
- ENOSPC error

**Detection:**
- Write operation fails
- Before atomic rename

**Recovery:**
- Delete incomplete `.compact` file
- Continue using old data file
- Return error (or log and retry)

**Result:**
- No corruption
- Compaction failed atomically

## Network Failures (IPC)

### Scenario: Client Disconnects During Request

**Symptom:**
- TCP/Unix connection closed
- Server cannot send response

**Detection:**
- `conn.Write` returns error
- Connection broken

**Recovery:**
- Transaction is rolled back
- No WAL record written
- No state changes

**Result:**
- Operation failed atomically
- No corruption
- Client can retry

### Scenario: Server Crashes During IPC Exchange

**Symptom:**
- Server dies mid-request
- Client connection closed

**Detection:**
- Client receives EOF/error
- Server crash recovery runs

**Recovery:**
- Same as crash recovery
- WAL replay handles partial writes

**Result:**
- Same as crash recovery
- Client receives connection error

## Known Limitations

### Data File Corruption (v0)

**Issue:**
- No checksums on data file records
- Corruption may go undetected

**Impact:**
- May return corrupted data to clients
- Silent data corruption possible

**Mitigation:**
- Use filesystem with journaling
- Regular backups
- Compaction (when implemented) validates data

**Future Fix:**
- Add CRC32 to data file records
- Validate on every read

### No WAL Rotation (v0)

**Issue:**
- Single WAL file grows indefinitely
- No automatic rotation

**Impact:**
- Large WAL files
- Slow recovery

**Mitigation:**
- Compaction can trigger WAL reset
- Manual WAL rotation

**Future Fix:**
- Automatic WAL rotation based on size
- Multiple WAL files with checkpoints

### No Write-Ahead Trimming (v0)

**Issue:**
- WAL contains all historical records
- Even after compaction

**Impact:**
- Larger WAL than necessary
- Slower recovery

**Mitigation:**
- Manual WAL deletion after verified backup

**Future Fix:**
- Automatic WAL trimming after checkpoints
- Periodic checkpoints

## Error Codes

| Code | Error                    | Meaning |
|------|--------------------------|---------|
| 0    | OK                       | Success |
| 1    | Error                    | General error |
| 2    | NotFound                 | Document/database not found |
| 3    | Conflict                 | Document already exists |
| 4    | MemoryLimit              | Memory limit exceeded |
| 5    | CorruptRecord            | WAL record corrupted |
| 6    | CRCMismatch              | CRC32 validation failed |
| 7    | FileOpen                 | Cannot open file |
| 8    | FileWrite                | Cannot write file |
| 9    | FileSync                 | Cannot sync file |
| 10   | FileRead                 | Cannot read file |

## Testing Failure Modes

Run failure mode tests:

```bash
go test ./tests/failure
```

Tests cover:
- Corrupted WAL records
- Truncated WAL files
- Missing WAL files
- Partial writes
- Crash recovery
