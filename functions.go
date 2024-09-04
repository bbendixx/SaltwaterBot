package main

import (
	"bufio"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
	"unicode"

	"github.com/gin-gonic/gin"
)

func UploadMap(c *gin.Context) string {

	fileName := c.Query("fileName")
	winner := strings.ToLower(c.Query("winner"))
	mapPlayed := strings.ToLower(c.Query("map"))
	matchID, _ := strconv.Atoi(c.Query("matchID"))

	winner = strings.ReplaceAll(winner, "_", " ")
	mapPlayed = strings.ReplaceAll(mapPlayed, "_", " ")

	if matchID == 0 || mapPlayed == "" || winner == "" || fileName == "" {
		return "Missing required query parameters"
	}

	playerStats, mapInfo, err := readFile(fileName + ".txt")
	if err != nil {
		return "Couldn't read file"
	}

	mapInfo.Name, mapInfo.Winner, mapInfo.MatchID = mapPlayed, winner, matchID

	mapID := createMap(mapInfo)

	response := saveStatsToDB(playerStats, mapID, mapInfo.TotalTimeInSeconds)

	return response
}

func CreateMatch(c *gin.Context) string {
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
			return "Internal server error"
		}

		if count == 0 {
			sqlInsert := `INSERT INTO team (name, seasonsPlayed) VALUES (?, 1)`
			_, err := db.Exec(sqlInsert, team)
			if err != nil {
				fmt.Println(err, "CreateMatch()")
				return "Internal server error"
			}
		}
	}

	sqlInsert := `INSERT INTO game (team1, team2, grandfinals) VALUES (?, ?, ?)`
	_, err := db.Exec(sqlInsert, team1, team2, grandfinals)
	if err != nil {
		fmt.Println(err, "CreateMatch()")
		return "Internal server error"
	}

	err = db.QueryRow("SELECT MAX(ID) FROM game").Scan(&matchID)
	if err != nil {
		fmt.Println(err, "CreateMatch()")
		return "Internal server error"
	}

	return fmt.Sprintf("%d", matchID)
}

func PStats(c *gin.Context) string {
	var (
		stats PlayerStats
	)

	stats.Name = strings.ToLower(c.Query("player"))

	db := ConnectToDatabase()
	defer db.Close()

	stats, err := getPlayerStats(stats, db)
	if err != nil {
		return "No player stats found"
	}

	stats = calcStatsP10(stats)

	stats, err = getTop3Heroes(stats, db)
	if err != nil {
		return "An error occured while fetching most played heroes"
	}

	team, err := getPlayerTeam(stats.Name, db)
	if err != nil {
		return "An error occured while fetching player team"
	}
	stats.Team = team

	return formatPlayerStatsMessage(stats, nil)

}

func CompareStats(c *gin.Context) string {

	var (
		playerStats [2]PlayerStats
	)

	playerStats[0].Name, playerStats[1].Name = strings.ToLower(c.Query("player1")), strings.ToLower(c.Query("player2"))

	db := ConnectToDatabase()
	defer db.Close()

	for i := 0; i < 2; i++ {

		var stats PlayerStats
		stats = playerStats[i]

		stats, err := getPlayerStats(stats, db)
		if err != nil {
			return "No player stats found."
		}

		stats = calcStatsP10(stats)

		stats, err = getTop3Heroes(stats, db)
		if err != nil {
			return "An error occured while fetching most played heroes"
		}

		team, err := getPlayerTeam(stats.Name, db)
		stats.Team = team
		if err != nil {
			return "Player not found"
		}

		playerStats[i] = stats

	}

	statsDifference := playerStatsDifference(playerStats)

	message := formatCompareMessage(statsDifference, playerStats)

	return message
}

func PStatsHero(c *gin.Context) string {

	var (
		stats PlayerStats
		err   error
	)

	hero := strings.ToLower(strings.ReplaceAll(c.Query("hero"), "_", " "))
	stats.Name = strings.ToLower(c.Query("player"))

	hero = handleWeirdHeroNames(hero)

	db := ConnectToDatabase()
	defer db.Close()

	stats.Team, err = getPlayerTeam(stats.Name, db)

	if err != nil {
		return "Player not found"
	}

	stats, err = getPlayerHeroStats(stats.Name, hero, db)

	if err != nil {
		return "No player stats found for this hero"
	}

	stats = calcStatsP10(stats)

	message := formatPlayerStatsMessage(stats, &hero)

	return message
}

func TStats(c *gin.Context) string {

	var teamStats TeamStats

	teamStats.Team = strings.ToLower(strings.ReplaceAll(c.Query("team"), "_", " "))

	db := ConnectToDatabase()

	teamStats, err := getTeamStats(teamStats, db)
	if err != nil {
		return "No team stats found"
	}

	message := formatTeamStatsMessage(teamStats)

	return message
}

func TStatsMap(c *gin.Context) string {

	var teamStats TeamStats

	teamStats.Team = strings.ToLower(strings.ReplaceAll(c.Query("team"), "_", " "))
	mapName := strings.ToLower(strings.ReplaceAll(c.Query("map"), "_", " "))

	db := ConnectToDatabase()

	teamStats, err := getTeamMapStats(teamStats, mapName, db)
	if err != nil {
		return "No stats found"
	}

	message := formatTeamStatsMessage(teamStats)

	return message
}

func UpdateLeaderboards() string {

	heroLeaderboards := calculateHeroStatLeaderboards()
	generalLeaderboards := calculateGeneralStatLeaderboards()

	if heroLeaderboards && generalLeaderboards {
		return "Leaderboards successfully updated"
	}

	return "Error updating leaderboards"
}

func getTeamMapStats(teamStats TeamStats, mapName string, db *sql.DB) (TeamStats, error) {

	var winner string

	query := "SELECT map.winner FROM map JOIN game ON map.gameID = game.ID WHERE (game.team1 = ? OR game.team2 = ?) AND map.name = ?"

	rows, err := db.Query(query, teamStats.Team, teamStats.Team, mapName)

	if err != nil {
		fmt.Println(err, "getTeamMapStats()")
		return teamStats, err
	}

	defer rows.Close()

	for rows.Next() {
		
		err := rows.Scan(&winner)
		if err != nil {
			fmt.Println(err, "getTeamMapStats()")
			return teamStats, err
		}

		if winner == "draw" {
			teamStats.MapDraws += 1
		} else if winner == teamStats.Team {
			teamStats.MapWins += 1
		} else {
			teamStats.MapLosses += 1
		}
	}

	return teamStats, nil

}

func getTeamStats(teamStats TeamStats, db *sql.DB) (TeamStats, error) {

	var (
		winner  string
		mapName string
	)

	query := "SELECT map.name, map.winner FROM map JOIN game ON map.gameID = game.ID WHERE game.team1 = ? OR game.team2 = ?"

	rows, err := db.Query(query, teamStats.Team, teamStats.Team)

	if err != nil {
		fmt.Println(err, "getTeamStats()")
		return teamStats, err
	}

	defer rows.Close()

	for rows.Next() {

		err := rows.Scan(&mapName, &winner)
		if err != nil {
			return teamStats, err
		}

		found := false
		for i := 0; i < len(teamStats.Maps); i++ {

			if mapName != teamStats.Maps[i].Name {
				continue
			}

			if winner == "draw" {
				teamStats.Maps[i].Draws += 1
				teamStats.MapDraws += 1
				found = true
			} else if winner == teamStats.Team {
				teamStats.Maps[i].Wins += 1
				teamStats.MapWins += 1
				found = true
			} else {
				teamStats.Maps[i].Losses += 1
				teamStats.MapLosses += 1
				found = true
			}

		}
		if !found {

			if winner == "draw" {
				teamStats.Maps = append(teamStats.Maps, MapStats{Name: mapName, Draws: 1, Wins: 0, Losses: 0})
				teamStats.MapDraws += 1
			} else if winner == teamStats.Team {
				teamStats.Maps = append(teamStats.Maps, MapStats{Name: mapName, Draws: 0, Wins: 1, Losses: 0})
				teamStats.MapWins += 1
			} else {
				teamStats.Maps = append(teamStats.Maps, MapStats{Name: mapName, Draws: 0, Wins: 0, Losses: 1})
				teamStats.MapLosses += 1
			}
		}

	}

	return teamStats, nil
}

func formatTeamStatsMessage(stats TeamStats) string {

	var (
		mostPlayedMaps []MapStats
		mp int
		response string
	)

	percent := "%"

	if len(stats.Maps) > 0 {

		var length int

		if len(stats.Maps) < 3 {
			length = len(stats.Maps)
		} else {
			length = 3
		}

		response = "Most played maps:\n"
		for i := 0; i < length; i++ {
			counter := 0
			for j := 0; j < len(stats.Maps); j++ {
				if (stats.Maps[i].Wins + stats.Maps[i].Losses + stats.Maps[i].Draws) > counter {
					mp = i
				}
			}
			mostPlayedMaps = append(mostPlayedMaps, stats.Maps[mp])
			

			wr := (float64(mostPlayedMaps[i].Wins) / float64((mostPlayedMaps[i].Wins + mostPlayedMaps[i].Losses))) * 100

			stats.Maps[mp].Wins, stats.Maps[mp].Losses, stats.Maps[mp].Draws = 0, 0, 0

			response += fmt.Sprintf("%s: %.2f%s W/L\n", capitalizeFirstLetterOfEachWord(mostPlayedMaps[i].Name), wr, percent)
		}
		response += "\n"
	}

	response += fmt.Sprintf("Map Wins: %d\nMap Losses: %d\nMap Draws: %d", stats.MapWins, stats.MapLosses, stats.MapDraws)

	return response
} 

func calculateHeroStatLeaderboards() bool {

	var (
		player         string
		heroStatMaps   [][]map[string]float64
		heroStatArrays [][][]string
	)

	heroes := []string{"ana", "ashe", "baptiste", "bastion", "brigitte", "cassidy", "d.va", "doomfist", "echo", "genji", "hanzo", "illari", "junker queen", "junkrat", "juno", "kiriko", "lifeweaver", "lúcio",
		"mauga", "mei", "mercy", "moira", "orisa", "pharah", "ramattra", "reaper", "reinhardt", "roadhog", "sigma", "sojourn", "soldier: 76", "sombra", "symmetra", "torbjörn", "tracer",
		"venture", "widowmaker", "winston", "wrecking ball", "zarya", "zenyatta"}

	db := ConnectToDatabase()

	for i := 0; i < len(heroes); i++ {

		var statsMap []map[string]float64

		heroStatMaps = append(heroStatMaps, statsMap)
		heroStatMaps[i] = createDicts()
	}

	rows, err := db.Query("SELECT name FROM player")

	if err != nil {
		fmt.Println(err, "calculateHeroStatLeaderboards")
		return false
	}

	defer rows.Close()

	for rows.Next() {

		err := rows.Scan(&player)

		if err != nil {
			fmt.Println(err, "calculateHeroStatLeaderboards")
			return false
		}

		for i := 0; i < len(heroStatMaps); i++ {

			enoughHeroPlaytime := check10MinutesHeroPlaytime(player, heroes[i], db)

			if !enoughHeroPlaytime {
				continue
			}

			heroStatMaps[i], err = putPlayerHeroStatsInMaps(player, heroes[i], heroStatMaps[i], db)

			if err != nil {
				continue
			}
		}
	}

	for i := 0; i < len(heroes); i++ {

		var statArray [][]string

		heroStatArrays = append(heroStatArrays, statArray)
		heroStatArrays[i] = sortDictsIntoArrays(heroStatMaps[i])
	}

	err = saveHeroStatsLeaderboardToJSON(heroStatArrays, "heroLeaderboards.json")

	return err == nil
}

func saveHeroStatsLeaderboardToJSON(arr [][][]string, fileName string) error {

	file, err := json.MarshalIndent(arr, "", "  ")
	if err != nil {
		fmt.Println(err, "saveLeaderboardArraysToJSON()")
		return err
	}

	err = os.WriteFile(fileName, file, 0644)
	if err != nil {
		fmt.Println(err, "saveLeaderboardArraysToJSON()")
		return err
	}

	return nil
}

func loadHeroStatsLeaderboardJSONtoArray(fileName string) ([][][]string, error) {

	var leaderboardArrays [][][]string

	file, err := os.ReadFile(fileName)
	if err != nil {
		fmt.Println(err, "loadLeaderboardJSONtoArray()")
		return leaderboardArrays, err
	}

	err = json.Unmarshal(file, &leaderboardArrays)
	if err != nil {
		fmt.Println(err, "loadLeaderboardJSONtoArray()")
		return leaderboardArrays, err
	}

	return leaderboardArrays, nil

}

func putPlayerHeroStatsInMaps(player string, hero string, statMaps []map[string]float64, db *sql.DB) ([]map[string]float64, error) {

	stats := []string{"damageDealt", "damageTaken", "deaths", "finalBlows", "eliminations", "soloKills", "healingDealt", "environmentalKills", "offensiveAssists", "ultsUsed"}

	for i := 0; i < len(stats); i++ {

		var outputStat float64
		var outputTime int
 
		query := fmt.Sprintf("SELECT %s, durationInSeconds FROM playerHero WHERE player = ? AND hero = ?", stats[i])

		err := db.QueryRow(query, player, hero).Scan(&outputStat, &outputTime)

		if err != nil {
			fmt.Println(err, "putPlayerHeroStatsInMaps()")
			return statMaps, err
		}

		statMaps[i][player] = (outputStat / float64(outputTime)) * 600

	}

	return statMaps, nil
}

func check10MinutesHeroPlaytime(player string, hero string, db *sql.DB) bool {

	var playtime int

	query := "SELECT durationInSeconds FROM playerHero WHERE player = ? AND hero = ?"

	err := db.QueryRow(query, player, hero).Scan(&playtime)
	if err != nil {
		return false
	}
	if playtime < 600 {
		return false
	}
	return true
}

func calculateGeneralStatLeaderboards() bool {

	var player string

	db := ConnectToDatabase()

	leaderboardDicts := createDicts()

	rows, err := db.Query("SELECT name FROM player")

	if err != nil {
		fmt.Println(err, "calculateStatLeaderboards()")
		return false
	}

	for rows.Next() {

		err := rows.Scan(&player)

		if err != nil {
			fmt.Println(err, "calculateStatLeaderboards()")
			return false
		}

		enoughTimePlayed := check30MinutesTotalPlaytime(player, db)

		if !enoughTimePlayed {
			continue
		}

		leaderboardDicts, err = putPlayerStatsInDicts(player, leaderboardDicts, db)

		if err != nil {
			return false
		}

	}

	leaderboardArrays := sortDictsIntoArrays(leaderboardDicts)

	err = saveGeneralLeaderboardArraysToJSON(leaderboardArrays, "leaderboards.json")

	return err == nil

}

func saveGeneralLeaderboardArraysToJSON(arr [][]string, fileName string) error {

	file, err := json.MarshalIndent(arr, "", "  ")
	if err != nil {
		fmt.Println(err, "saveLeaderboardArraysToJSON()")
		return err
	}

	err = os.WriteFile(fileName, file, 0644)
	if err != nil {
		fmt.Println(err, "saveLeaderboardArraysToJSON()")
		return err
	}

	return nil
}

func loadGeneralLeaderboardJSONtoArray(fileName string) ([][]string, error) {

	var leaderboardArrays [][]string

	file, err := os.ReadFile(fileName)
	if err != nil {
		fmt.Println(err, "loadLeaderboardJSONtoArray()")
		return leaderboardArrays, err
	}

	err = json.Unmarshal(file, &leaderboardArrays)
	if err != nil {
		fmt.Println(err, "loadLeaderboardJSONtoArray()")
		return leaderboardArrays, err
	}

	return leaderboardArrays, nil

}

func generalLeaderboardRanks(leaderboards [][]string, player string) []int {

	var ranks []int

	for i := 0; i < len(leaderboards); i++ {

		ranks = append(ranks, 0)

		for j := 0; j < len(leaderboards[i]); j++ {

			if leaderboards[i][j] == player {
				ranks[i] = j + 1
				break
			}

		}

	}

	return ranks

}

func sortDictsIntoArrays(leaderboardDicts []map[string]float64) [][]string {

	var leaderboardArrays [][]string

	for i := 0; i < 10; i++ {

		var stringSlice []string

		leaderboardArrays = append(leaderboardArrays, stringSlice)

		for j := 0; j < 10; j++ {

			maxStat := 0.0
			maxStatPlayer := ""

			for player, stat := range leaderboardDicts[i] {
				if stat <= maxStat {
					continue
				}
				maxStat = stat
				maxStatPlayer = player
			}

			if maxStatPlayer != "" {
				leaderboardArrays[i] = append(leaderboardArrays[i], maxStatPlayer)
			}

			delete(leaderboardDicts[i], maxStatPlayer)
		}
	}

	return leaderboardArrays
}

func putPlayerStatsInDicts(player string, leaderboardDicts []map[string]float64, db *sql.DB) ([]map[string]float64, error) {

	stats := []string{"damageDealt", "damageTaken", "deaths", "finalBlows", "eliminations", "soloKills", "healingDealt", "environmentalKills", "offensiveAssists", "ultsUsed"}

	for i := range stats {

		queryStat := stats[i]
		var outputStat float64
		var outputTime int

		query := fmt.Sprintf("SELECT SUM(%s), SUM(durationInSeconds) FROM mapPlayer WHERE player = ?", queryStat)

		err := db.QueryRow(query, player).Scan(&outputStat, &outputTime)

		if err != nil {
			fmt.Println(err, "putPlayerStatsInDicts()")
			return leaderboardDicts, err
		}

		leaderboardDicts[i][player] = (outputStat / float64(outputTime)) * 600

	}

	return leaderboardDicts, nil

}

func check30MinutesTotalPlaytime(player string, db *sql.DB) bool {
	var durationInSeconds int

	query := "SELECT SUM(durationInSeconds) FROM mapPlayer WHERE player = ?"

	err := db.QueryRow(query, player).Scan(&durationInSeconds)

	if err != nil {
		return false
	}
	if durationInSeconds < 1800 {
		return false
	}
	return true
}

func createDicts() []map[string]float64 {

	var leaderboardDicts []map[string]float64

	damageDealtDict := make(map[string]float64)
	damageTakenDict := make(map[string]float64)
	deathsDict := make(map[string]float64)
	finalBlowsDict := make(map[string]float64)
	eliminationsDict := make(map[string]float64)
	soloKillsDict := make(map[string]float64)
	healingDealtDict := make(map[string]float64)
	environmentalKillsDict := make(map[string]float64)
	offensiveAssistsDict := make(map[string]float64)
	ultsUsedDict := make(map[string]float64)

	// Dict =/= Dick
	leaderboardDicts = append(leaderboardDicts, damageDealtDict)
	leaderboardDicts = append(leaderboardDicts, damageTakenDict)
	leaderboardDicts = append(leaderboardDicts, deathsDict)
	leaderboardDicts = append(leaderboardDicts, finalBlowsDict)
	leaderboardDicts = append(leaderboardDicts, eliminationsDict)
	leaderboardDicts = append(leaderboardDicts, soloKillsDict)
	leaderboardDicts = append(leaderboardDicts, healingDealtDict)
	leaderboardDicts = append(leaderboardDicts, environmentalKillsDict)
	leaderboardDicts = append(leaderboardDicts, offensiveAssistsDict)
	leaderboardDicts = append(leaderboardDicts, ultsUsedDict)

	return leaderboardDicts
}

func handleWeirdHeroNames(hero string) string {

	if hero == "lucio" {
		return "lúcio"
	}
	if hero == "jq" || hero == "queen" || hero == "junkerqueen" || hero == "junker" {
		return "junker queen"
	}
	if hero == "dva" || hero == "d" {
		return "d.va"
	}
	if hero == "ball" || hero == "hammond" || hero == "wreckingball" || hero == "hamster" || hero == "wrecking" {
		return "wrecking ball"
	}
	if hero == "torb" || hero == "torbjorn" {
		return "torbjörn"
	} 
	if hero == "brig" || hero == "briggite" || hero == "briggitte" {
		return "brigitte"
	}
	if hero == "soldier" || hero == "soldier:76" || hero == "soldier76" || hero == "soldier:" {
		return "soldier: 76"
	}

	return hero
}

func getPlayerHeroStats(player string, hero string, db *sql.DB) (PlayerStats, error) {

	var stats PlayerStats

	stats.Name = player

	query := "SELECT damageDealt, damageTaken, deaths, finalBlows, eliminations, soloKills, healingDealt, environmentalKills, offensiveAssists, ultsUsed, durationInSeconds FROM playerHero WHERE player = ? AND hero = ?"

	err := db.QueryRow(query, player, hero).Scan(&stats.DamageDealt, &stats.DamageTaken, &stats.Deaths, &stats.FinalBlows, &stats.Eliminations, &stats.SoloKills, &stats.HealingDealt, &stats.EnvironmentalKills, &stats.OffensiveAssists, &stats.UltsUsed, &stats.DurationInSeconds)

	if err != nil {
		return stats, err
	}

	return stats, nil

}

func formatCompareMessage(statsDifference [10]float64, playerStats [2]PlayerStats) string {

	var (
		message string
		arrows1 []string
		arrows2 []string
	)

	player1 := playerStats[0]
	player2 := playerStats[1]

	for i := 0; i < len(statsDifference); i++ {

		var (
			arrow1 string
			arrow2 string
		)

		if statsDifference[i] > 0 {
			arrow1 = " << "
			arrow2 = "  "
		} else if statsDifference[i] < 0 {
			arrow1 = "  "
			arrow2 = " >> "
		} else {
			arrow1 = "  "
			arrow2 = "  "
		}

		arrows1 = append(arrows1, arrow1)
		arrows2 = append(arrows2, arrow2)
	}

	message = fmt.Sprintf(
		"%s vs %s\n\n",
		player1.Name, player2.Name,
	)

	message += fmt.Sprintf("DD: %.2f %s %.2f %s %.2f\n", player1.DamageDealt, arrows1[0], math.Abs(statsDifference[0]), arrows2[0], player2.DamageDealt)
	message += fmt.Sprintf("DT: %.2f %s %.2f %s %.2f\n", player1.DamageTaken, arrows1[1], math.Abs(statsDifference[1]), arrows2[1], player2.DamageTaken)
	message += fmt.Sprintf("D:  %.2f %s %.2f %s %.2f\n", player1.Deaths, arrows1[2], math.Abs(statsDifference[2]), arrows2[2], player2.Deaths)
	message += fmt.Sprintf("FB: %.2f %s %.2f %s %.2f\n", player1.FinalBlows, arrows1[3], math.Abs(statsDifference[3]), arrows2[3], player2.FinalBlows)
	message += fmt.Sprintf("E:  %.2f %s %.2f %s %.2f\n", player1.Eliminations, arrows1[4], math.Abs(statsDifference[4]), arrows2[4], player2.Eliminations)
	message += fmt.Sprintf("SK: %.2f %s %.2f %s %.2f\n", player1.SoloKills, arrows1[5], math.Abs(statsDifference[5]), arrows2[5], player2.SoloKills)
	message += fmt.Sprintf("HD: %.2f %s %.2f %s %.2f\n", player1.HealingDealt, arrows1[6], math.Abs(statsDifference[6]), arrows2[6], player2.HealingDealt)
	message += fmt.Sprintf("EK: %.2f %s %.2f %s %.2f\n", player1.EnvironmentalKills, arrows1[7], math.Abs(statsDifference[7]), arrows2[7], player2.EnvironmentalKills)
	message += fmt.Sprintf("OA: %.2f %s %.2f %s %.2f\n", player1.OffensiveAssists, arrows1[8], math.Abs(statsDifference[8]), arrows2[8], player2.OffensiveAssists)
	message += fmt.Sprintf("UU: %.2f %s %.2f %s %.2f\n\n", player1.UltsUsed, arrows1[9], math.Abs(statsDifference[9]), arrows2[9], player2.UltsUsed)
	message += "All stats per 10 minutes"

	return message
}

func playerStatsDifference(playerStats [2]PlayerStats) [10]float64 {

	var statsDifference [10]float64

	statsDifference[0] = playerStats[0].DamageDealt - playerStats[1].DamageDealt
	statsDifference[1] = playerStats[0].DamageTaken - playerStats[1].DamageTaken
	statsDifference[2] = playerStats[0].Deaths - playerStats[1].Deaths
	statsDifference[3] = playerStats[0].FinalBlows - playerStats[1].FinalBlows
	statsDifference[4] = playerStats[0].Eliminations - playerStats[1].Eliminations
	statsDifference[5] = playerStats[0].SoloKills - playerStats[1].SoloKills
	statsDifference[6] = playerStats[0].HealingDealt - playerStats[1].HealingDealt
	statsDifference[7] = playerStats[0].EnvironmentalKills - playerStats[1].EnvironmentalKills
	statsDifference[8] = playerStats[0].OffensiveAssists - playerStats[1].OffensiveAssists
	statsDifference[9] = playerStats[0].UltsUsed - playerStats[1].UltsUsed

	return statsDifference
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
		_, err = statement.Exec(mapID, playerName, playerStats[i].DamageDealt, playerStats[i].DamageTaken, playerStats[i].Deaths, playerStats[i].FinalBlows, playerStats[i].Eliminations, playerStats[i].SoloKills, playerStats[i].HealingDealt, playerStats[i].EnvironmentalKills, playerStats[i].OffensiveAssists, playerStats[i].UltsUsed, timePlayed)
		if err != nil {
			fmt.Println(err, "saveStatsToDB() - Executing map player insert")
			continue
		}

		for j := 0; j < len(playerStats[i].Heroes); j++ {
			playerHero := playerStats[i].Heroes[j]
			heroName := strings.ToLower(playerHero.Hero)
			durationInSeconds := playerHero.TimeSpentInSeconds
			damageDealt := playerHero.DamageDealt
			damageTaken := playerHero.DamageTaken
			deaths := playerHero.Deaths
			finalBlows := playerHero.FinalBlows
			eliminations := playerHero.Eliminations
			soloKills := playerHero.SoloKills
			healingDealt := playerHero.HealingDealt
			environmentalKills := playerHero.EnvironmentalKills
			offensiveAssists := playerHero.OffensiveAssists
			ultsUsed := playerHero.UltsUsed

			// Check if player-hero exists
			err := db.QueryRow("SELECT COUNT(*) FROM playerHero WHERE player = ? AND hero = ?", playerName, heroName).Scan(&count)
			if err != nil {
				fmt.Println(err, "saveStatsToDB() - Checking player-hero existence")
				continue
			}

			if count == 0 {
				stmt = `INSERT INTO playerHero (player, hero, damageDealt, damageTaken, deaths, finalBlows, eliminations, soloKills, healingDealt, environmentalKills, offensiveAssists, ultsUsed, durationInSeconds) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
				statement, err := db.Prepare(stmt)
				if err != nil {
					fmt.Println(err, "saveStatsToDB() - Preparing player-hero insert statement")
					continue
				}
				_, err = statement.Exec(playerName, heroName, damageDealt, damageTaken, deaths, finalBlows, eliminations, soloKills, healingDealt, environmentalKills, offensiveAssists, ultsUsed, durationInSeconds)
				if err != nil {
					fmt.Println(err, "saveStatsToDB() - Executing player-hero insert")
					continue
				}
			} else {
				_, err := db.Exec("UPDATE playerHero SET damageDealt = damageDealt + ? AND damageTaken = damageTaken + ? AND deaths = deaths + ? AND finalBlows = finalBlows + ? AND eliminations = eliminations + ? AND soloKills = soloKills + ? AND healingDealt = healingDealt + ? AND environmentalKills = environmentalKills + ? AND offensiveAssists = offensiveAssists + ? AND ultsUsed = ultsUsed + ? AND durationInSeconds = durationInSeconds + ? WHERE player = ? AND hero = ?",
					damageDealt, damageTaken, deaths, finalBlows, eliminations, soloKills, healingDealt, environmentalKills, offensiveAssists, ultsUsed, durationInSeconds, playerName, heroName)
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

func getPlayerStats(totalStats PlayerStats, db *sql.DB) (PlayerStats, error) {

	rows, err := db.Query("SELECT damageDealt, damageTaken, deaths, finalBlows, eliminations, soloKills, healingDealt, environmentalKills, offensiveAssists, ultsUsed, durationInSeconds FROM mapPlayer WHERE player = ?", totalStats.Name)

	if err != nil {
		return totalStats, err
	}

	defer rows.Close()

	for rows.Next() {
		var stats PlayerStats
		err := rows.Scan(&stats.DamageDealt, &stats.DamageTaken, &stats.Deaths, &stats.FinalBlows, &stats.Eliminations, &stats.SoloKills, &stats.HealingDealt, &stats.EnvironmentalKills, &stats.OffensiveAssists, &stats.UltsUsed, &stats.DurationInSeconds)
		if err != nil {
			fmt.Println(err, "getPlayerStats()")
			return totalStats, err
		}

		totalStats.DurationInSeconds += stats.DurationInSeconds

		totalStats.DamageDealt += stats.DamageDealt
		totalStats.DamageTaken += stats.DamageTaken
		totalStats.Deaths += stats.Deaths
		totalStats.FinalBlows += stats.FinalBlows
		totalStats.Eliminations += stats.Eliminations
		totalStats.SoloKills += stats.SoloKills
		totalStats.HealingDealt += stats.HealingDealt
		totalStats.EnvironmentalKills += stats.EnvironmentalKills
		totalStats.OffensiveAssists += stats.OffensiveAssists
		totalStats.UltsUsed += stats.UltsUsed
	}

	if err = rows.Err(); err != nil {
		fmt.Println(err, "getPlayerStats()")
		return totalStats, err
	}

	return totalStats, nil
}

func calcStatsP10(stats PlayerStats) PlayerStats {

	stats.DamageDealt = stats.DamageDealt / float64(stats.DurationInSeconds) * 600
	stats.DamageTaken = stats.DamageTaken / float64(stats.DurationInSeconds) * 600
	stats.Deaths = stats.Deaths / float64(stats.DurationInSeconds) * 600
	stats.FinalBlows = stats.FinalBlows / float64(stats.DurationInSeconds) * 600
	stats.Eliminations = stats.Eliminations / float64(stats.DurationInSeconds) * 600
	stats.SoloKills = stats.SoloKills / float64(stats.DurationInSeconds) * 600
	stats.HealingDealt = stats.HealingDealt / float64(stats.DurationInSeconds) * 600
	stats.EnvironmentalKills = stats.EnvironmentalKills / float64(stats.DurationInSeconds) * 600
	stats.OffensiveAssists = stats.OffensiveAssists / float64(stats.DurationInSeconds) * 600
	stats.UltsUsed = stats.UltsUsed / float64(stats.DurationInSeconds) * 600

	return stats
}

func getTop3Heroes(stats PlayerStats, db *sql.DB) (PlayerStats, error) {

	sql := `SELECT hero, durationInSeconds FROM playerHero WHERE player = ? ORDER BY durationInSeconds DESC LIMIT 3`

	rows, err := db.Query(sql, stats.Name)

	if err != nil {
		fmt.Println(err, "getTop3Heroes()")
		return stats, err
	}

	defer rows.Close()

	for rows.Next() {
		var heroStats HeroStats
		err := rows.Scan(&heroStats.Hero, &heroStats.TimeSpentInSeconds)
		if err != nil {
			fmt.Println(err, "getTop3Heroes()")
			return stats, err
		}
		stats.Heroes = append(stats.Heroes, heroStats)
	}

	return stats, nil
}

func getPlayerTeam(player string, db *sql.DB) (string, error) {

	var team string

	err := db.QueryRow("SELECT team FROM player WHERE name = ?", player).Scan(&team)
	if err != nil {
		fmt.Println(err, "getPlayerTeam()")
		return team, err
	}
	return team, nil
}

func formatPlayerStatsMessage(stats PlayerStats, heroPointer *string) string {

	var (
		heroInfo string
		ranks    []int
	)

	heroes := []string{"ana", "ashe", "baptiste", "bastion", "brigitte", "cassidy", "d.va", "doomfist", "echo", "genji", "hanzo", "illari", "junker queen", "junkrat", "juno", "kiriko", "lifeweaver", "lúcio",
		"mauga", "mei", "mercy", "moira", "orisa", "pharah", "ramattra", "reaper", "reinhardt", "roadhog", "sigma", "sojourn", "soldier: 76", "sombra", "symmetra", "torbjörn", "tracer",
		"venture", "widowmaker", "winston", "wrecking ball", "zarya", "zenyatta"}

	if len(stats.Heroes) > 0 {
		heroInfo = fmt.Sprintf("Team: %s\n\nMost Played Heroes:\n", capitalizeFirstLetterOfEachWord(stats.Team))
		for i := range stats.Heroes {

			var secondsString string
			var minutesString string

			minutes := stats.Heroes[i].TimeSpentInSeconds / 60
			if minutes < 10 {
				minutesString = fmt.Sprintf("0%d", minutes)
			} else {
				minutesString = fmt.Sprintf("%d", minutes)
			}

			seconds := stats.Heroes[i].TimeSpentInSeconds % 60
			if seconds < 10 {
				secondsString = fmt.Sprintf("0%d", seconds)
			} else {
				secondsString = fmt.Sprintf("%d", seconds)
			}

			heroInfo += fmt.Sprintf("%d. %s %s:%s\n", i+1, capitalizeFirstLetterOfEachWord(stats.Heroes[i].Hero), minutesString, secondsString)
		}

		leaderboards, err := loadGeneralLeaderboardJSONtoArray("leaderboards.json")
		if err != nil {
			return "Error loading leaderboards.json"
		}

		ranks = generalLeaderboardRanks(leaderboards, stats.Name)

		return fmt.Sprintf(
			"%s\n"+
				"Damage Dealt: %.2f - %d/%d\n"+
				"Damage Taken: %.2f - %d/%d\n"+
				"Deaths: %.2f - %d/%d\n"+
				"Final Blows: %.2f - %d/%d\n"+
				"Eliminations: %.2f - %d/%d\n"+
				"Solo Kills: %.2f - %d/%d\n"+
				"Healing Dealt: %.2f - %d/%d\n"+
				"Environmental Kills: %.2f - %d/%d\n"+
				"Offensive Assists: %.2f - %d/%d\n"+
				"Ultimates Used: %.2f - %d/%d\n\n"+
				"All Stats per 10 minutes",
			heroInfo,
			stats.DamageDealt, ranks[0], len(leaderboards[0]),
			stats.DamageTaken, ranks[1], len(leaderboards[1]),
			stats.Deaths, ranks[2], len(leaderboards[2]),
			stats.FinalBlows, ranks[3], len(leaderboards[3]),
			stats.Eliminations, ranks[4], len(leaderboards[4]),
			stats.SoloKills, ranks[5], len(leaderboards[5]),
			stats.HealingDealt, ranks[6], len(leaderboards[6]),
			stats.EnvironmentalKills, ranks[7], len(leaderboards[7]),
			stats.OffensiveAssists, ranks[8], len(leaderboards[8]),
			stats.UltsUsed, ranks[9], len(leaderboards[9]),
		)

	} else {

		if heroPointer == nil {
			return "Error fetching hero"
		}

		leaderboards, err := loadHeroStatsLeaderboardJSONtoArray("heroLeaderboards.json")
		if err != nil {
			return "Error loading heroLeaderboards.json"
		}

		heroIndex := findIndexInSlice(heroes, *heroPointer)

		ranks := generalLeaderboardRanks(leaderboards[heroIndex], stats.Name)

		return fmt.Sprintf(
			"%s\n"+
				"Damage Dealt: %.2f - %d/%d\n"+
				"Damage Taken: %.2f - %d/%d\n"+
				"Deaths: %.2f - %d/%d\n"+
				"Final Blows: %.2f - %d/%d\n"+
				"Eliminations: %.2f - %d/%d\n"+
				"Solo Kills: %.2f - %d/%d\n"+
				"Healing Dealt: %.2f - %d/%d\n"+
				"Environmental Kills: %.2f - %d/%d\n"+
				"Offensive Assists: %.2f - %d/%d\n"+
				"Ultimates Used: %.2f - %d/%d\n\n"+
				"All Stats per 10 minutes",
			heroInfo,
			stats.DamageDealt, ranks[0], len(leaderboards[heroIndex][0]),
			stats.DamageTaken, ranks[1], len(leaderboards[heroIndex][1]),
			stats.Deaths, ranks[2], len(leaderboards[heroIndex][2]),
			stats.FinalBlows, ranks[3], len(leaderboards[heroIndex][3]),
			stats.Eliminations, ranks[4], len(leaderboards[heroIndex][4]),
			stats.SoloKills, ranks[5], len(leaderboards[heroIndex][5]),
			stats.HealingDealt, ranks[6], len(leaderboards[heroIndex][6]),
			stats.EnvironmentalKills, ranks[7], len(leaderboards[heroIndex][7]),
			stats.OffensiveAssists, ranks[8], len(leaderboards[heroIndex][8]),
			stats.UltsUsed, ranks[9], len(leaderboards[heroIndex][9]),
		)
	}
}

func readFile(fileName string) ([10]PlayerStats, Map, error) {

	var (
		lines       []string
		playerStats PlayerStats
		players     [10]PlayerStats
		prevLines   [10]string
		playedMap   Map
	)

	file, err := os.Open(fileName)
	if err != nil {
		fmt.Println("Error opening file:", err)
		return players, playedMap, err
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
			if currSeconds <= prevSeconds+10 {
				totalTimeInSeconds += 5
				setupPhase = false
			} else {
				setupPhase = true
			}
			prevSeconds = currSeconds
		}

		if setupPhase {
			prevLines[(lineCount-1)%10] = line
		}

		if !setupPhase && lineCount > 10 {

			playerStats, updatedPrevLines := getHeroStats(line, players, prevLines, (lineCount-1)%10)

			for j := 0; j < len(players); j++ {
				if players[j].Name == playerStats.Name {
					players[j] = playerStats
					break
				}
			}
			prevLines = updatedPrevLines

		}
	}

	start := len(lines) - 10
	lines = lines[start:]

	for i := 0; i < len(lines); i++ {
		playerStats = getEndOfGamePlayerStats(lines[i], players, totalTimeInSeconds)
		for j := 0; j < len(players); j++ {
			if players[j].Name == playerStats.Name {
				players[j] = playerStats
			}
		}
	}

	playedMap.TotalTimeInSeconds = totalTimeInSeconds

	if err := scanner.Err(); err != nil {
		fmt.Println("Error reading file:", err)
		return players, playedMap, err
	}

	return players, playedMap, nil
}

func getEndOfGamePlayerStats(line string, players [10]PlayerStats, timePlayed int) PlayerStats {

	stats := strings.Split(line, ",")
	playerName := stats[1]
	var player PlayerStats

	for i := 0; i < len(players); i++ {
		player = players[i]
		if player.Name == playerName {

			player.DamageDealt, _ = strconv.ParseFloat(stats[3], 64)
			player.DamageTaken, _ = strconv.ParseFloat(stats[4], 64)
			player.Deaths, _ = strconv.ParseFloat(stats[5], 64)
			player.FinalBlows, _ = strconv.ParseFloat(stats[6], 64)
			player.Eliminations, _ = strconv.ParseFloat(stats[7], 64)
			player.SoloKills, _ = strconv.ParseFloat(stats[8], 64)
			player.HealingDealt, _ = strconv.ParseFloat(stats[9], 64)
			player.EnvironmentalKills, _ = strconv.ParseFloat(stats[10], 64)
			player.OffensiveAssists, _ = strconv.ParseFloat(stats[11], 64)
			player.UltsUsed, _ = strconv.ParseFloat(stats[12], 64)
			player.Team = stats[13]
			player.DurationInSeconds = timePlayed

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

func getHeroStats(line string, players [10]PlayerStats, prevLines [10]string, index int) (PlayerStats, [10]string) {
	stats := strings.Split(line, ",")
	playerName := stats[1]
	hero := stats[2]
	var player PlayerStats

	for i := 0; i < len(players); i++ {
		player = players[i]
		if player.Name == playerName {
			found := false
			stats := subtractStats(line, prevLines[index])
			for j := 0; j < len(player.Heroes); j++ {
				if player.Heroes[j].Hero == hero {
					player.Heroes[j].TimeSpentInSeconds += 5
					player.Heroes[j] = addHeroStats(player.Heroes[j], stats)
					found = true
					break
				}
			}
			if !found {
				player.Heroes = append(player.Heroes, HeroStats{Hero: hero, TimeSpentInSeconds: 0})
			}

			prevLines[index] = line

			return player, prevLines
		}
	}
	return player, prevLines
}

func addHeroStats(playerHero HeroStats, stats [10]float64) HeroStats {

	playerHero.DamageDealt += stats[0]
	playerHero.DamageTaken += stats[1]
	playerHero.Deaths += stats[2]
	playerHero.FinalBlows += stats[3]
	playerHero.Eliminations += stats[4]
	playerHero.SoloKills += stats[5]
	playerHero.HealingDealt += stats[6]
	playerHero.EnvironmentalKills += stats[7]
	playerHero.OffensiveAssists += stats[8]
	playerHero.UltsUsed += stats[9]

	return playerHero

}

func subtractStats(newLine string, oldLine string) [10]float64 {

	newStats := stringStatsToFloatStats(strings.Split(newLine, ","))
	oldStats := stringStatsToFloatStats(strings.Split(oldLine, ","))

	for i := 0; i < 10; i++ {
		newStats[i] = newStats[i] - oldStats[i]
	}

	return newStats
}

func stringStatsToFloatStats(stringStats []string) [10]float64 {

	var floatStats [10]float64

	for i := 3; i <= 12; i++ {
		floatStats[i-3], _ = strconv.ParseFloat(stringStats[i], 64)
	}

	return floatStats
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

func capitalizeFirstLetterOfEachWord(str string) string {

	var outStr string

	strArr := strings.Split(str, " ")

	for i := 0; i < len(strArr); i++ {
		firstChar := unicode.ToUpper(rune(strArr[i][0]))
		outStr += string(firstChar) + strArr[i][1:] + " "
	}

	return strings.Trim(outStr, " ")

}

func findIndexInSlice[T comparable](slice []T, item T) int {
	for i := 0; i < len(slice); i++ {
		if slice[i] == item {
			return i
		}
	}
	return -1
}
