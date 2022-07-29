package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/algolia/algoliasearch-client-go/v3/algolia/search"
	"github.com/oliveagle/jsonpath"
	"github.com/spf13/cobra"
)

type Recommend struct {
	FacetName string              `json:"facetName"`
	FBT       map[string][]string `json:"FBT"`
	Trends    Trends              `json:"trends"`
}

type Trends struct {
	TrendingItems       []string `json:"trendingItems"`
	TrendingFacetValues []string `json:"trendingFacetValues"`
}

type Config struct {
	facetName    string
	searchIndex  *search.Index
	searchClient *search.Client
	seperator    string
	outputFile   *os.File
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

func gen(cfg *Config) error {
	res, err := cfg.searchIndex.BrowseObjects()
	if err != nil {
		return err
	}

	pat, err := jsonpath.Compile(cfg.facetName)
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
		return fmt.Errorf("No items were loaded. Is the provided facetName correct? (%s)", cfg.facetName)
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
			neighbors := neighbors(unusedFacets, cfg.seperator)

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
		FacetName: cfg.facetName,
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

	_, err = fmt.Fprintf(cfg.outputFile, string(b))
	if err != nil {
		return err
	}
	return nil
}

func main() {
	var seed int64
	var appId, apiKey, indexName, facetName, outputFileName, seperator string

	cmd := &cobra.Command{
		Use:   "figgen",
		Short: "Generate fig's recommend.json",
		RunE: func(cmd *cobra.Command, args []string) error {
			if appId == "" || apiKey == "" || indexName == "" {
				return fmt.Errorf("missing required flags: app-id, api-key, index-name")
			}
			var outputFile *os.File
			if outputFileName == "" {
				outputFile = os.Stdout
			} else {
				f, err := os.Create(outputFileName)
				if err != nil {
					return err
				}
				outputFile = f
				defer outputFile.Close()
			}

			rand.Seed(seed)
			searchClient := search.NewClient(appId, apiKey)
			searchIndex := searchClient.InitIndex(indexName)
			cfg := &Config{
				facetName:    facetName,
				searchIndex:  searchIndex,
				searchClient: searchClient,
				seperator:    seperator,
				outputFile:   outputFile,
			}
			return gen(cfg)
		},
	}

	cmd.Flags().Int64Var(&seed, "seed", time.Now().UnixNano(), "Specify a seed for the random number generator")
	cmd.Flags().StringVarP(&appId, "app-id", "a", "", "Algolia Application ID")
	cmd.Flags().StringVarP(&apiKey, "api-key", "k", "", "Algolia API key")
	cmd.Flags().StringVarP(&indexName, "index-name", "i", "", "Index name")
	cmd.Flags().StringVarP(&facetName, "facet-name", "f", "$.hierarchical_categories.lvl2", "Facet name for FBT categorizations in jsonpath format. You must use the Hierarchical Categories facet.")
	cmd.Flags().StringVarP(&seperator, "seperator", "s", ">", "Hierearchical categories seperator")
	cmd.Flags().StringVarP(&outputFileName, "output", "o", "", "Output file name")
	cmd.Execute()
}
