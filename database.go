package main

import (
	"database/sql"

	_ "github.com/mattn/go-sqlite3"

	"fmt"
)

func ConnectToDatabase() *sql.DB {
	db, err := sql.Open("sqlite3", "./database.db")
	if err != nil {
		fmt.Println("Error connecting to database. (ConnectToDatabase)")
		return nil
	}

	_, err = db.Exec("PRAGMA foreign_keys = ON")
	if err != nil {
		fmt.Println("Error allowing foreign keys. (ConnectToDatabase)")
		db.Close()
		return nil
	}

	return db
}

func CreateDatabase() {

	fmt.Println("Creating database")

	db := ConnectToDatabase()

	defer db.Close()

	statement := `
    CREATE TABLE IF NOT EXISTS team (
		name TEXT PRIMARY KEY,
		seasonsPlayed INTEGER
    );

	CREATE TABLE IF NOT EXISTS player (
		name TEXT PRIMARY KEY,
		team TEXT,
		FOREIGN KEY (team) REFERENCES team(name)
    );

	CREATE TABLE IF NOT EXISTS division (
		ID INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
		name TEXT,
		colorHexcode TEXT
    );

	CREATE TABLE IF NOT EXISTS teamGroup (
		divisionID INTEGER,
		team TEXT,
		PRIMARY KEY (divisionID, team),
		FOREIGN KEY (divisionID) REFERENCES division(ID),
		FOREIGN KEY (team) REFERENCES team(name)
	);

	CREATE TABLE IF NOT EXISTS game (
		ID INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
		team1 TEXT,
		team2 TEXT,
		grandfinals INTEGER,
		FOREIGN KEY (team1) REFERENCES team(name),
		FOREIGN KEY (team2) REFERENCES team(name)
	);

	CREATE TABLE IF NOT EXISTS map (
		ID INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
		gameID INTEGER,
		name TEXT,
		winner TEXT,
		durationInSeconds INTEGER,
		FOREIGN KEY (gameID) REFERENCES game(ID),
		FOREIGN KEY (winner) REFERENCES team(name)
	);

	CREATE TABLE IF NOT EXISTS mapPlayer (
		ID INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
		mapID INTEGER,
		player TEXT,
		damageDealt REAL,
		damageTaken REAL,
		deaths REAL,
		finalBlows REAL,
		eliminations REAL,
		soloKills REAL,
		healingDealt REAL,
		environmentalKills REAL,
		offensiveAssists REAL,
		ultsUsed REAL,
		durationInSeconds INTEGER,
		FOREIGN KEY (mapID) REFERENCES map(ID),
		FOREIGN KEY (player) REFERENCES player(name)
	);

	CREATE TABLE IF NOT EXISTS playerHero (
		player TEXT,
		hero TEXT,
		damageDealt REAL,
		damageTaken REAL,
		deaths REAL,
		finalBlows REAL,
		eliminations REAL,
		soloKills REAL,
		healingDealt REAL,
		environmentalKills REAL,
		offensiveAssists REAL,
		ultsUsed REAL,
		durationInSeconds INTEGER,
		PRIMARY KEY (player, Hero),
		FOREIGN KEY (player) REFERENCES player(name)
	);
    `

	_, err := db.Exec(statement)
	if err != nil {
		panic(fmt.Sprintf("%q: %s\n", err, statement))
	}

}
