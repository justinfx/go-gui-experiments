package memory

import (
	"slices"
	"strings"
)

// Word lists are kept sorted so that comparing (adjective, material, noun)
// index tuples gives the same order as comparing the joined name strings.
var (
	adjectives = sorted([]string{"Antique", "Bold", "Brushed", "Classic", "Compact", "Deluxe", "Durable", "Elegant",
		"Essential", "Foldable", "Glossy", "Handmade", "Heavy", "Light", "Matte", "Modern", "Polished", "Portable",
		"Premium", "Rugged", "Rustic", "Sleek", "Smart", "Vintage"})
	materials = sorted([]string{"Aluminum", "Bamboo", "Brass", "Canvas", "Carbon", "Ceramic", "Copper", "Cotton",
		"Glass", "Granite", "Leather", "Linen", "Maple", "Marble", "Nylon", "Oak", "Rubber", "Silicone", "Steel", "Walnut"})
	nouns = sorted([]string{"Backpack", "Basket", "Blender", "Bottle", "Bowl", "Bucket", "Cable", "Chair", "Charger",
		"Clock", "Cutter", "Desk", "Drill", "Fan", "Hammer", "Headphones", "Helmet", "Kettle", "Ladder", "Lamp", "Lantern",
		"Mat", "Mirror", "Mug", "Notebook", "Pan", "Pen", "Planter", "Rack", "Shelf", "Speaker", "Stand", "Stool", "Tent",
		"Toolbox", "Tray", "Umbrella", "Vase", "Wallet", "Wrench"})
	sentences = []string{"Ships in a gift box.", "Customer favorite this season.", "Limited stock from the spring batch.",
		"Assembly required.", "Includes a two-year warranty.", "Hand-checked before shipping.",
		"Popular with outdoor enthusiasts.", "Replacement parts available on request.", "Fragile, handle with care.",
		"Restock expected soon.", "Made from recycled materials.", "Best seller in its category.",
		"Batteries not included.", "Dishwasher safe.", "Water resistant.", "Color may vary slightly."}

	adjectivesLower = lower(adjectives)
	materialsLower  = lower(materials)
	nounsLower      = lower(nouns)
)

func sorted(s []string) []string { slices.Sort(s); return s }

func lower(s []string) [][]byte {
	out := make([][]byte, len(s))
	for i, w := range s {
		out[i] = []byte(strings.ToLower(w))
	}
	return out
}
