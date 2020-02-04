package sld

//########### SLD structures ###########//

//Filter contains raw XML data from a filter tag of a SLD
type Filter struct {
	XMLContent []byte `xml:",innerxml"`
}

//Symbolizer contains raw XML data from a symbolic tag of a SLD
type Symbolizer struct {
	XMLContent []byte `xml:",innerxml"`
}

//Rule describes the structure of a rule in an SLD
type Rule struct {
	Name              string       `xml:"Name,omitempty"`
	Title             string       `xml:"Title,omitempty"`
	Abstract          string       `xml:"Abstract,omitempty"`
	MinScale          int          `xml:"MinScaleDenominator,omitempty"`
	MaxScale          int          `xml:"MaxScaleDenominator,omitempty"`
	Filter            Filter       `xml:"Filter,omitempty"`
	PointSymbolizer   []Symbolizer `xml:"PointSymbolizer,omitempty"`
	LineSymbolizer    []Symbolizer `xml:"LineSymbolizer,omitempty"`
	PolygonSymbolizer []Symbolizer `xml:"PolygonSymbolizer,omitempty"`
	TextSymbolizer    []Symbolizer `xml:"TextSymbolizer,omitempty"`
	RasterSymbolizer  []Symbolizer `xml:"RasterSymbolizer,omitempty"`
}

//VendorOption contains the name and value of a VendorOption from an SLD
type VendorOption struct {
	OptionName string `xml:"name,attr"`
	Value      string `xml:",chardata"`
}

//########### Parser structures ###########//

type TypeScaleDenominator struct {
	MinScaleDenominator int
	MaxScaleDenominator int
}

//RequiredMappingValue contains the name and key values of a mapping class
type RequiredMappingValue struct {
	Name  string
	Scale TypeScaleDenominator
}

//RequiredColumn contains the key name and key values of a mapping class
type RequiredColumn struct {
	PropertyName string
	Literals     []string
}

//TableRequirements combine all required table columns and mapping values
type TableRequirements struct {
	RequiredColumnList    []RequiredColumn
	RequiredMappingValues []RequiredMappingValue
}

//ParsedSLD contains necessary information about the parsed SLD file
//FileName = the path to the parsed SLD file
//Requirements = List of the required table columns/mapping values
//UseAllMappingTypes = If all values are to be used
type ParsedSLD struct {
	FileName           string
	Requirements       TableRequirements
	UseAllMappingTypes bool
}
