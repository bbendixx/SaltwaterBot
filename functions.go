package main

import (
	"bufio"
	_ "database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	_ "github.com/mattn/go-sqlite3"
)

func SaveMapStats(c *gin.Context) {

	fileName := c.Query("fileName")
	winner := strings.ToLower(c.Query("winner"))
	mapPlayed := strings.ToLower(c.Query("map"))
	matchID, _ := strconv.Atoi(c.Query("matchID"))

	winner = strings.ReplaceAll(winner, "_", " ")
	mapPlayed = strings.ReplaceAll(mapPlayed, "_", " ")

	if matchID == 0 || mapPlayed == "" || winner == "" || fileName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Missing required query parameters"})
		return
	}

	playerStats, mapInfo := readFile(fileName + ".txt")

	mapInfo.Name, mapInfo.Winner, mapInfo.MatchID = mapPlayed, winner, matchID

	mapID := createMap(mapInfo)

	response := saveStatsToDB(playerStats, mapID, mapInfo.TotalTimeInSeconds)

	c.JSON(http.StatusOK, gin.H{"message": response})
}

func CreateMatch(c *gin.Context) {
	var (
		count   int
		teams   [2]string
		matchID int
	)

	team1 := strings.ToLower(c.Query("team1"))
	team2 := strings.ToLower(c.Query("team2"))
	grandfinals, _ := strconv.Atoi(c.Query("grandfinals"))

	team1 = strings.ReplaceAll(team1, "_", " ")
	team2 = strings.ReplaceAll(team2, "_", " ")

	teams[0], teams[1] = team1, team2

	db := ConnectToDatabase()
	defer db.Close()

	for _, team := range teams {
		err := db.QueryRow("SELECT COUNT(*) FROM team WHERE name = ?", team).Scan(&count)
		if err != nil {
			fmt.Println(err, "CreateMatch()")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
			return
		}

		if count == 0 {
			sqlInsert := `INSERT INTO team (name, seasonsPlayed) VALUES (?, 1)`
			_, err := db.Exec(sqlInsert, team)
			if err != nil {
				fmt.Println(err, "CreateMatch()")
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
				return
			}
		}
	}

	sqlInsert := `INSERT INTO game (team1, team2, grandfinals) VALUES (?, ?, ?)`
	_, err := db.Exec(sqlInsert, team1, team2, grandfinals)
	if err != nil {
		fmt.Println(err, "CreateMatch()")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	err = db.QueryRow("SELECT MAX(ID) FROM game").Scan(&matchID)
	if err != nil {
		fmt.Println(err, "CreateMatch()")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": matchID})
}

func GetPlayerStats(c *gin.Context) {

	var totalTime int
	var totalStats Stats

	player := strings.ToLower(c.Query("player"))

	db := ConnectToDatabase()
	defer db.Close()

	rows, err := db.Query("SELECT damageDealt, damageTaken, deaths, finalBlows, eliminations, soloKills, healingDealt, environmentalKills, offensiveAssists, ultsUsed, durationInSeconds FROM mapPlayer WHERE player = ?", player)

	if err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "No player stats found."})
		return
	}
	defer rows.Close()

	for rows.Next() {
		var stats Stats
		err := rows.Scan(&stats.DamageDealt, &stats.DamageTaken, &stats.Deaths, &stats.FinalBlows, &stats.Eliminations, &stats.SoloKills, &stats.HealingDealt, &stats.EnvironmentalKills, &stats.OffensiveAssists, &stats.UltsUsed, &stats.DurationInSeconds)
		if err != nil {
			log.Fatal(err, "GetPlayerStats()")
		}

		totalTime += stats.DurationInSeconds

		totalStats.DamageDealt += stats.DamageDealt * float64(stats.DurationInSeconds)
		totalStats.DamageTaken += stats.DamageTaken * float64(stats.DurationInSeconds)
		totalStats.Deaths += stats.Deaths * float64(stats.DurationInSeconds)
		totalStats.FinalBlows += stats.FinalBlows * float64(stats.DurationInSeconds)
		totalStats.Eliminations += stats.Eliminations * float64(stats.DurationInSeconds)
		totalStats.SoloKills += stats.SoloKills * float64(stats.DurationInSeconds)
		totalStats.HealingDealt += stats.HealingDealt * float64(stats.DurationInSeconds)
		totalStats.EnvironmentalKills += stats.EnvironmentalKills * float64(stats.DurationInSeconds)
		totalStats.OffensiveAssists += stats.OffensiveAssists * float64(stats.DurationInSeconds)
		totalStats.UltsUsed += stats.UltsUsed * float64(stats.DurationInSeconds)
	}

	if err = rows.Err(); err != nil {
		log.Fatal(err, "GetPlayerStats()")
	}

	normalizedStats := Stats{
		DamageDealt:        (totalStats.DamageDealt / float64(totalTime)),
		DamageTaken:        (totalStats.DamageTaken / float64(totalTime)),
		Deaths:             (totalStats.Deaths / float64(totalTime)),
		FinalBlows:         (totalStats.FinalBlows / float64(totalTime)),
		Eliminations:       (totalStats.Eliminations / float64(totalTime)),
		SoloKills:          (totalStats.SoloKills / float64(totalTime)),
		HealingDealt:       (totalStats.HealingDealt / float64(totalTime)),
		EnvironmentalKills: (totalStats.EnvironmentalKills / float64(totalTime)),
		OffensiveAssists:   (totalStats.OffensiveAssists / float64(totalTime)),
		UltsUsed:           (totalStats.UltsUsed / float64(totalTime)),
	}

	sql := `SELECT hero, time FROM playerHero WHERE player = ? ORDER BY time DESC LIMIT 3`

	rows, err = db.Query(sql, player)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "No player stats found."})
		return
	}
	defer rows.Close()

	for rows.Next() {
		var heroStats HeroStats
		err := rows.Scan(&heroStats.Hero, &heroStats.TimeSpentInSeconds)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{"message": "No player stats found."})
			return
		}
		normalizedStats.TopHeroes = append(normalizedStats.TopHeroes, heroStats)
	}

	var team string

	err = db.QueryRow("SELECT team FROM player WHERE name = ?", player).Scan(&team)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "No player stats found."})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": formatStatsMessage(normalizedStats, team)})

}

func formatStatsMessage(stats Stats, team string) string {

	heroInfo := fmt.Sprintf("Team: %s\n\nMost Played Heroes:\n", team)
	for i := range stats.TopHeroes {
		minutes := stats.TopHeroes[i].TimeSpentInSeconds / 60
		seconds := stats.TopHeroes[i].TimeSpentInSeconds % 60
		heroInfo += fmt.Sprintf("%d. %s %d:%d\n", i+1, stats.TopHeroes[i].Hero, minutes, seconds)
	}

	return fmt.Sprintf(
		"%s\n"+
			"Damage per 10 Minutes: %.2f\n"+
			"Damage Taken per 10 Minutes: %.2f\n"+
			"Deaths per 10 Minutes: %.2f\n"+
			"Final Blows per 10 Minutes: %.2f\n"+
			"Eliminations per 10 Minutes: %.2f\n"+
			"Solo Kills per 10 Minutes: %.2f\n"+
			"Healing per 10 Minutes: %.2f\n"+
			"Environmental Kills per 10 Minutes: %.2f\n"+
			"Offensive Assists per 10 Minutes: %.2f\n"+
			"Ultimates Used per 10 Minutes: %.2f\n",
		heroInfo,
		stats.DamageDealt,
		stats.DamageTaken,
		stats.Deaths,
		stats.FinalBlows,
		stats.Eliminations,
		stats.SoloKills,
		stats.HealingDealt,
		stats.EnvironmentalKills,
		stats.OffensiveAssists,
		stats.UltsUsed,
	)
}

func createMap(mapInfo Map) int {

	db := ConnectToDatabase()

	var mapID int

	sql := `INSERT INTO map (gameID, name, winner, durationInSeconds) VALUES (?, ?, ?, ?)`

	statement, err := db.Prepare(sql)

	if err != nil {
		fmt.Println(err, "CreateMap()")
	}
	_, err = statement.Exec(mapInfo.MatchID, mapInfo.Name, mapInfo.Winner, mapInfo.TotalTimeInSeconds)

	if err != nil {
		fmt.Println(err, "createMap()")
	}

	db.Close()

	db = ConnectToDatabase()
	defer db.Close()

	err = db.QueryRow("SELECT MAX(ID) FROM map").Scan(&mapID)
	if err != nil {
		fmt.Println(err, "createMap()")
	}

	return mapID
}

func readFile(fileName string) ([10]PlayerStats, Map) {

	var (
		lines       []string
		playerStats PlayerStats
		players     [10]PlayerStats
		playedMap   Map
	)

	file, err := os.Open(fileName)
	if err != nil {
		fmt.Println("Error opening file:", err)
		return players, playedMap
	}

	defer file.Close()

	scanner := bufio.NewScanner(file)

	prevSeconds, currSeconds, totalTimeInSeconds := 0, 0, 0
	lineCount := 0
	setupPhase := false

	for scanner.Scan() {
		line := scanner.Text()
		lines = append(lines, line)
		lineCount++
		if lineCount <= 10 {
			players = getPlayerNames(line, lineCount, players)
		}

		if lineCount%10 == 1 {
			currSeconds = getCurrentSeconds(line)
			if currSeconds == prevSeconds+1 {
				totalTimeInSeconds += 1
				setupPhase = false
			} else {
				setupPhase = true
			}
			prevSeconds = currSeconds
		}

		if !setupPhase && lineCount > 10 {
			playerStats = getHeroPlaytime(line, players)
			for j := 0; j < len(players); j++ {
				if players[j].Name == playerStats.Name {
					players[j] = playerStats
				}
			}
		}
	}

	start := len(lines) - 10
	lines = lines[start:]

	for i := 0; i < len(lines); i++ {
		playerStats = getPlayerStats(lines[i], players, totalTimeInSeconds)
		for j := 0; j < len(players); j++ {
			if players[j].Name == playerStats.Name {
				players[j] = playerStats
			}
		}
	}

	playedMap.TotalTimeInSeconds = totalTimeInSeconds

	if err := scanner.Err(); err != nil {
		fmt.Println("Error reading file:", err)
	}

	return players, playedMap
}

func getPlayerStats(line string, players [10]PlayerStats, timePlayed int) PlayerStats {

	stats := strings.Split(line, ",")
	playerName := stats[1]
	var player PlayerStats

	for i := 0; i < len(players); i++ {
		player = players[i]
		if player.Name == playerName {

			player.DamageDealtP10 = calculateStatP10(stats[3], timePlayed)
			player.DamageTakenP10 = calculateStatP10(stats[4], timePlayed)
			player.DeathsP10 = calculateStatP10(stats[5], timePlayed)
			player.FinalBlowsP10 = calculateStatP10(stats[6], timePlayed)
			player.EliminationsP10 = calculateStatP10(stats[7], timePlayed)
			player.SoloKillsP10 = calculateStatP10(stats[8], timePlayed)
			player.HealingDealtP10 = calculateStatP10(stats[9], timePlayed)
			player.EnvironmentalKillsP10 = calculateStatP10(stats[10], timePlayed)
			player.OffensiveAssistsP10 = calculateStatP10(stats[11], timePlayed)
			player.UltsUsedP10 = calculateStatP10(stats[12], timePlayed)
			player.Team = stats[13]

			return player
		}
	}

	return player

}

func getPlayerNames(line string, lineCount int, players [10]PlayerStats) [10]PlayerStats {

	playerName := strings.Split(line, ",")[1]
	players[lineCount-1].Name = playerName

	return players
}

func getHeroPlaytime(line string, players [10]PlayerStats) PlayerStats {
	stats := strings.Split(line, ",")
	playerName := stats[1]
	hero := stats[2]
	var player PlayerStats

	for i := 0; i < len(players); i++ {
		player = players[i]
		if player.Name == playerName {
			found := false
			for j := 0; j < len(player.Heroes); j++ {
				if player.Heroes[j].Hero == hero {
					player.Heroes[j].TimeSpentInSeconds += 1
					found = true
					break
				}
			}
			if !found {
				player.Heroes = append(player.Heroes, HeroStats{Hero: hero, TimeSpentInSeconds: 1})
			}
			return player
		}
	}

	return player
}

func saveStatsToDB(playerStats [10]PlayerStats, mapID int, timePlayed int) string {
	var response string

	db := ConnectToDatabase()
	defer db.Close()

	for i := 0; i < len(playerStats); i++ {
		playerName := strings.ToLower(playerStats[i].Name)
		teamName := strings.ToLower(playerStats[i].Team)

		var count int
		err := db.QueryRow("SELECT COUNT(*) FROM player WHERE name = ?", playerName).Scan(&count)
		if err != nil {
			fmt.Println(err, "saveStatsToDB() - Checking player existence")
			continue
		}

		if count == 0 {
			sql := `INSERT INTO player (name, team) VALUES (?, ?)`
			statement, err := db.Prepare(sql)
			if err != nil {
				fmt.Println(err, "saveStatsToDB() - Preparing player insert statement")
				continue
			}
			_, err = statement.Exec(playerName, teamName)
			if err != nil {
				fmt.Println(err, "saveStatsToDB() - Executing player insert")
				continue
			}
		}

		stmt := `INSERT INTO mapPlayer (mapID, player, damageDealt, damageTaken, deaths, finalBlows, eliminations, soloKills, healingDealt, environmentalKills, offensiveAssists, ultsUsed, durationInSeconds) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
		statement, err := db.Prepare(stmt)
		if err != nil {
			fmt.Println(err, "saveStatsToDB() - Preparing map player insert statement")
			continue
		}
		_, err = statement.Exec(mapID, playerName, playerStats[i].DamageDealtP10, playerStats[i].DamageTakenP10, playerStats[i].DeathsP10, playerStats[i].FinalBlowsP10, playerStats[i].EliminationsP10, playerStats[i].SoloKillsP10, playerStats[i].HealingDealtP10, playerStats[i].EnvironmentalKillsP10, playerStats[i].OffensiveAssistsP10, playerStats[i].UltsUsedP10, timePlayed)
		if err != nil {
			fmt.Println(err, "saveStatsToDB() - Executing map player insert")
			continue
		}

		for j := 0; j < len(playerStats[i].Heroes); j++ {
			heroName := playerStats[i].Heroes[j].Hero
			timeSpent := playerStats[i].Heroes[j].TimeSpentInSeconds

			// Check if player-hero exists
			err := db.QueryRow("SELECT COUNT(*) FROM playerHero WHERE player = ? AND hero = ?", playerName, heroName).Scan(&count)
			if err != nil {
				fmt.Println(err, "saveStatsToDB() - Checking player-hero existence")
				continue
			}

			if count == 0 {
				stmt = `INSERT INTO playerHero (player, hero, time) VALUES (?, ?, ?)`
				statement, err := db.Prepare(stmt)
				if err != nil {
					fmt.Println(err, "saveStatsToDB() - Preparing player-hero insert statement")
					continue
				}
				_, err = statement.Exec(playerName, heroName, timeSpent)
				if err != nil {
					fmt.Println(err, "saveStatsToDB() - Executing player-hero insert")
					continue
				}
			} else {
				_, err := db.Exec("UPDATE playerHero SET time = time + ? WHERE player = ? AND hero = ?", timeSpent, playerName, heroName)
				if err != nil {
					fmt.Println(err, "saveStatsToDB() - Updating player-hero time")
					continue
				}
			}
		}
	}

	response = "Stats added successfully."
	return response
}

func getCurrentSeconds(line string) int {

	timeString := strings.Split(line, " ")[0]
	timeString = strings.ReplaceAll(timeString, "[", "")
	timeString = strings.ReplaceAll(timeString, "]", "")
	timeArray := strings.Split(timeString, ":")

	minutes, _ := strconv.Atoi(timeArray[1])
	seconds, _ := strconv.Atoi(timeArray[2])

	return minutes*60 + seconds
}

func calculateStatP10(statString string, timePlayed int) float64 {
	stat, _ := strconv.ParseFloat(statString, 64)
	stat = (stat / float64(timePlayed)) * 600

	return stat
}
