package rootmulti

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	sqlite "github.com/mattn/go-sqlite3"
	dbm "github.com/tendermint/tm-db"
)

const GetQuery = `
     SELECT value
       FROM %s
      WHERE key = ? AND
            height <= ? AND
            (NOT deleted_at <= ? OR deleted_at is null)
   GROUP BY height
     HAVING height = MAX(height)
   ORDER BY height DESC
      LIMIT 1`

const InsertStatement = `
	INSERT INTO %s(height, key, value, deleted_at)
	SELECT ?, ?, ?, NULL
	 WHERE ? IS NOT
					(
					 SELECT value
					   FROM %s
				      WHERE key = ? AND
							height <= ? AND
						    (NOT deleted_at <= ? OR deleted_at is null)
                   GROUP BY height
    		         HAVING height = MAX(height)
                   ORDER BY height DESC
                      LIMIT 1
				   )
`

const DeleteStatement = `
   DELETE FROM %s
    WHERE EXISTS (
					 SELECT 1
					   FROM %s
				      WHERE key = ? AND
							height <= ? AND
						    (NOT deleted_at <= ? OR deleted_at is null)
                   GROUP BY height
    		         HAVING height = MAX(height)
                   ORDER BY height DESC
                      LIMIT 1
    )
`

type AppStateDB struct {
	sqlDB    *sql.DB
	tableMap map[string]bool
}

func NewAppStateDB(name string, dir string) (*AppStateDB, error) {
	// Find a way to not have to do this
	fmt.Println(sqlite.SQLITE_OK)
	database, dbError := sql.Open("sqlite", fmt.Sprintf("%s/%s.db", dir, name))
	if dbError != nil {
		return nil, dbError
	}
	return &AppStateDB{
		sqlDB:    database,
		tableMap: map[string]bool{},
	}, nil
}

func (asdb *AppStateDB) createTableIfNotExist(table string) error {
	//if asdb.tableMap[table] != true {
	//	//tableCreationError := asdb.createTableIfNotExist(table)
	//	if tableCreationError != nil {
	//		panic(tableCreationError)
	//	}
	//}
	return errors.New("IMPLEMENT THIS")
}

func (asdb *AppStateDB) Has(height int64, table string, key string) (bool, error) {
	// First make sure table exists
	tableExistsErr := asdb.createTableIfNotExist(table)
	if tableExistsErr != nil {
		return false, tableExistsErr
	}

	// Prepare the query
	queryStmt, queryStmtErr := asdb.sqlDB.Prepare(fmt.Sprintf(GetQuery, table))
	if queryStmtErr != nil {
		return false, queryStmtErr
	}
	defer queryStmt.Close()

	// Execute the query
	var result string
	queryError := queryStmt.QueryRow(key, height, height).Scan(&result)
	if queryError != nil {
		return false, queryError
	}
	return true, nil
}

func (asdb *AppStateDB) Get(height int64, table string, key string) ([]byte, error) {
	// First make sure table exists
	tableExistsErr := asdb.createTableIfNotExist(table)
	if tableExistsErr != nil {
		return nil, tableExistsErr
	}

	// Prepare the query
	queryStmt, queryStmtErr := asdb.sqlDB.Prepare(fmt.Sprintf(GetQuery, table))
	if queryStmtErr != nil {
		return nil, queryStmtErr
	}
	defer queryStmt.Close()

	// Execute the query
	var result string
	queryError := queryStmt.QueryRow(key, height, height).Scan(&result)
	if queryError != nil {
		return nil, queryError
	}
	return []byte(result), nil
}

func (asdb *AppStateDB) Set(height int64, table string, key string, value string) error {
	// First make sure table exists
	tableExistsErr := asdb.createTableIfNotExist(table)
	if tableExistsErr != nil {
		return tableExistsErr
	}

	// Prepare the insert statement
	insertStmt, insertStmtErr := asdb.sqlDB.Prepare(fmt.Sprintf(InsertStatement, table, table))
	if insertStmtErr != nil {
		return insertStmtErr
	}
	defer insertStmt.Close()

	// Execute the insert statement
	_, insertExecErr := insertStmt.Exec(height, key, value, key, height, height)
	if insertExecErr != nil {
		return insertExecErr
	}

	// Success!
	return nil
}

func (asdb *AppStateDB) Delete(height int64, table string, key string) error {
	// First make sure table exists
	tableExistsErr := asdb.createTableIfNotExist(table)
	if tableExistsErr != nil {
		return tableExistsErr
	}

	// Prepare the delete statement
	deleteStmt, deleteStmtErr := asdb.sqlDB.Prepare(fmt.Sprintf(DeleteStatement, table, table))
	if deleteStmtErr != nil {
		return deleteStmtErr
	}
	defer deleteStmt.Close()

	// Execute the delete statement
	_, deleteExecErr := deleteStmt.Exec(key, height, height)
	if deleteExecErr != nil {
		return deleteExecErr
	}

	// Success!
	return nil
}

// Mutable App State DB
type MutableAppStateDB struct {
	cancel   context.CancelFunc
	tx       *sql.Tx
	isTxDone bool
}

func NewMutableAppStateDB(asdb *AppStateDB) (*MutableAppStateDB, error) {
	ctx, cancel := context.WithCancel(context.Background())
	// TODO: understand txoptions
	tx, beginTxError := asdb.sqlDB.BeginTx(ctx, nil)
	if beginTxError != nil {
		return nil, beginTxError
	}
	return &MutableAppStateDB{
		cancel:   cancel,
		tx:       tx,
		isTxDone: false,
	}, nil
}

func (masdb *MutableAppStateDB) createTableIfNotExist(table string) error {
	//if asdb.tableMap[table] != true {
	//	//tableCreationError := asdb.createTableIfNotExist(table)
	//	if tableCreationError != nil {
	//		panic(tableCreationError)
	//	}
	//}
	return errors.New("IMPLEMENT THIS")
}

func (masdb *MutableAppStateDB) Has(height int64, table string, key string) (bool, error) {
	// First make sure table exists
	tableExistsErr := masdb.createTableIfNotExist(table)
	if tableExistsErr != nil {
		return false, tableExistsErr
	}

	// Prepare the query
	queryStmt, queryStmtErr := masdb.tx.Prepare(fmt.Sprintf(GetQuery, table))
	if queryStmtErr != nil {
		return false, queryStmtErr
	}
	defer queryStmt.Close()

	// Execute the query
	var result string
	queryError := queryStmt.QueryRow(key, height, height).Scan(&result)
	if queryError != nil {
		return false, queryError
	}
	return true, nil
}

func (masdb *MutableAppStateDB) Get(height int64, table string, key string) ([]byte, error) {
	// First make sure table exists
	tableExistsErr := masdb.createTableIfNotExist(table)
	if tableExistsErr != nil {
		return nil, tableExistsErr
	}

	// Prepare the query
	queryStmt, queryStmtErr := masdb.tx.Prepare(fmt.Sprintf(GetQuery, table))
	if queryStmtErr != nil {
		return nil, queryStmtErr
	}
	defer queryStmt.Close()

	// Execute the query
	var result string
	queryError := queryStmt.QueryRow(key, height, height).Scan(&result)
	if queryError != nil {
		return nil, queryError
	}
	return []byte(result), nil
}

func (masdb *MutableAppStateDB) Set(height int64, table string, key string, value string) error {
	// First make sure table exists
	tableExistsErr := masdb.createTableIfNotExist(table)
	if tableExistsErr != nil {
		return tableExistsErr
	}

	// Prepare the insert statement
	insertStmt, insertStmtErr := masdb.tx.Prepare(fmt.Sprintf(InsertStatement, table, table))
	if insertStmtErr != nil {
		return insertStmtErr
	}
	defer insertStmt.Close()

	// Execute the insert statement
	_, insertExecErr := insertStmt.Exec(height, key, value, key, height, height)
	if insertExecErr != nil {
		return insertExecErr
	}

	// Success!
	return nil
}

func (masdb *MutableAppStateDB) Delete(height int64, table string, key string) error {
	// First make sure table exists
	tableExistsErr := masdb.createTableIfNotExist(table)
	if tableExistsErr != nil {
		return tableExistsErr
	}

	// Prepare the delete statement
	deleteStmt, deleteStmtErr := masdb.tx.Prepare(fmt.Sprintf(DeleteStatement, table, table))
	if deleteStmtErr != nil {
		return deleteStmtErr
	}
	defer deleteStmt.Close()

	// Execute the delete statement
	_, deleteExecErr := deleteStmt.Exec(key, height, height)
	if deleteExecErr != nil {
		return deleteExecErr
	}

	// Success!
	return nil
}

func (masdb *MutableAppStateDB) Close() {
	masdb.cancel()
}

func (masdb *MutableAppStateDB) Commit() error {
	return masdb.tx.Commit()
}

var _ dbm.Iterator = AppDBIterator{}

// Is 'height/storeKey' aware
type AppDBIterator struct {
	it dbm.Iterator
}

func (s AppDBIterator) Key() (key []byte) {
	return KeyFromStoreKey(s.it.Key())
}

func (s AppDBIterator) Valid() bool {
	return s.it.Valid()
}

func (s AppDBIterator) Next() {
	s.it.Next()
}

func (s AppDBIterator) Value() (value []byte) {
	return s.it.Value()
}

func (s AppDBIterator) Error() error {
	return s.it.Error()
}

func (s AppDBIterator) Close() {
	s.it.Close()
}

func (s AppDBIterator) Domain() (start []byte, end []byte) {
	st, end := s.it.Domain()
	return KeyFromStoreKey(st), KeyFromStoreKey(end)
}
