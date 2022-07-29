# Figgen

A config generator for `fig`

# Usage

```
% figgen -h
Generate fig's recommend.json

Usage:
  figgen [flags]

Flags:
  -k, --api-key string      Algolia API key
  -a, --app-id string       Algolia Application ID
  -f, --facet-name string   Facet name for FBT categorizations in jsonpath format. You must use the Hierarchical Categories facet. (default "$.hierarchical_categories.lvl2")
  -h, --help                help for figgen
  -i, --index-name string   Index name
  -o, --output string       Output file name
      --seed int            Specify a seed for the random number generator (default 1659086526896243000)
  -s, --seperator string    Hierearchical categories seperator (default ">")
```

# Sample

```
% figgen --app-id ********* --api-key ******************************** --index-name prod_ECOM --facet-name '$.hierarchical_categories.lvl2' | jq .
{
  "facetName": "$.hierarchical_categories.lvl2",
  "FBT": {
    "Men > Clothing > Blazer": [
      "Women > Bags > Clutches",
      "Women > Shoes > Ankle boots"
    ],
    "Men > Clothing > Jeans": [
      "Women > Clothing > Blazer",
      "Women > Clothing > Shirts"
    ],
    "Men > Clothing > T-shirts": [
      "Men > Clothing > Shirts",
      "Women > Bags > Wallets"
    ],
    "Men > Clothing > Trousers": [
      "Women > Shoes > Sandals",
      "Women > Shoes > Loafers"
    ],
    "Men > Shoes > Lace-up shoes": [
      "Men > Shoes > Loafers"
    ],
    "Men > Shoes > Sneakers": [
      "Women > Clothing > T-shirts",
      "Women > Clothing > Jeans"
    ],
    "Women > Bags > Shopper": [
      "Men > Clothing > Suits",
      "Accessories > Women > Clothing"
    ],
    "Women > Clothing > Dresses": [
      "Accessories > Women > Looks",
      "Women > Shoes > Pumps"
    ],
    "Women > Clothing > Jackets": [
      "Women > Bags > Shoulder bags",
      "Accessories > Women > Sunglasses"
    ],
    "Women > Clothing > Tops": [
      "Men > Clothing > Tops",
      "Accessories > Men > Clothing"
    ],
    "Women > Clothing > Trouser": [
      "Women > Bags > Handbag",
      "Women > Shoes > Ballerinas"
    ],
    "Women > Shoes > Sneakers": [
      "Men > Clothing > Jackets",
      "Women > Clothing > Skirts"
    ]
  },
  "trends": {
    "trendingItems": [
      "A0E20000000283M",
      "M0E20000000EKVR",
      "A0E2000000021XZ",
      "M0E20000000DMU3",
      "M0E20000000DJOS",
      "A0E2000000021UH",
      "A0E200000002BDY",
      "M0E20000000EXC2",
      "A0E2000000022KH",
      "M0E20000000DLIA",
      "A0E2000000027DS",
      "M0E20000000E2YP"
    ],
    "trendingFacetValues": [
      "Women > Clothing > Jeans",
      "Women > Clothing > Jackets",
      "Women > Bags > Shopper",
      "Men > Shoes > Sneakers",
      "Men > Clothing > T-shirts"
    ]
  }
}
```