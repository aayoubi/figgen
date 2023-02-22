package recommend

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"reflect"
	"sort"
	"strings"

	"github.com/algolia/algoliasearch-client-go/v3/algolia/search"
	"github.com/oliveagle/jsonpath"
)

type Config struct {
	FacetName    string
	SearchIndex  *search.Index
	SearchClient *search.Client
	Seperator    string
	OutputFile   *os.File
}

type Recommend struct {
	FacetName string              `json:"facetName"`
	FBT       map[string][]string `json:"FBT"`
	Trends    Trends              `json:"trends"`
}

type Trends struct {
	TrendingItems       []string `json:"trendingItems"`
	TrendingFacetValues []string `json:"trendingFacetValues"`
}

func neighbors(facets map[string]bool, sep string) map[int][]string {
	index := -1
	buckets := make(map[string]int)
	hash := func(k string) int {
		if _, ok := buckets[k]; ok {
			return buckets[k]
		} else {
			index++
			buckets[k] = index
			return buckets[k]
		}
	}
	neighbors := make(map[int][]string, 0)
	for key := range facets {
		parent := strings.Split(key, sep)[0] // woah what are you doing mate
		bucket := hash(parent)
		neighbors[bucket] = append(neighbors[bucket], key)
	}
	return neighbors
}

func Gen(cfg *Config) error {
	res, err := cfg.SearchIndex.BrowseObjects()
	if err != nil {
		return err
	}

	pat, err := jsonpath.Compile(cfg.FacetName)
	if err != nil {
		return err
	}

	items := make([]map[string]interface{}, 0)
	facets := make([]string, 0)
	itemsByFacet := make(map[string]int, 0)
	unusedFacets := make(map[string]bool, 0)

	for {
		obj, err := res.Next()
		if err != nil {
			break
		}
		objM := obj.(map[string]interface{})
		res, _ := pat.Lookup(objM)
		facetValue, ok := res.(string)
		if !ok {
			continue
		}
		items = append(items, map[string]interface{}{
			"objectID": objM["objectID"],
			"facet":    facetValue,
		})
		if _, ok := itemsByFacet[facetValue]; ok {
			itemsByFacet[facetValue] += 1
		} else {
			itemsByFacet[facetValue] = 1
			facets = append(facets, facetValue)
			unusedFacets[facetValue] = true
		}
	}

	if len(items) == 0 {
		return fmt.Errorf("No items were loaded. Is the provided facetName correct? (%s)", cfg.FacetName)
	}

	fbt := map[string][]string{}
	for len(unusedFacets) > 0 {
		// select random facet
		keys := reflect.ValueOf(unusedFacets).MapKeys()
		parentFacet := keys[rand.Intn(len(keys))].Interface().(string) // don't do this :)
		delete(unusedFacets, parentFacet)
		if len(unusedFacets) == 0 {
			break
		}

		for i := 0; i < 2; i++ {
			// re-build nearest "neighbors"
			neighbors := neighbors(unusedFacets, cfg.Seperator)

			// select a random facet from nearest neighbor
			// if not found, select random nodes ???
			// how to handle non-hierarchy facets ???
			b := 0 // start with nearest bucket
			for len(neighbors[b]) == 0 {
				b++ // go to next bucket when empty
			}
			child := neighbors[b][rand.Intn(len(neighbors[b]))]
			fbt[parentFacet] = append(fbt[parentFacet], child)
			delete(unusedFacets, child)
			if len(unusedFacets) == 0 {
				break
			}
		}
	}

	// get X random objects as trending items
	var trendingItems []string
	for i := 0; i < 12; i++ {
		trendingItems = append(trendingItems, items[rand.Intn(len(items))]["objectID"].(string))
	}

	// order list of facets by number of items
	// use for trending facet values
	sort.SliceStable(facets, func(i, j int) bool {
		return itemsByFacet[facets[i]] > itemsByFacet[facets[j]]
	})

	recommend := &Recommend{
		FacetName: cfg.FacetName,
		Trends: Trends{
			TrendingItems:       trendingItems,
			TrendingFacetValues: facets[5:10],
		},
		FBT: fbt,
	}

	b, err := json.Marshal(recommend)
	if err != nil {
		return err
	}

	_, err = fmt.Fprintf(cfg.OutputFile, string(b))
	if err != nil {
		return err
	}
	return nil
}
