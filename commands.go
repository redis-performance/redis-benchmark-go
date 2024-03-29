package main

var readOnlyCommands = map[string]bool{
	"LLEN":                 true,
	"DUMP":                 true,
	"PTTL":                 true,
	"GEOHASH":              true,
	"HEXISTS":              true,
	"GROUPS":               true,
	"GET":                  true,
	"SSCAN":                true,
	"TOUCH":                true,
	"CONSUMERS":            true,
	"BITCOUNT":             true,
	"MGET":                 true,
	"SUBSTR":               true,
	"HVALS":                true,
	"TTL":                  true,
	"FREQ":                 true,
	"XREAD":                true,
	"PFCOUNT":              true,
	"ZREVRANGEBYSCORE":     true,
	"HMGET":                true,
	"LINDEX":               true,
	"XLEN":                 true,
	"GEODIST":              true,
	"ZLEXCOUNT":            true,
	"LPOS":                 true,
	"ZREVRANK":             true,
	"GEORADIUS_RO":         true,
	"GETRANGE":             true,
	"STRLEN":               true,
	"REFCOUNT":             true,
	"IDLETIME":             true,
	"SCARD":                true,
	"EXPIRETIME":           true,
	"ZRANGE":               true,
	"LOLWUT":               true,
	"SINTER":               true,
	"XREVRANGE":            true,
	"ZRANGEBYSCORE":        true,
	"SUNION":               true,
	"GETBIT":               true,
	"BITFIELD_RO":          true,
	"ZDIFF":                true,
	"HGETALL":              true,
	"ZSCORE":               true,
	"ZREVRANGE":            true,
	"ZREVRANGEBYLEX":       true,
	"SINTERCARD":           true,
	"SMISMEMBER":           true,
	"KEYS":                 true,
	"LCS":                  true,
	"SORT_RO":              true,
	"ZRANK":                true,
	"HSTRLEN":              true,
	"LRANGE":               true,
	"PEXPIRETIME":          true,
	"RANDOMKEY":            true,
	"SMEMBERS":             true,
	"ZMSCORE":              true,
	"HKEYS":                true,
	"ZCOUNT":               true,
	"ZSCAN":                true,
	"GEOSEARCH":            true,
	"ZUNION":               true,
	"XPENDING":             true,
	"HRANDFIELD":           true,
	"HLEN":                 true,
	"ZRANDMEMBER":          true,
	"DBSIZE":               true,
	"SDIFF":                true,
	"STREAM":               true,
	"ZINTER":               true,
	"SCAN":                 true,
	"TYPE":                 true,
	"USAGE":                true,
	"ZINTERCARD":           true,
	"SISMEMBER":            true,
	"HGET":                 true,
	"SRANDMEMBER":          true,
	"ENCODING":             true,
	"EXISTS":               true,
	"GEOPOS":               true,
	"XRANGE":               true,
	"ZRANGEBYLEX":          true,
	"GEORADIUSBYMEMBER_RO": true,
	"BITPOS":               true,
	"ZCARD":                true,
	"HSCAN":                true,
}
