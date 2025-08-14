package config

import (
	"fmt"
	"sort"
	"strings"
)

type Database struct {
	DatabaseName string
	Host         string
	Port         int
	Username     string
	Password     string
	Option       map[string]string
}

// ConnectionString generates a connection string to be passed to sql.Open or equivalents, assuming Postgres syntax
func (d *Database) ConnectionString() string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s", d.Host, d.Port, d.Username, d.Password, d.DatabaseName))

	if _, ok := d.Option["sslmode"]; !ok {
		b.WriteString(" sslmode=disable")
	}

	if len(d.Option) > 0 {
		options := make([]string, 0, len(d.Option))
		for option := range d.Option {
			options = append(options, option)
		}

		sort.Strings(options)

		for _, option := range options {
			fmt.Fprintf(&b, " %s=%s", option, d.Option[option])
		}
	}

	return b.String()
}
