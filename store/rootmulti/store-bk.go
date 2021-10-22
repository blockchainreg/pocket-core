//package rootmulti
//
//import (
//	"database/sql"
//	"fmt"
//	"github.com/mattn/go-sqlite3"
//	"github.com/pokt-network/pocket-core/store/cachemulti"
//	"github.com/pokt-network/pocket-core/store/iavl"
//	"github.com/pokt-network/pocket-core/store/types"
//	dbm "github.com/tendermint/tm-db"
//	"strconv"
//	"strings"
//)
//
//// Prefixed abstractions living inside AppDB;
//type Store struct {
//	deletionStmts []*sql.Stmt
//	sqlDB         *sql.DB
//	appDB         dbm.DB      // parent db, (where everything lives except for state)
//	//state         dbm.DB      // ephemeral state used to 'stage' potential changes; only latest height; nuked on startup;
//	iavl          *iavl.Store // used for latest height state commitments ONLY; may be pruned;
//	storeKey      string      // constant; part of the prefix
//	height        int64       // dynamic; part of the prefix
//	isMutable     bool        // !isReadOnly
//}
//
//func NewStore(appDB dbm.DB, height int64, storeKey string, commitID types.CommitID, stateDir string, isMutable bool) *Store {
//	store := &Store{
//		appDB:     appDB,
//		storeKey:  storeKey,
//		isMutable: isMutable,
//		height:    height,
//	}
//	if isMutable {
//		// load height-1 into state from AppDB
//		prefix := StoreKey(height-1, storeKey, "")
//		it, err := appDB.Iterator(prefix, types.PrefixEndBytes(prefix))
//		if err != nil {
//			panic(fmt.Sprintf("unable to create an iterator for height %d storeKey %s", height, storeKey))
//		}
//		defer it.Close()
//		store.state, err = dbm.NewGoLevelDB(storeKey, stateDir)
//		if err != nil {
//			panic(err)
//		}
//		for ; it.Valid(); it.Next() {
//			err := store.state.Set(KeyFromStoreKey(it.Key()), it.Value())
//			if err != nil {
//				panic("unable to set k/v in state: " + err.Error())
//			}
//		}
//		// load IAVL from AppDB
//		store.iavl, err = iavl.LoadStore(dbm.NewPrefixDB(appDB, []byte("s/k:"+storeKey+"/")), commitID, false)
//		if err != nil {
//			panic("unable to load iavlStore in rootmultistore: " + err.Error())
//		}
//	}
//	fmt.Println(sqlite3.SQLITE_OK)
//	database, dbError := sql.Open("sqlite3", "/Users/luyzdeleon/current_projects/pocket-datadirs/waves/app.db")
//	if dbError != nil {
//		fmt.Println(dbError)
//	}
//	database.Exec(fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (height NUMBER, keyvalue TEXT, value TEXT, deletedat NUMBER, PRIMARY KEY (height, keyvalue))", storeKey))
//	store.sqlDB = database
//	store.deletionStmts = []*sql.Stmt{}
//	return store
//}
//
//func (is *Store) LoadImmutableVersion(version int64, stateDir string) *Store {
//	return NewStore(is.appDB, version, is.storeKey, types.CommitID{}, stateDir, false)
//}
//
//func (is *Store) Get(key []byte) ([]byte, error) {
//	if is.isMutable { // if latestHeight
//		return is.state.Get(key)
//	}
//	return is.appDB.Get(StoreKey(is.height-1, is.storeKey, string(key)))
//}
//
//func (is *Store) Has(key []byte) (bool, error) {
//	if is.isMutable { // if latestHeight
//		return is.state.Has(key)
//	}
//	return is.appDB.Has(StoreKey(is.height-1, is.storeKey, string(key)))
//}
//
//func (is *Store) Set(key, value []byte) error {
//	if is.isMutable {
//		err := is.iavl.Set(key, value)
//		if err != nil {
//			panic("unable to set to iavl: " + err.Error())
//		}
//		return is.state.Set(key, value)
//	}
//	panic("'Set()' called on immutable store")
//}
//
//func (is *Store) Delete(key []byte) error {
//	if is.isMutable {
//		err := is.iavl.Delete(key)
//		if err != nil {
//			panic("unable to delete to iavl: " + err.Error())
//		}
//		keyStr := string(key)
//		splitKey := strings.Split(keyStr, "/")
//		recordHeight, recordHeightError := strconv.Atoi(splitKey[0])
//		if recordHeightError != nil {
//			panic(recordHeightError)
//		}
//		keyvalue := splitKey[2]
//		deletedat := is.height + 1
//		delStmt, _ := is.sqlDB.Prepare(fmt.Sprintf("UPDATE %s SET deletedat = %d WHERE keyvalue = '%s' AND height = %d", is.storeKey, deletedat, keyvalue, recordHeight))
//		is.deletionStmts = append(is.deletionStmts, delStmt)
//		return is.state.Delete(key)
//	}
//	panic("'Delete()' called on immutable store")
//}
//
//func (is *Store) Iterator(start, end []byte) (types.Iterator, error) {
//	if is.isMutable {
//		return is.state.Iterator(start, end)
//	}
//	baseIterator, err := is.appDB.Iterator(StoreKey(is.height-1, is.storeKey, string(start)), StoreKey(is.height-1, is.storeKey, string(end)))
//	return AppDBIterator{it: baseIterator}, err
//}
//
//func (is *Store) ReverseIterator(start, end []byte) (types.Iterator, error) {
//	if is.isMutable {
//		return is.state.ReverseIterator(start, end)
//	}
//	baseIterator, err := is.appDB.ReverseIterator(StoreKey(is.height-1, is.storeKey, string(start)), StoreKey(is.height-1, is.storeKey, string(end)))
//	return AppDBIterator{it: baseIterator}, err
//}
//
//// Persist State & IAVL
//func (is *Store) CommitBatch(b dbm.Batch) (commitID types.CommitID, batch dbm.Batch) {
//	// commit iavl
//	commitID = is.iavl.Commit()
//	// commit entire state
//	it, err := is.state.Iterator(nil, nil)
//	if err != nil {
//		panic(fmt.Sprintf("unable to create an iterator for height %d storeKey %s in Commit()", is.height, is.storeKey))
//	}
//	defer it.Close()
//	for ; it.Valid(); it.Next() {
//		b.Set(StoreKey(is.height, is.storeKey, string(it.Key())), it.Value())
//
//		// Insert into SQL db
//		// key := string(StoreKey(is.height, is.storeKey, string(it.Key())))
//		height := is.height + 1
//		storekey := is.storeKey
//		keyvalue := string(it.Key())
//		queryStmt, _ := is.sqlDB.Prepare(fmt.Sprintf("SELECT MAX(height), value, deletedat FROM %s WHERE keyvalue = ?", storekey))
//		rows, _ := queryStmt.Query(keyvalue)
//		var maxHeight sql.NullInt64
//		var latestValue sql.NullString
//		var deletedat sql.NullInt64
//		for rows.Next() {
//			rows.Scan(&maxHeight, &latestValue, &deletedat)
//		}
//
//		// If the latest value is null OR the latest value is different than the value being written OR deleted at != null
//		if !latestValue.Valid || latestValue.String != string(it.Value()) || deletedat.Valid {
//			// Insert the record because the value changed or doesn't exist
//			statement, _ := is.sqlDB.Prepare(fmt.Sprintf("INSERT OR REPLACE INTO %s (height, keyvalue, value, deletedat) VALUES (?, ?, ?, ?)", storekey))
//			statement.Exec(height, keyvalue, string(it.Value()), sql.NullInt64{})
//		}
//
//		// Process deletions
//		for i := 0; i < len(is.deletionStmts); i++ {
//			deletionStmt := is.deletionStmts[i]
//			if deletionStmt != nil {
//				deletionStmt.Exec()
//			}
//		}
//
//		// Reinitialize deletions slice
//		is.deletionStmts = []*sql.Stmt{}
//	}
//	is.height++
//	return commitID, b
//}
//
//// Prune version in IAVL & AppDB
//func (is *Store) PruneVersion(batch dbm.Batch, version int64) dbm.Batch {
//	// iavl
//	is.iavl.DeleteVersion(version)
//	// appDB
//	prefix := StoreKey(version, is.storeKey, "")
//	it, err := is.appDB.Iterator(prefix, types.PrefixEndBytes(prefix))
//	if err != nil {
//		panic("unable to create iterator in PruneVersion for appDB")
//	}
//	defer it.Close()
//	for ; it.Valid(); it.Next() {
//		batch.Delete(it.Key())
//	}
//	return batch
//}
//
//func (is *Store) LastCommitID() types.CommitID {
//	if is.isMutable {
//		return is.iavl.LastCommitID()
//	}
//	panic("LastCommitID for called on an immutable store")
//}
//
//func (is *Store) CacheWrap() types.CacheWrap {
//	return cachemulti.NewStoreCache(is)
//}
//
//func (is *Store) GetStoreType() types.StoreType {
//	return types.StoreTypeDefault
//}
//
//func (is *Store) Commit() types.CommitID {
//	panic("use CommitBatch for atomic safety")
//}
//
//var _ types.CommitKVStore = &Store{}
//
//func StoreKey(height int64, store string, key string) []byte {
//	height += 1
//	if store == "" {
//		return []byte(fmt.Sprintf("%d/", height))
//	}
//	if key == "" {
//		return []byte(fmt.Sprintf("%d/%s/", height, store))
//	}
//	return []byte(fmt.Sprintf("%d/%s/%s", height, store, key))
//}
//
//func KeyFromStoreKey(storeKey []byte) (key []byte) {
//	delim := 0
//	for i, b := range storeKey {
//		if b == byte('/') {
//			delim++
//		}
//		if delim == 2 {
//			return storeKey[i+1:]
//		}
//	}
//	panic("attempted to get key from store key that doesn't have exactly 2 delims")
//}
//
//var _ dbm.Iterator = AppDBIterator{}
//
//// Is 'height/storeKey' aware
//type AppDBIterator struct {
//	it dbm.Iterator
//}
//
//func (s AppDBIterator) Key() (key []byte) {
//	return KeyFromStoreKey(s.it.Key())
//}
//
//func (s AppDBIterator) Valid() bool {
//	return s.it.Valid()
//}
//
//func (s AppDBIterator) Next() {
//	s.it.Next()
//}
//
//func (s AppDBIterator) Value() (value []byte) {
//	return s.it.Value()
//}
//
//func (s AppDBIterator) Error() error {
//	return s.it.Error()
//}
//
//func (s AppDBIterator) Close() {
//	s.it.Close()
//}
//
//func (s AppDBIterator) Domain() (start []byte, end []byte) {
//	st, end := s.it.Domain()
//	return KeyFromStoreKey(st), KeyFromStoreKey(end)
//}
//
