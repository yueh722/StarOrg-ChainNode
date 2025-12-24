package utils

import (
	"errors"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func SqliteOpen(dbFilePath string) (gorm.Dialector, error) {
	dbFilePath = strings.TrimSpace(dbFilePath)
	if dbFilePath == "" {
		return nil, errors.New("dbFilePath is empty")
	}

	// 1. convert abs file path if need to
	absPath, err := filepath.Abs(dbFilePath)
	if err != nil {
		return nil, err
	}

	// 2. Windows file path：\ -> /
	//    C:\data\master.db -> C:/data/master.db
	unixStylePath := filepath.ToSlash(absPath)

	// 3. URL encode
	escapedPath := url.PathEscape(unixStylePath)

	// 4. make DSN
	dsn := "file:" + escapedPath +
		"?_pragma=busy_timeout(5000)" +
		"&_pragma=journal_mode(WAL)" +
		"&_pragma=foreign_keys(ON)"

	conn := sqlite.Open(dsn)

	return conn, nil
}
