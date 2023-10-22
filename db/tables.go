package db

// array of [2]string{SQL statement, context}
var setup = [][2]string{
	{sql_TABLE_IMAGES, "creating the img cache table"},
	{sql_INDEX_SUB,    "creating the sub index"},
	{sql_INDEX_NSFW,   "creating the nsfw index"},
}

const sql_TABLE_IMAGES = `CREATE TABLE IF NOT EXISTS images (
	post_id TEXT,
	img TEXT,
	nsfw BOOLEAN,
	sub TEXT,
	true_sub TEXT,
	width INTEGER,
	height INTEGER,
	created_at INTEGER
)`

const sql_REQ_STATS = `CREATE TABLE IF NOT EXISTS req_stats (
	sub TEXT,
	nsfw VARCHAR(2),
	requests INTEGER DEFAULT 0,

	UNIQUE(sub, nsfw)
)`

const sql_INDEX_SUB = `CREATE INDEX IF NOT EXISTS ind_sub ON images (sub)`
const sql_INDEX_NSFW = `CREATE INDEX IF NOT EXISTS ind_nsfw ON images (nsfw)`
