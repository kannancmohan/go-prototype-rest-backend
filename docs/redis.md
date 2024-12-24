


### Redis data type


| **Data Type**       | **Short Description**                                  | **Storage Command**                              | **Retrieval Command**                         |
|----------------------|-------------------------------------------------------|-------------------------------------------------|-----------------------------------------------|
| **String**           | Binary-safe, holds any type of data.                  | `SET key value`                                 | `GET key`                                     |
| **Hash**             | Collection of field-value pairs (like a dictionary).  | `HSET key field value`                          | `HGET key field`                              |
|                      |                                                       | `HMSET key field1 value1 field2 value2`         | `HGETALL key`                                 |
| **List**             | Ordered list of strings.                              | `LPUSH key value` (left/head)                   | `LRANGE key start stop`                       |
|                      |                                                       | `RPUSH key value` (right/tail)                  | `LPOP key` or `RPOP key`                      |
| **Set**              | Unordered collection of unique strings.               | `SADD key value`                                | `SMEMBERS key`                                |
|                      |                                                       |                                                 | `SISMEMBER key value`                         |
| **Sorted Set (Zset)**| Set with scores, sorted by score.                      | `ZADD key score member`                         | `ZRANGE key start stop`                       |
|                      |                                                       |                                                 | `ZRANGEBYSCORE key min max`                   |
| **Bitmap**           | Bit-level manipulation of a string.                   | `SETBIT key offset value`                       | `GETBIT key offset`                           |
| **HyperLogLog**      | Probabilistic structure for cardinality estimation.   | `PFADD key element`                             | `PFCOUNT key`                                 |
| **Streams**          | Log-like structure for events and messages.           | `XADD key * field1 value1 field2 value2`        | `XRANGE key start end`                        |
|                      |                                                       |                                                 | `XREAD COUNT count STREAMS key start`         |
| **Geospatial**       | Geospatial indexes for storing locations.             | `GEOADD key longitude latitude member`          | `GEORADIUS key longitude latitude radius unit`|
|                      |                                                       |                                                 | `GEOPOS key member`                           |
| **JSON** (RedisJSON) | Stores and retrieves JSON documents.                  | `JSON.SET key $ 'json_value'`                   | `JSON.GET key`                                |
