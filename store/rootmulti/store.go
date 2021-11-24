package rootmulti

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"github.com/pokt-network/pocket-core/store/cachemulti"
	"github.com/pokt-network/pocket-core/store/iavl"
	"github.com/pokt-network/pocket-core/store/rootmulti/appstatedb"
	"github.com/pokt-network/pocket-core/store/types"
	dbm "github.com/tendermint/tm-db"
)

// Prefixed abstractions living inside AppDB;
type Store struct {
	appIAVLDB dbm.DB                 // iavl exclusive db (where everything lives except for state)
	asdb      *appstatedb.AppStateDB // Inmutable app state database
	iavl      *iavl.Store            // used for latest height state commitments ONLY; may be pruned;
	isMutable bool                   // !isReadOnly
	height    int64                  // dynamic; part of the prefix
	storeKey  string                 // constant; part of the prefix
	isDebug   bool
}

func NewStore(appIAVLDB dbm.DB, height int64, storeKey string, commitID types.CommitID, stateDir string, isMutable bool) *Store {
	store := &Store{
		appIAVLDB: appIAVLDB,
		isMutable: isMutable,
		height:    height,
		storeKey:  storeKey,
		isDebug:   true,
	}

	// load IAVL from AppDB
	iavlStore, err := iavl.LoadStore(dbm.NewPrefixDB(appIAVLDB, []byte("s/k:"+storeKey+"/")), commitID, false)
	if err != nil {
		panic("unable to load iavlStore in store: " + err.Error())
	}
	store.iavl = iavlStore

	// Load the app state db for this store
	asdb, asdbErr := appstatedb.NewAppStateDB(stateDir, storeKey)
	if asdbErr != nil {
		panic("Unable to load app statedb in store: " + asdbErr.Error())
	}
	store.asdb = asdb
	return store
}

func (is *Store) LoadImmutableVersion(version int64, stateDir string) *Store {
	return NewStore(is.appIAVLDB, version, is.storeKey, types.CommitID{
		Version: version,
	}, stateDir, false)
}

// Get returns from asdb
func (is *Store) Get(key []byte) ([]byte, error) {
	var result []byte
	var getErr error
	if is.isMutable {
		// Query the current block from the getmutable
		result, getErr = is.asdb.GetMutable(is.height, is.storeKey, key)
	} else {
		// Need to query from the block before the store to get immutable results
		result, getErr = is.asdb.GetMutable(is.height-1, is.storeKey, key)
	}

	if getErr != nil {
		panic("Error on asdb get:" + getErr.Error())
	}

	if is.isDebug {
		iavlGet, iavlGetErr := is.iavl.Get(key)
		if iavlGetErr != nil {
			panic("Error on iavl get: " + iavlGetErr.Error())
		}

		if !bytes.Equal(iavlGet, result) || (iavlGet == nil && result != nil) || (iavlGet != nil && result == nil) {
			fmt.Println(fmt.Sprintf("Different get results for key: %s", hex.EncodeToString(key)))
			fmt.Println(fmt.Sprintf("IAVL Value: %s", hex.EncodeToString(iavlGet)))
			fmt.Println(fmt.Sprintf("ASDB Value: %s", hex.EncodeToString(result)))
			panic(fmt.Sprintf("Difference in get response between iavl get %s and result get %s", iavlGet, result))
		}
	}

	return result, getErr
}

// Has returns from asdb
func (is *Store) Has(key []byte) (bool, error) {
	var result bool
	var hasErr error
	if is.isMutable {
		// Query the current block from the getmutable
		result, hasErr = is.asdb.HasMutable(is.height, is.storeKey, key)
	} else {
		// Need to query from the block before the store to get immutable results
		result, hasErr = is.asdb.HasMutable(is.height-1, is.storeKey, key)
	}

	if hasErr != nil {
		panic("Error on asdb has:" + hasErr.Error())
	}

	if is.isDebug {
		iavlHas, iavlHasErr := is.iavl.Has(key)
		if iavlHasErr != nil {
			panic(iavlHasErr)
		}

		if iavlHas != result {
			panic(fmt.Sprintf("Iavl has %s different from asdb has %s", iavlHas, result))
		}
	}

	return result, hasErr
}

// Sets both to iavl and asdb
func (is *Store) Set(key, value []byte) error {
	if is.isMutable {
		iavlErr := is.iavl.Set(key, value)
		if iavlErr != nil {
			panic("unable to set to the iavl: " + iavlErr.Error())
		}
		err := is.asdb.SetMutable(is.height, is.storeKey, key, value)
		if err != nil {
			panic("unable to set to the state: " + err.Error())
		}
		return err
	}
	panic("'Set()' called on immutable store")
}

// Deletes both from the iavl and asdb
func (is *Store) Delete(key []byte) error {
	if is.isMutable {
		if is.height == 73 {
			fmt.Println("WE'VE REACHED 73")
		}

		err := is.iavl.Delete(key)
		if err != nil {
			panic("unable to delete to iavl: " + err.Error())
		}
		delErr := is.asdb.DeleteMutable(is.height, is.storeKey, key)
		if delErr != nil {
			panic("unable to delete from mutable asdb: " + delErr.Error())
		}
		return nil
	}
	panic("'Delete()' called on immutable store")
}

func iteratorEquals(iterator1, iterator2 types.Iterator) bool {
	// First compares validity
	if iterator1.Valid() != iterator2.Valid() {
		return false
	}

	// Compare contents and order
	for iterator1.Valid() && iterator2.Valid() {
		if !bytes.Equal(iterator1.Key(), iterator2.Key()) || !bytes.Equal(iterator1.Value(), iterator2.Value()) {
			fmt.Println(fmt.Sprintf("Iterator Keys are different between %s and %s", hex.EncodeToString(iterator1.Key()), hex.EncodeToString(iterator2.Key())))
			fmt.Println(fmt.Sprintf("Iterator Values are different between %s and %s", hex.EncodeToString(iterator1.Value()), hex.EncodeToString(iterator2.Value())))
			return false
		}

		if (iterator1.Key() == nil && iterator2.Key() != nil) || (iterator2.Key() == nil && iterator1.Key() != nil) {
			fmt.Println(fmt.Sprintf("Iterator Keys are different between %s and %s", hex.EncodeToString(iterator1.Key()), hex.EncodeToString(iterator2.Key())))
			fmt.Println(fmt.Sprintf("Iterator Values are different between %s and %s", hex.EncodeToString(iterator1.Value()), hex.EncodeToString(iterator2.Value())))
			return false
		}

		if (iterator1.Value() == nil && iterator2.Value() != nil) || (iterator2.Value() == nil && iterator1.Value() != nil) {
			fmt.Println(fmt.Sprintf("Iterator Keys are different between %s and %s", hex.EncodeToString(iterator1.Key()), hex.EncodeToString(iterator2.Key())))
			fmt.Println(fmt.Sprintf("Iterator Values are different between %s and %s", hex.EncodeToString(iterator1.Value()), hex.EncodeToString(iterator2.Value())))
			return false
		}
		iterator1.Next()
		iterator2.Next()

		if iterator1.Valid() != iterator2.Valid() {
			if iterator1.Valid() {
				fmt.Println("PRINTING THE REMAINDER OF ITERATOR 1 ENTRIES")
				for iterator1.Valid() {
					fmt.Println("------------")
					fmt.Println(fmt.Sprintf("Key: %s", hex.EncodeToString(iterator1.Key())))
					fmt.Println(fmt.Sprintf("Value: %s", hex.EncodeToString(iterator1.Value())))
					fmt.Println("------------")
					iterator1.Next()
				}
			}

			if iterator2.Valid() {
				fmt.Println("PRINTING THE REMAINDER OF ITERATOR 2 ENTRIES")
				for iterator2.Valid() {
					fmt.Println("------------")
					fmt.Println(fmt.Sprintf("Key: %s", hex.EncodeToString(iterator2.Key())))
					fmt.Println(fmt.Sprintf("Value: %s", hex.EncodeToString(iterator2.Value())))
					fmt.Println("------------")
					iterator2.Next()
				}
			}
			return false
		}
	}
	return true
}

// Iterator returns the asdb iterator
func (is *Store) Iterator(start, end []byte) (types.Iterator, error) {
	var result types.Iterator
	var itErr error
	if is.isMutable {
		// Query the current block from the getmutable
		result, itErr = is.asdb.IteratorMutable(is.height, is.storeKey, start, end)
	} else {
		// Need to query from the block before the store to get immutable results
		result, itErr = is.asdb.IteratorMutable(is.height - 1, is.storeKey, start, end)
	}

	if is.isDebug {
		iavlIt, iavlItErr := is.iavl.Iterator(start, end)
		if iavlItErr != nil {
			panic("Iavl Iterator error: " + iavlItErr.Error())
		}

		if !iteratorEquals(iavlIt, result) {
			fmt.Println(fmt.Sprintf("Different Iterators on height: %d", is.height))
			fmt.Println(fmt.Sprintf("Different Iterators with start: %s and end: %s", hex.EncodeToString(start), hex.EncodeToString(end)))
			panic(fmt.Sprintf("Different Iterators on table %s", is.storeKey))
		}

		// The actual result returned
		if is.isMutable {
			// Query the current block from the getmutable
			result, itErr = is.asdb.IteratorMutable(is.height, is.storeKey, start, end)
		} else {
			// Need to query from the block before the store to get immutable results
			result, itErr = is.asdb.IteratorMutable(is.height - 1, is.storeKey, start, end)
		}
	}

	return result, itErr
}

// Returns the asdb reverseiterator
func (is *Store) ReverseIterator(start, end []byte) (types.Iterator, error) {
	var result types.Iterator
	var itErr error
	if is.isMutable {
		// Query the current block from the getmutable
		result, itErr = is.asdb.ReverseIteratorMutable(is.height, is.storeKey, start, end)
	} else {
		// Need to query from the block before the store to get immutable results
		result, itErr = is.asdb.ReverseIteratorMutable(is.height - 1, is.storeKey, start, end)
	}

	if is.isDebug {
		iavlIt, iavlItErr := is.iavl.ReverseIterator(start, end)
		if iavlItErr != nil {
			panic("Iavl Iterator error: " + iavlItErr.Error())
		}

		if !iteratorEquals(iavlIt, result) {
			panic(fmt.Sprintf("Different Iterators on table %s", is.storeKey))
		}

		// The actual result returned
		if is.isMutable {
			// Query the current block from the getmutable
			result, itErr = is.asdb.ReverseIteratorMutable(is.height, is.storeKey, start, end)
		} else {
			// Need to query from the block before the store to get immutable results
			result, itErr = is.asdb.ReverseIteratorMutable(is.height - 1, is.storeKey, start, end)
		}
	}

	return result, itErr
}

// Prune version in IAVL & AppDB
func (is *Store) PruneIAVLVersion(version int64) {
	// iavl
	is.iavl.DeleteVersion(version)
}

func (is *Store) LastCommitID() types.CommitID {
	if is.isMutable {
		return is.iavl.LastCommitID()
	}
	panic("LastCommitID for called on an immutable store")
}

func (is *Store) CacheWrap() types.CacheWrap {
	return cachemulti.NewStoreCache(is)
}

func (is *Store) GetStoreType() types.StoreType {
	return types.StoreTypeDefault
}

func (is *Store) Commit() types.CommitID {
	// commit iavl
	commitID := is.iavl.Commit()
	// Commit state
	commitErr := is.asdb.CommitMutable()
	if commitErr != nil {
		panic(commitErr.Error())
	}
	// Increase the version of the store
	is.height++

	return commitID
}

var _ types.CommitKVStore = &Store{}
