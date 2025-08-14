package main

//go:generate sqlboiler --wipe psql

import (
	"github.com/Abdurrochman25/multi-tenant-messaging-system/internal"
	_ "github.com/lib/pq"
)

func main() {
	internal.Init()
}
