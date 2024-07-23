package main

type HeroStats struct {
	Hero               string
	TimeSpentInSeconds int
}

type Map struct {
	Name               string
	Winner             string
	TotalTimeInSeconds int
	MatchID            int
}

type PlayerStats struct {
	Name                  string
	Team                  string
	DamageDealtP10        float64
	DamageTakenP10        float64
	DeathsP10             float64
	FinalBlowsP10         float64
	EliminationsP10       float64
	SoloKillsP10          float64
	HealingDealtP10       float64
	EnvironmentalKillsP10 float64
	OffensiveAssistsP10   float64
	UltsUsedP10           float64
	Heroes                []HeroStats
}

	
type Stats struct {
	DamageDealt float64
	DamageTaken float64
	Deaths float64
	FinalBlows float64
	Eliminations float64
	SoloKills float64
	HealingDealt float64
	EnvironmentalKills float64
	OffensiveAssists float64
	UltsUsed float64
	DurationInSeconds int
	TopHeroes []HeroStats
}