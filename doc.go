// Package manta parses Dota 2 Source 2 replays into match statistics.
//
// ParseFile reads a replay from a path and ParseReader reads it from a stream.
// Both accept .dem, .dem.bz2 and .dem.zst data; compression is detected from
// the input rather than the file name. The returned [Match] is the odota-cli
// match object.
//
// Calls are independent and safe for concurrent use. The decoder reads the
// stream incrementally and does not retain the replay.
package manta
