package qcat

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"regexp"
	"sort"
	"strings"
	"unicode"

	"github.com/algolia/algoliasearch-client-go/v3/algolia/opt"
	"github.com/algolia/algoliasearch-client-go/v3/algolia/search"
	"github.com/oliveagle/jsonpath"
)

type Config struct {
	SearchIndex  *search.Index
	SearchClient *search.Client
	OutputFile   *os.File
	Combinations [][]string
	StopWords    map[string]bool
}

type searchTerm struct {
	Term             string  `json:"term"`
	ClickThroughRate float64 `json:"click_through_rate,omitempty"`
	ConversionRate   float64 `json:"conversion_rate,omitempty"`
	ClickPosition    int     `json:"click_position,omitempty"`
}

const (
	mathAlphaSymbolsStart rune = 0x1D400
	mathAlphaSymbolsEnd   rune = 0x1D800
)

var specialCharRegexp = regexp.MustCompile(`[^\p{L}\s0-9]`)

func NewConfig(appId, apiKey, indexName, outputFileName, language string, facetCombinations []string) (*Config, error) {
	if appId == "" || apiKey == "" || indexName == "" {
		return nil, fmt.Errorf("missing app-id, api-key, index-name")
	}
	searchClient := search.NewClient(appId, apiKey)
	searchIndex := searchClient.InitIndex(indexName)

	var outputFile *os.File
	if outputFileName == "" {
		outputFile = os.Stdout
	} else {
		f, err := os.Create(outputFileName)
		if err != nil {
			return nil, err
		}
		outputFile = f
		defer outputFile.Close()
	}

	var combinations [][]string
	for _, facet := range facetCombinations {
		combinations = append(combinations, strings.Split(facet, ","))
	}

	var stopWords map[string]bool = make(map[string]bool)
	file, err := os.Open(fmt.Sprintf("stopwords/%s.txt", language))
	if err != nil {
		log.Printf("Unable to read stop words file under `stopwords/%s.txt - skipping", language)
	}
	defer file.Close()
	scn := bufio.NewScanner(file)
	for scn.Scan() {
		stopWords[scn.Text()] = true
	}

	return &Config{
		SearchIndex:  searchIndex,
		SearchClient: searchClient,
		OutputFile:   outputFile,
		Combinations: combinations,
		StopWords:    stopWords,
	}, nil
}

func containsMathAlphaSymbols(query string) bool {
	for _, runeValue := range query {
		if mathAlphaSymbolsStart <= runeValue && runeValue < mathAlphaSymbolsEnd {
			return true
		}
	}
	return false
}

func skipSpecialIfSpecialChars(query string) bool {
	return specialCharRegexp.MatchString(query) || containsMathAlphaSymbols(query)
}

func facetValues(s *search.Index, facet, filters string) ([]string, error) {
	var values []string
	res, err := s.Search("",
		opt.HitsPerPage(0),
		opt.AttributesToRetrieve(),
		opt.AttributesToHighlight(),
		opt.AttributesToSnippet(),
		opt.Facets(facet),
		opt.Filters(filters),
		opt.MaxValuesPerFacet(1000),
	)
	if err != nil {
		return nil, fmt.Errorf("unable to get facet values for facet %q, having filters %q: %w", facet, filters, err)
	}

	val, ok := res.Facets[facet]
	if !ok {
		return values, nil
	}

	for k := range val {
		values = append(values, k)
	}
	return values, nil
}

func generateQueriesUsingFacetCombinations(cfg *Config) ([]string, error) {
	var searches []string
	for _, combination := range cfg.Combinations {
		values, err := generateFacetCombination(cfg.SearchIndex, combination, 0, "")
		if err != nil {
			return nil, err
		}
		for _, v := range values {
			searches = append(searches, v)
		}
	}
	return searches, nil
}

func generateFacetCombination(s *search.Index, facets []string, i int, filters string) ([]string, error) {
	name := facets[i]
	values, err := facetValues(s, name, filters)
	if err != nil {
		return nil, err
	}
	if len(facets) == i+1 {
		return values, nil
	}
	var res []string
	for _, val := range values {
		if skipSpecialIfSpecialChars(val) {
			continue
		}
		newFilters := filters
		if newFilters != "" {
			newFilters += " AND "
		}
		newFilters += fmt.Sprintf(`"%s":"%s"`, name, val)
		nested, err := generateFacetCombination(s, facets, i+1, newFilters)
		if err != nil {
			return nil, err
		}
		for _, nestedVal := range nested {
			res = append(res, fmt.Sprintf("%s %s", val, nestedVal))
		}
	}
	return res, nil
}

func normalizeSearchableAttributes(attributes []string) []string {
	for i, attrib := range attributes {
		attrib = attrib[strings.LastIndex(attrib, "(")+1 : strings.LastIndex(attrib, ")")]
		attributes[i] = attrib
	}
	return attributes
}

func normalize(value string) string {
	var res []rune
	for _, r := range value {
		if unicode.IsLetter(r) || unicode.IsNumber(r) || unicode.IsSpace(r) {
			res = append(res, r)
		}
	}
	return string(res)
}

func generateQueriesUsingSearchableAttributesCombinations(cfg *Config) ([]string, error) {
	settings, err := cfg.SearchIndex.GetSettings()
	if err != nil {
		return nil, err
	}

	searchableAttributes := normalizeSearchableAttributes(settings.SearchableAttributes.Get())
	patterns := make(map[string]*jsonpath.Compiled)
	for _, attrib := range searchableAttributes {
		pat, err := jsonpath.Compile(fmt.Sprintf("$.%s", attrib))
		if err != nil {
			return nil, err
		}
		patterns[attrib] = pat
	}

	res, err := cfg.SearchIndex.BrowseObjects()
	if err != nil {
		return nil, err
	}

	var terms []string
	for {
		obj, err := res.Next()
		if err != nil {
			break
		}
		objM := obj.(map[string]interface{})
		var query []string
		for _, searchableAttribute := range searchableAttributes {
			res, _ := patterns[searchableAttribute].Lookup(objM)
			value, ok := res.(string)
			if !ok || value == "" {
				continue
			}
			query = append(query, strings.ToLower(normalize(value)))
		}
		terms = append(terms, strings.Join(query, " "))
	}

	return generateQueryCombinations(terms, 10, 5, cfg.StopWords), nil
}

func contains(slice []string, s string) bool {
	for _, elem := range slice {
		if elem == s {
			return true
		}
	}
	return false
}

func generateQueryCombinations(terms []string, threshold int, queryMaxLength int, stopWords map[string]bool) []string {
	wordCount := make(map[string]int)
	for _, sentence := range terms {
		words := strings.Fields(sentence)
		for _, word := range words {
			wordCount[word]++
		}
	}

	for k, v := range wordCount {
		if v < threshold || stopWords[k] {
			delete(wordCount, k)
		}
	}

	freqList := make([]string, 0, len(wordCount))
	for word := range wordCount {
		freqList = append(freqList, word)
	}
	sort.Slice(freqList, func(i, j int) bool {
		return wordCount[freqList[i]] > wordCount[freqList[j]]
	})

	var result []string
	var generatedCombos map[string]bool = make(map[string]bool)
	for _, sentence := range terms {
		words := strings.Split(sentence, " ")
		for i := 0; i < len(words); i++ {
			for j := i + 1; j <= len(words) && j <= i+1+5; j++ {
				subWords := words[i:j]
				sentenceWords := make([]string, 0)
				for i, word := range subWords {
					if _, ok := wordCount[word]; ok && !contains(subWords[i+1:], word) {
						sentenceWords = append(sentenceWords, word)
					}
				}
				query := strings.Join(sentenceWords, " ")
				if !generatedCombos[query] && query != "" {
					result = append(result, query)
					generatedCombos[query] = true
				}
			}
		}
	}
	return result
}

func Gen(cfg *Config) error {
	// settings, err := cfg.SearchIndex.GetSettings()
	// if err != nil {
	// 	return err
	// }
	// log.Printf("%s attributesForFaceting: %+q\n", cfg.SearchIndex.GetName(), settings.AttributesForFaceting.Get())
	// log.Printf("%s searchableAttributes: %+q\n", cfg.SearchIndex.GetName(), settings.SearchableAttributes.Get())

	var searchTerms []*searchTerm

	searchesFromFacets, err := generateQueriesUsingFacetCombinations(cfg)
	if err != nil {
		return err
	}
	for _, s := range searchesFromFacets {
		searchTerms = append(searchTerms, &searchTerm{
			Term:             s,
			ClickThroughRate: 100,
			ConversionRate:   15,
			ClickPosition:    3,
		})
	}

	searchesFromCombinations, err := generateQueriesUsingSearchableAttributesCombinations(cfg)
	if err != nil {
		return err
	}
	for _, s := range searchesFromCombinations {
		searchTerms = append(searchTerms, &searchTerm{
			Term:             s,
			ClickThroughRate: 100,
			ConversionRate:   15,
			ClickPosition:    3,
		})
	}

	if len(searchTerms) < 1000 {
		log.Printf("Not enough unique queries [%d] were generated, we need at least 1000 for qcat]!\n", len(searchTerms))
		return fmt.Errorf("Please specify additional facet combinations to generate more than 1000 unique search queries.")
	}

	b, err := json.Marshal(searchTerms)
	if err != nil {
		return err
	}

	_, err = fmt.Fprint(cfg.OutputFile, string(b))
	if err != nil {
		return err
	}

	return nil
}
