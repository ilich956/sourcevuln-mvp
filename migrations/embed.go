package migrations

import "embed"

//go:embed *.up.sql
var UpFiles embed.FS
