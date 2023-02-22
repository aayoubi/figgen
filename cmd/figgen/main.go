package main

import (
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/aayoubi/figgen/pkg/qcat"
	"github.com/aayoubi/figgen/pkg/recommend"
	"github.com/aayoubi/figgen/pkg/version"
	"github.com/algolia/algoliasearch-client-go/v3/algolia/search"
	"github.com/spf13/cobra"
)

func newQCatFiggenCmd() *cobra.Command {
	var appId, apiKey, indexName, outputFileName, language string
	var facetCombinations []string
	command := &cobra.Command{
		Use:   "qcat",
		Short: "Generate fig's events searches.json optimized for Query Categorization",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := qcat.NewConfig(appId, apiKey, indexName, outputFileName, language, facetCombinations)
			if err != nil {
				return err
			}
			return qcat.Gen(cfg)
		},
	}
	command.Flags().StringVarP(&appId, "app-id", "a", "", "Algolia Application ID")
	command.Flags().StringVarP(&apiKey, "api-key", "k", "", "Algolia API key")
	command.Flags().StringVarP(&indexName, "index-name", "i", "", "Index name")
	command.Flags().StringVarP(&outputFileName, "output", "o", "", "Output file name")
	command.Flags().StringVarP(&language, "language", "l", "english", "Stop words language")
	command.Flags().StringArrayVar(&facetCombinations, "facets", []string{}, `Comma-separated list of facets combinations to use to generate unique queries.
--facets can be used multiple times to specify multiple combinations:
Example: --facets "color,brand" --facets "brand" --facets "gender,categories"`)
	return command
}

func newRecommendFiggenCmd() *cobra.Command {
	var seed int64
	var appId, apiKey, indexName, facetName, outputFileName, seperator string
	command := &cobra.Command{
		Use:   "recommend",
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
			cfg := &recommend.Config{
				FacetName:    facetName,
				SearchIndex:  searchIndex,
				SearchClient: searchClient,
				Seperator:    seperator,
				OutputFile:   outputFile,
			}
			return recommend.Gen(cfg)
		},
	}

	command.Flags().Int64Var(&seed, "seed", time.Now().UnixNano(), "Specify a seed for the random number generator")
	command.Flags().StringVarP(&appId, "app-id", "a", "", "Algolia Application ID")
	command.Flags().StringVarP(&apiKey, "api-key", "k", "", "Algolia API key")
	command.Flags().StringVarP(&indexName, "index-name", "i", "", "Index name")
	command.Flags().StringVarP(&facetName, "facet-name", "f", "$.hierarchical_categories.lvl2", "Facet name for FBT categorizations in jsonpath format. You must use the Hierarchical Categories facet.")
	command.Flags().StringVarP(&seperator, "seperator", "s", ">", "Hierearchical categories seperator")
	command.Flags().StringVarP(&outputFileName, "output", "o", "", "Output file name")
	return command
}

func main() {
	rootCmd := &cobra.Command{
		Use:   "figgen",
		Short: "Generate fig's input data based on your index data",
		Run: func(cmd *cobra.Command, args []string) {
		},
		Version: version.Version,
	}
	rootCmd.SetVersionTemplate(version.Template)
	rootCmd.AddCommand(newRecommendFiggenCmd())
	rootCmd.AddCommand(newQCatFiggenCmd())
	rootCmd.Execute()
}
