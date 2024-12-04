package v2

import (
	"encoding/json"
	"fmt"

	"github.com/r3quie/provision-crawler"
)

/* TODO:	LINT
		METHOD TO CRAWLER
		MOVE CRAWLER TO V2
*/

type DuoBool struct {
	search bool
	term     bool
}

func isSubset(main, sub []string) bool {
	elementCount := make(map[string]int)
	for _, v := range main {
		elementCount[v]++
	}
	for _, v := range sub {
		if elementCount[v] == 0 {
			return false
		}
		elementCount[v]--
	}
	return true
}

func findInJson(f []byte, terms []string, animal string, podnikatel DuoBool, gender DuoBool, rozhodnuti DuoBool) ([]crawler.Rozh, error) {
	var v []crawler.Rozh
	json.Unmarshal(f, &v)
	var found []crawler.Rozh
	for _, x := range v {
		if !isSubset(x.Provisions, terms) {
			continue
		}
		if podnikatel.search && podnikatel.term != x.Podnikatel {
			continue
		}
		if gender.search && gender.term != x.Male {
			continue
		}
		if rozhodnuti.search && rozhodnuti.term != x.Rozhodnuti {
			continue
		}
		if !strings.HasSuffix(x.Path, animal) {
			continue
		}
		
		found = append(found, x)
	}
	return found, nil
}
