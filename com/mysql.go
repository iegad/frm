package com

import (
	"database/sql"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/gox/frm/log"
)

type MysqlConfig struct {
	Addr     string `yaml:"addr"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	Timeout  int64  `yaml:"timeout"`
}

func NewSql(c *MysqlConfig) (*sql.DB, error) {
	if c == nil {
		log.Fatal("c is nil")
	}

	mc := &mysql.Config{
		Addr:                 c.Addr,
		User:                 c.Username,
		Passwd:               c.Password,
		Net:                  "tcp",
		Loc:                  time.Local,
		Collation:            "utf8mb4_general_ci",
		MaxAllowedPacket:     64 << 20,
		AllowNativePasswords: true,
		CheckConnLiveness:    true,
		ParseTime:            true,
		Timeout:              time.Duration(c.Timeout) * time.Second,
	}

	db, err := sql.Open("mysql", mc.FormatDSN())
	if err != nil {
		return nil, err
	}

	err = db.Ping()
	if err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}
