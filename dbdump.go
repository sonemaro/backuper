package backuper

import (
	"database/sql"
	"fmt"

	"github.com/JamesStewy/go-mysqldump"
	_ "github.com/go-sql-driver/mysql"
)

// DBDump is responsible for dumping database
type DBDump struct {
	Username string
	Password string
	Host     string
	Port     int
	DBName   string
}

// NewDBDump returns an instance of DBDump
func NewDBDump(username, password, host, dbname string, port int) DBDump {
	return DBDump{
		Username: username,
		Password: password,
		Host:     host,
		DBName:   dbname,
		Port:     port,
	}
}

// Dump dumps the database to path and returns the dump name
// Returns ("", error) if can't create a successful dump
func (d *DBDump) Dump(path string) (string, error) {
	db, err := sql.Open("mysql", d.createURI())
	if err != nil {
		return "", err
	}

	dumpFilenameFormat := fmt.Sprintf("%s-20060102T150405", d.DBName)
	dumper, err := mysqldump.Register(db, path, dumpFilenameFormat)
	if err != nil {
		return "", err
	}

	// Dump database to file
	resultFilename, err := dumper.Dump()
	if err != nil {
		return "", err
	}

	dumper.Close()
	return resultFilename, nil
}

// createURI creates a sql connection URI
func (d *DBDump) createURI() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s", d.Username, d.Password, d.Host, d.Port, d.DBName)
}
