package mapping

//file parsing structures

//Tags wip
type Tags struct {
	LoadAll bool     `yaml:"load_all,omitempty" json:"load_all,omitempty"`
	Exclude []string `yaml:"exclude,omitempty" json:"exclude,omitempty"`
	Include []string `yaml:"include,omitempty" json:"include,omitempty"`
}

//GeneralizedTable wip
type GeneralizedTable struct {
	Source    string  `yaml:"source" json:"source"`
	SQLFilter string  `yaml:"sql_filter,omitempty" json:"sql_filter,omitempty"`
	Tolerance float64 `yaml:"tolerance" json:"tolerance"`
}

//Areas wip
type Areas struct {
	AreaTags   []string `yaml:"area_tags" json:"area_tags"`
	LinearTags []string `yaml:"linear_tags" json:"linear_tags"`
}

//TableFilter wip
type TableFilter struct {
	Require       map[string][]string `yaml:"require" json:"require"`
	Reject        map[string][]string `yaml:"reject" json:"reject"`
	RequireRegexp map[string][]string `yaml:"require_regexp" json:"require_regexp"`
	RejectRegexp  map[string][]string `yaml:"reject_regexp" json:"reject_regexp"`
}

//TableColumn contains all informations about a table column from a mapping file
type TableColumn struct {
	Type       string                 `yaml:"type" json:"type"`
	Name       string                 `yaml:"name" json:"name"`
	Key        string                 `yaml:"key,omitempty" json:"key,omitempty"`
	Arguments  map[string]interface{} `yaml:"args,omitempty" json:"args,omitempty"`
	FromMember bool                   `yaml:"from_member,omitempty" json:"from_member,omitempty"`
}

//TableMapping contains mapping values in a map key=mapping key value=array of mapping values
type TableMapping struct {
	Mapping map[string][]string `yaml:"mapping,flow" json:"mapping,flow"`
}

//Table structure contains all informations about one table exported from the mappingfile
type Table struct {
	Type          string                  `yaml:"type" json:"type"`
	Columns       []TableColumn           `yaml:"columns" json:"columns"`
	Mapping       map[string][]string     `yaml:"mapping,flow,omitempty" json:"mapping,flow,omitempty"`
	Mappings      map[string]TableMapping `yaml:"mappings,flow,omitempty" json:"mappings,flow,omitempty"`
	RelationTypes []string                `yaml:"relation_types,flow,omitempty" json:"relation_types,flow,omitempty"`
	Filter        *TableFilter            `yaml:"filters,omitempty" json:"filters,omitempty"`
}

//Mapping Root of the mapping file
type Mapping struct {
	Tabels            map[string]Table            `yaml:"tables" json:"tables"`
	Areas             *Areas                      `yaml:"areas" json:"areas"`
	GeneralizedTables map[string]GeneralizedTable `yaml:"generalized_tables" json:"generalized_tables"`
	Tags              *Tags                       `yaml:"tags,omitempty" json:"tags,omitempty"`
}
