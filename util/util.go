package util

import (
	"fmt"

	"github.com/knightpp/mal-api/mal"
)

func LongTitle(anime *mal.Anime) string {
	titles := anime.AlternativeTitles
	if titles.En != "" && titles.En != anime.Title {
		return fmt.Sprintf("%s (%s)", titles.En, anime.Title)
	}
	return anime.Title
}

func PrefTitle(anime *mal.Anime) string {
	titles := anime.AlternativeTitles
	if titles.En != "" {
		return titles.En
	}
	return anime.Title
}
