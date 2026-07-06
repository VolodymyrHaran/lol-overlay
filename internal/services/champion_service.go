package services

var champions = map[int]string{
	103: "Ahri",
	222: "Jinx",
	64:  "Lee Sin",
}

func GetChampionName(championId int) string {
	if name, ok := champions[championId]; ok {
		return name
	}

	return "Unknown"
}
