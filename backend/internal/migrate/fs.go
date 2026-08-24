package migrate

import "embed"

//go:embed sql
var Files embed.FS

const Dir = "sql"
