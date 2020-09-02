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

//########### Parser structures ###########//

//ScaleDenominator contains information of the scale denominator of a specific sld file
type ScaleDenominator struct {
	MinScaleDenominator int
	MaxScaleDenominator int
}

//RequiredColumn contains the key name and key values of a mapping class
type RequiredColumn struct {
	PropertyName string
	Literals     []string
}

//TableRequirements combine all required table columns and mapping values
type TableRequirements struct {
	MappingColumns         MappingColumnNames
	RequiredColumnList     []RequiredColumn
	RequiredMappingValues  []string
	ImplicitFilteredValues []string
}

//ParsedSLD contains necessary information about the parsed SLD file
//FileName = the path to the parsed SLD file
//Requirements = List of the required table columns/mapping values
//UseAllMappingTypes = If all mapping values are to be used, is caused by missing filtering of the mapping column
type ParsedSLD struct {
	FileName           string
	Requirements       TableRequirements
	Scale              ScaleDenominator
	UseAllMappingTypes bool
}

//MappingColumnNames stores the column names, which have the value "mapping_value" or "mapping_key" as type
type MappingColumnNames struct {
	MappingKeyColumnName   string
	MappingValueColumnName string
}
