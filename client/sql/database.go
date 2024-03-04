package sql

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/go-sql-driver/mysql"
)

var DB *sql.DB

var ErrNil = fmt.Errorf("not found")

type PasskyAuth struct {
	UUID string
	Algo string
	Hash string
	Salt string
	IP   string
	Date string
}

type WhitelistPlayer struct {
	ID          int64
	Name        string
	UUID        string
	Discord     string
	Whitelisted int8
}

func init() {
	dblog := log.New(os.Stdout, "[DB] ", log.LstdFlags)

	cfg := mysql.Config{
		User:                 os.Getenv("DB_USER"),
		Passwd:               os.Getenv("DB_PASS"),
		Net:                  "tcp",
		Addr:                 os.Getenv("DB_ADDR"),
		DBName:               os.Getenv("DB_NAME"),
		AllowNativePasswords: true,
	}

	var err error
	DB, err = sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		dblog.Println(err)
		return
	}

	pingErr := DB.Ping()
	if pingErr != nil {
		dblog.Println(pingErr)
		return
	}

	dblog.Println("connected")
}

func GetPlayer(username string, discordID string) (WhitelistPlayer, error) {
	var user WhitelistPlayer

	rows, err := DB.Query("SELECT * FROM noble_whitelist WHERE Name=? OR Discord=? LIMIT 1", username, discordID)
	if err != nil {
		return user, err
	}

	defer rows.Close()
	for rows.Next() {
		if err := rows.Scan(&user.ID, &user.Name, &user.UUID, &user.Discord, &user.Whitelisted); err != nil {
			return user, fmt.Errorf("GetPlayer %q: %v", username, err)
		}
	}
	if err := rows.Err(); err != nil {
		return user, err
	}
	if user.ID == 0 {
		return user, ErrNil
	}

	return user, nil
}

func GetPlayerAuth(username string) (PasskyAuth, error) {
	var pAuth PasskyAuth

	rows, err := DB.Query("SELECT * FROM passky_players WHERE uuid=? LIMIT 1", username)
	if err != nil {
		return pAuth, err
	}

	defer rows.Close()
	for rows.Next() {
		if err := rows.Scan(pAuth.UUID, pAuth.Algo, pAuth.Hash, pAuth.Salt, pAuth.IP, pAuth.Date); err != nil {
			return pAuth, fmt.Errorf("GetPlayerAuth %q: %v", username, err)
		}
	}
	if err := rows.Err(); err != nil {
		return pAuth, err
	}
	if pAuth.UUID == "" {
		return pAuth, ErrNil
	}

	return pAuth, nil
}
