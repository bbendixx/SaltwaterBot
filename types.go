package main

type HeroStats struct {
	Hero               string
	TimeSpentInSeconds int
	DamageDealt        float64
	DamageTaken        float64
	Deaths             float64
	FinalBlows         float64
	Eliminations       float64
	SoloKills          float64
	HealingDealt       float64
	EnvironmentalKills float64
	OffensiveAssists   float64
	UltsUsed           float64
}

type Map struct {
	Name               string
	Winner             string
	TotalTimeInSeconds int
	MatchID            int
}

type PlayerStats struct {
	Name               string
	Team               string
	DurationInSeconds  int
	DamageDealt        float64
	DamageTaken        float64
	Deaths             float64
	FinalBlows         float64
	Eliminations       float64
	SoloKills          float64
	HealingDealt       float64
	EnvironmentalKills float64
	OffensiveAssists   float64
	UltsUsed           float64
	Heroes             []HeroStats
}

type TeamStats struct {
	Team string
	MapWins int
	MapLosses int
	MapDraws int
	Maps []MapStats
}

type MapStats struct {
	Name string
	Wins int
	Losses int
	Draws int
}