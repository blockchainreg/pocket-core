package appstatedb

// OLD GET QUERY
//const GetQuery = `
//    SELECT value
//      FROM %s
//     WHERE HEX(key) = ? AND
//           height <= ? AND
//           (NOT deleted_at <= (?) OR deleted_at is null)
//  GROUP BY height
//    HAVING height = MAX(height)
//  ORDER BY height DESC
//     LIMIT 1`

const GetQuery = `
SELECT %s.value
  FROM %s INNER JOIN (
      SELECT height as j_height, key as j_key, deleted_at as j_deleted_at
      FROM %s
      WHERE key = ? AND
            height <= ?
      GROUP BY height
      HAVING height = MAX(height)
      ORDER BY height DESC
      LIMIT 1
      ) ON %s.height = j_height AND %s.key = j_key
WHERE %s.deleted_at IS NULL OR NOT (%s.deleted_at <= ?)
`

const InsertStatement = `
	INSERT OR REPLACE INTO %s(height, key, value)
	SELECT ?, ?, ?
	 WHERE ? NOT IN
					(
					 SELECT %s.value
					  FROM %s INNER JOIN (
						  SELECT height as j_height, key as j_key, deleted_at as j_deleted_at
						  FROM %s
						  WHERE key = ? AND
                                height <= ?
						  GROUP BY height
						  HAVING height = MAX(height)
						  ORDER BY height DESC
						  LIMIT 1
						  ) ON %s.height = j_height AND %s.key = j_key
					WHERE %s.deleted_at IS NULL OR NOT (%s.deleted_at <= ?)
				   )
`

const DeleteStatement = `
  UPDATE %s
     SET deleted_at = %d
   WHERE (key, height) IN (
					SELECT %s.key, %s.height
					  FROM %s INNER JOIN (
						  SELECT height as j_height, key as j_key, deleted_at as j_deleted_at
						    FROM %s
						   WHERE key = ? AND
                                height <= ?
						  GROUP BY height
						  HAVING height = MAX(height)
						  ORDER BY height DESC
						  LIMIT 1
						  ) ON %s.height = j_height AND %s.key = j_key
					WHERE %s.deleted_at IS NULL OR NOT (%s.deleted_at <= ?)
   )
`

const IteratorQuery = `
SELECT key, value
  FROM %s
 WHERE (key, height) IN (
	SELECT %s.key, %s.height
	  FROM %s INNER JOIN (
		SELECT height as j_height, key as j_key, deleted_at as j_deleted_at
		  FROM %s
		 WHERE height <= ? AND
               HEX(key) LIKE '%s%%' AND
               HEX(key) < '%s'
      GROUP BY key
        HAVING height = MAX(height)
      ORDER BY height DESC
      ) ON %s.height = j_height AND %s.key = j_key
     WHERE %s.deleted_at IS NULL OR NOT (%s.deleted_at <= ?)
)
ORDER BY key %s
`

const IteratorAllQuery = `
SELECT key, value
  FROM %s
 WHERE (key, height) IN (
	SELECT %s.key, %s.height
	  FROM %s INNER JOIN (
		SELECT height as j_height, key as j_key, deleted_at as j_deleted_at
		  FROM %s
		 WHERE height <= ?
      GROUP BY key
        HAVING height = MAX(height)
      ORDER BY height DESC
      ) ON %s.height = j_height AND %s.key = j_key
     WHERE %s.deleted_at IS NULL OR NOT (%s.deleted_at <= ?)
)
ORDER BY key %s
`

//const IteratorQuery = `
//    SELECT key, value
//      FROM %s INNER JOIN (
//			SELECT MAX(height) as latestheight, key as submaxkey
//			  FROM %s
//            WHERE height <= ? AND
//                  (NOT deleted_at <= (?) OR deleted_at is null)
//         GROUP BY key
//		) submax ON %s.height = submax.latestheight AND %s.key = submax.submaxkey
//     WHERE HEX(key) LIKE '%s%%'
//  ORDER BY key %s`

//const IteratorAllQuery = `
//    SELECT key, value
//      FROM %s INNER JOIN (
//			SELECT MAX(height) as latestheight, key as submaxkey
//			  FROM %s
//            WHERE height <= ? AND
//                  (NOT deleted_at <= (?) OR deleted_at is null)
//         GROUP BY key
//		) submax ON %s.height = submax.latestheight AND %s.key = submax.submaxkey
//  ORDER BY key %s`

const createTableStatement = `
CREATE TABLE IF NOT EXISTS %s (height NUMBER NOT NULL, key BLOB, value BLOB, deleted_at NUMBER, PRIMARY KEY (height, key));
CREATE INDEX IF NOT EXISTS idx_%s_height_key_deleted_at ON %s (height, key, deleted_at);
CREATE INDEX IF NOT EXISTS idx_%s_height_deleted_at ON %s (height, deleted_at);
`

