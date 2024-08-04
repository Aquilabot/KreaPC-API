package models

type Vendor struct {
	Name    string
	Image   string
	InStock bool
	Price   Price
	URL     string
}

type SearchPart struct {
	Name   string
	Image  string
	URL    string
	Vendor Vendor
}

type RatingStats struct {
	Stars   uint
	Count   uint
	Average float64
}

type PartSpec struct {
	Name   string
	Values []string
}

type Part struct {
	Type    string
	Name    string
	Images  []string
	URL     string
	Vendors []Vendor
	Specs   []PartSpec
	Rating  RatingStats
}

type ListPart struct {
	Type   string
	Name   string
	Image  string
	URL    string
	Vendor Vendor
}

type CompatibilityInfo struct {
	Message string
	Level   string
}

type PartList struct {
	URL           string
	Parts         []ListPart
	Price         Price
	Wattage       string
	Compatibility []CompatibilityInfo
}
