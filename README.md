# StructOf
### Generate model from database connection
#
```
go install github.com/opannapo/structof@latest
```

### Postgres Database - Gorm Tag
```
structof --sqltype=postgres \
--connstr "host={$DB_HOST} user={$DB_USERNAME} password={$PASSWORD} dbname={$DB_NAME} port={$DB_PORT} sslmode=disable TimeZone=Asia/Jakarta" \
--database {$DB_NAME} \
--json \
--gorm \
--out ./
```

### Result
```
package model

import (
	"database/sql"
	"time"
)

// User struct is a row record of the user table in the DbName database
type User struct {
	UserRefID string         `gorm:"primary_key;column:user_ref_id;type:VARCHAR;" json:"user_ref_id" xml:"user_ref_id"` //[ 0] user_ref_id                                    VARCHAR              null: false  primary: true   isArray: false  auto: false  col: VARCHAR         len: -1      default: []
	Username  string         `gorm:"column:username;type:VARCHAR;" json:"username" xml:"username"`                      //[ 1] username                                       VARCHAR              null: false  primary: false  isArray: false  auto: false  col: VARCHAR         len: -1      default: []
	Password  sql.NullString `gorm:"column:password;type:VARCHAR;" json:"password" xml:"password"`                      //[ 2] password                                       VARCHAR              null: true   primary: false  isArray: false  auto: false  col: VARCHAR         len: -1      default: []
	Roles     sql.NullString `gorm:"column:roles;type:VARCHAR;" json:"roles" xml:"roles"`                               //[ 3] roles                                          VARCHAR              null: true   primary: false  isArray: false  auto: false  col: VARCHAR         len: -1      default: []
	UpdatedAt time.Time      `gorm:"column:updated_at;type:TIMESTAMP;" json:"updated_at" xml:"updated_at"`              //[ 4] updated_at                                     TIMESTAMP            null: true   primary: false  isArray: false  auto: false  col: TIMESTAMP       len: -1      default: [now()]

}

// TableName sets the insert table name for this struct type
func (c *User) TableName() string {
	return "user"
}
```

### Postgres Database - Db Tag
```
structof --sqltype=postgres \
--connstr "host={$DB_HOST} user={$DB_USERNAME} password={$PASSWORD} dbname={$DB_NAME} port={$DB_PORT} sslmode=disable TimeZone=Asia/Jakarta" \
--database {$DB_NAME} \
--json \
--db \
--out ./
```

### Result
```
package model

import (
	"database/sql"
	"time"
)

// User struct is a row record of the user table in the DbName database
type User struct {
	UserRefID string         `json:"user_ref_id" xml:"user_ref_id" db:"user_ref_id"` //[ 0] user_ref_id                                    VARCHAR              null: false  primary: true   isArray: false  auto: false  col: VARCHAR         len: -1      default: []
	Username  string         `json:"username" xml:"username" db:"username"`          //[ 1] username                                       VARCHAR              null: false  primary: false  isArray: false  auto: false  col: VARCHAR         len: -1      default: []
	Password  sql.NullString `json:"password" xml:"password" db:"password"`          //[ 2] password                                       VARCHAR              null: true   primary: false  isArray: false  auto: false  col: VARCHAR         len: -1      default: []
	Roles     sql.NullString `json:"roles" xml:"roles" db:"roles"`                   //[ 3] roles                                          VARCHAR              null: true   primary: false  isArray: false  auto: false  col: VARCHAR         len: -1      default: []
	UpdatedAt time.Time      `json:"updated_at" xml:"updated_at" db:"updated_at"`    //[ 4] updated_at                                     TIMESTAMP            null: true   primary: false  isArray: false  auto: false  col: TIMESTAMP       len: -1      default: [now()]
}

// TableName sets the insert table name for this struct type
func (c *User) TableName() string {
	return "user"
}
```
