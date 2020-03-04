package mapping

import (
	tagfinder "ConverterX/osm_tagfinder_api"
	"ConverterX/sld"
	functions "ConverterX/std_functions"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strconv"
	"strings"

	"gopkg.in/yaml.v2"
)

type mappingParser struct {
	filePath            string
	successfullPasing   bool
	mappingRoot         Mapping
	sourceFileType      string
	forceFiltering      bool
	allowResearch       bool
	toleranceScaling    float32
	requiredColumnTypes []string
}

//New (filePath) createts a new parser object, file path to the mapping file is requiered
func New(filePath string, allowResearch bool, forceFiltering bool, toleranceScaling float32, requiredColumnTypes []string) mappingParser {
	m := mappingParser{filePath, false, Mapping{}, "", forceFiltering, allowResearch, toleranceScaling, requiredColumnTypes}
	return m
}

//File parsing functions
func (m *mappingParser) parseMappingFileYAML() {
	yamlFile, err := ioutil.ReadFile(m.filePath)

	if err != nil {
		panic(err)
	}

	root := Mapping{}

	err = yaml.Unmarshal(yamlFile, &root)
	if err != nil {
		panic(err)
	}

	m.mappingRoot = root
	m.successfullPasing = true
}

func (m *mappingParser) parseMappingFileJSON() {
	jsonFile, err := ioutil.ReadFile(m.filePath)

	if err != nil {
		panic(err)
	}

	if json.Valid(jsonFile) == false {
		panic("JSON file is not valid")
	}

	root := Mapping{}

	err = json.Unmarshal(jsonFile, &root)
	if err != nil {
		panic(err)
	}

	m.mappingRoot = root
	m.successfullPasing = true
}

func (m *mappingParser) GetMappingContent() Mapping {
	if m.successfullPasing == true {
		return m.mappingRoot
	}

	fileExt := filepath.Ext(m.filePath)

	fmt.Println(`Parsing "` + m.filePath + `"...`)

	if fileExt == ".json" {
		m.parseMappingFileJSON()
	} else if fileExt == ".yaml" {
		m.parseMappingFileYAML()
	} else {
		panic("Mapping file must be a yaml or json file")
	}

	m.sourceFileType = fileExt

	return m.mappingRoot
}

//GetMappingColumnName returns the names of the columns which have the column type "mapping_value" and "mapping_key".
//tableName: Name of the table from which the values are needed.
//If the given table is a generalized table, the values of its source table are returned!
func (m *mappingParser) GetMappingColumnName(tableName string) sld.MappingColumnNames {
	if m.successfullPasing == false {
		m.GetMappingContent()
	}

	if m.mappingRoot.GeneralizedTables[tableName] != (GeneralizedTable{}) {
		tableName = m.GetGeneralizedRootSourceTable(tableName)
	}

	mappingColumns := sld.MappingColumnNames{}

	for _, column := range m.mappingRoot.Tabels[tableName].Columns {
		if strings.Compare("mapping_value", column.Type) == 0 {
			mappingColumns.MappingValueColumnName = column.Name
		} else if strings.Compare("mapping_key", column.Type) == 0 {
			mappingColumns.MappingKeyColumnName = column.Name
		}
	}

	return mappingColumns
}

func (m *mappingParser) buildMappingFile(newMappingStructure Mapping) []byte {
	var fileContent []byte
	var err error

	switch m.sourceFileType {
	case ".json":
		fileContent, err = json.MarshalIndent(newMappingStructure, "", "    ")
	case ".yaml":
		fileContent, err = yaml.Marshal(newMappingStructure)
	default:
		if err != nil {
			panic(`Cannot build mapping file: Unknown file type "` + m.sourceFileType + `!`)
		}
	}

	if err != nil {
		panic(err)
	}

	return fileContent
}

func (m *mappingParser) RebuildMappingStructure(parsedSLDs map[string][]sld.ParsedSLD) []byte {
	if m.successfullPasing == false {
		panic("Try to rebuild a non existing mapping structure!")
	}

	newMappingRoot := new(Mapping)

	newMappingRoot.Areas = m.mappingRoot.Areas
	newMappingRoot.GeneralizedTables = m.mappingRoot.GeneralizedTables
	newMappingRoot.Tags = m.mappingRoot.Tags
	newMappingRoot.Tabels = make(map[string]Table)

	//build all known tables
	for tableName, table := range m.mappingRoot.Tabels {

		fmt.Println(`Building Table "` + tableName + `"...`)

		newTable := new(Table)

		//copy static table data
		newTable.Type = table.Type
		newTable.RelationTypes = table.RelationTypes
		newTable.Filter = table.Filter

		//merge the parsed sld data to a list
		combinedRequirements := sld.TableRequirements{}
		useAllMappingTypes := false

		for _, comparedTable := range parsedSLDs[tableName] {
			appendRequirements(&combinedRequirements, comparedTable)

			if comparedTable.UseAllMappingTypes {
				useAllMappingTypes = true
			}
		}

		relatedGenTables := m.getRelatedGeneralizedTables(tableName)

		if len(relatedGenTables) > 0 {
			fmt.Println("- Related generalized tables:", relatedGenTables)
		}

		for _, relGenTable := range relatedGenTables {
			for _, comparedTable := range parsedSLDs[relGenTable] {
				appendRequirements(&combinedRequirements, comparedTable)

				if comparedTable.UseAllMappingTypes {
					useAllMappingTypes = true
				}
			}
		}

		requiredColumnList := combinedRequirements.RequiredColumnList
		requiredMappingValues := combinedRequirements.RequiredMappingValues

		buildColumnList(table, newTable, requiredColumnList, m.allowResearch, m.requiredColumnTypes)

		if len(requiredMappingValues) > 0 && (!useAllMappingTypes || m.forceFiltering) {
			buildMappingValueList(table, newTable, requiredMappingValues, m.allowResearch)
		} else {
			fmt.Println("- Not all filter tags filter a mapping type, therefore all existing mapping types are used!")

			newTable.Mapping = table.Mapping
			newTable.Mappings = table.Mappings
		}

		newMappingRoot.Tabels[tableName] = *newTable

		fmt.Println("")
	}

	for genTableName, table := range m.mappingRoot.GeneralizedTables {
		fmt.Println(`Building generalized Table "` + genTableName + `"...`)

		newGenTable := new(GeneralizedTable)

		//copy static table data
		newGenTable.Source = table.Source

		//merge the parsed sld data to a list
		combinedRequirements := sld.TableRequirements{}
		useAllMappingTypes := false

		//get the minimum min scale
		var minScale int = -1

		for _, comparedTable := range parsedSLDs[genTableName] {
			appendRequirements(&combinedRequirements, comparedTable)

			if comparedTable.UseAllMappingTypes {
				useAllMappingTypes = true
			}

			if minScale == -1 || minScale > comparedTable.Scale.MinScaleDenominator {
				minScale = comparedTable.Scale.MinScaleDenominator
			}
		}

		relatedGenTables := m.getRelatedGeneralizedTables(genTableName)

		if len(relatedGenTables) > 0 {
			fmt.Println("- Related generalized tables:", relatedGenTables)
		}

		for _, relGenTable := range relatedGenTables {
			for _, comparedTable := range parsedSLDs[relGenTable] {
				appendRequirements(&combinedRequirements, comparedTable)

				if comparedTable.UseAllMappingTypes {
					useAllMappingTypes = true
				}
			}
		}

		newGenTable.SQLFilter = generateSQLFilter(m.GetMappingColumnName(genTableName), combinedRequirements.RequiredColumnList, combinedRequirements.RequiredMappingValues, table.SQLFilter, (useAllMappingTypes && !m.forceFiltering))

		if newGenTable.SQLFilter != "" {
			fmt.Println("- SQL-Filter: " + newGenTable.SQLFilter)
		}

		newGenTable.Tolerance = float64(minScale) * (float64(m.toleranceScaling) / float64(100.0))
		fmt.Println("- Tolerance:", newGenTable.Tolerance)

		newMappingRoot.GeneralizedTables[genTableName] = *newGenTable

		fmt.Println("")
	}

	return m.buildMappingFile(*newMappingRoot)
}

func appendRequirements(source *sld.TableRequirements, new sld.ParsedSLD) {
	//add all found required table collumns
	for _, value := range new.Requirements.RequiredColumnList {

		found, foundAt := sld.ColumnInColumnlist(value.PropertyName, source.RequiredColumnList)

		if !found {
			source.RequiredColumnList = append(source.RequiredColumnList, value)
		} else {
			for _, literal := range value.Literals {
				if !functions.StringInSlice(literal, source.RequiredColumnList[foundAt].Literals) {
					source.RequiredColumnList[foundAt].Literals = append(source.RequiredColumnList[foundAt].Literals, literal)
				}
			}
		}
	}

	//add all found required types
	for _, value := range new.Requirements.RequiredMappingValues {
		if !functions.StringInSlice(value, source.RequiredMappingValues) {
			source.RequiredMappingValues = append(source.RequiredMappingValues, value)
		}
	}
}

//getter setter
func (m *mappingParser) IsParsed() bool {
	return m.successfullPasing
}

func (m *mappingParser) GetTableNames() []string {
	if m.successfullPasing == false {
		m.GetMappingContent()
	}

	var tables []string = make([]string, len(m.mappingRoot.Tabels))

	var tableIndex = 0

	for key := range m.mappingRoot.Tabels {
		tables[tableIndex] = key
		tableIndex++
	}

	return tables
}

func (m *mappingParser) GetGeneralizedTableNames() []string {
	if m.successfullPasing == false {
		m.GetMappingContent()
	}

	var tables []string = make([]string, len(m.mappingRoot.GeneralizedTables))

	var tableIndex = 0

	for key := range m.mappingRoot.GeneralizedTables {
		tables[tableIndex] = key
		tableIndex++
	}

	return tables
}

func (m *mappingParser) GetGeneralizedRootSourceTable(genTabelName string) string {
	if m.successfullPasing == false {
		m.GetMappingContent()
	}

	genSourceRootTable := genTabelName

	if m.mappingRoot.GeneralizedTables[genSourceRootTable] != (GeneralizedTable{}) {
		genSourceRootTable = m.mappingRoot.GeneralizedTables[genSourceRootTable].Source
		genSourceRootTable = m.GetGeneralizedRootSourceTable(genSourceRootTable)
	}

	return genSourceRootTable
}

func (m *mappingParser) RemoveTableFromRoot(tableName string) {
	delete(m.mappingRoot.Tabels, tableName)
}

func (m *mappingParser) RemoveGeneralizedTableFromRoot(genTableName string) {
	delete(m.mappingRoot.GeneralizedTables, genTableName)
}

func guessColumnType(literals []string) string {

	columnType := ""

	for _, literal := range literals {

		switch columnType {
		case "":
			if _, err := strconv.ParseInt(literal, 10, 64); err == nil {
				if functions.StringInSlice(literal, []string{"1", "0"}) {
					columnType = "boolint"
				} else if literal == "-1" {
					columnType = "direction"
				} else {
					columnType = "integer"
				}
			} else {
				if functions.StringInSlice(literal, []string{"true", "yes", "false", "no"}) {
					columnType = "bool"
				} else {
					return "string"
				}
			}
		case "boolint":
			if _, err := strconv.ParseInt(literal, 10, 64); err == nil {
				if literal == "-1" {
					columnType = "direction"
				} else if !functions.StringInSlice(literal, []string{"1", "0"}) {
					columnType = "integer"
				}
			} else {
				if functions.StringInSlice(literal, []string{"true", "yes", "false", "no"}) {
					columnType = "bool"
				} else {
					return "string"
				}
			}
		case "bool":
			if _, err := strconv.ParseInt(literal, 10, 64); err == nil {
				if !functions.StringInSlice(literal, []string{"1", "0"}) {
					return "string"
				}
			} else {
				if !functions.StringInSlice(literal, []string{"true", "yes", "false", "no"}) {
					return "string"
				}
			}
		case "direction":
			if _, err := strconv.ParseInt(literal, 10, 64); err == nil {
				if !functions.StringInSlice(literal, []string{"1", "0", "-1"}) {
					columnType = "integer"
				}
			} else {
				return "string"
			}
		case "integer":
			if _, err := strconv.ParseInt(literal, 10, 64); err != nil {
				return "string"
			}
		}
	}

	if columnType != "" {
		return columnType
	}

	return "string"
}

func buildColumnList(rootTable Table, newTable *Table, requiredColumnList []sld.RequiredColumn, allowResearch bool, requiredColumnTypes []string) {
	if len(requiredColumnList) > 0 {

		newTable.Columns = make([]TableColumn, 0)

		//extract required table columns and add them to the new table structure
		usedRequiredColumnList := make([]string, 0)

		for _, column := range rootTable.Columns {

			found, _ := sld.ColumnInColumnlist(column.Name, requiredColumnList)
			required := found || functions.StringInSlice(column.Type, requiredColumnTypes)

			if required {

				newTable.Columns = append(newTable.Columns, column)
				usedRequiredColumnList = append(usedRequiredColumnList, column.Name)

			} else {
				fmt.Println(`- Tabel column excluded "` + column.Name + `"`)
			}

		}

		//search for unused needed table columns and try to add them
		for _, rColumn := range requiredColumnList {
			if !functions.StringInSlice(rColumn.PropertyName, usedRequiredColumnList) {

				fmt.Println(`- WARNING: Table column "` + rColumn.PropertyName + `" is required in SLD, but is not defined in mapping!`)

				if allowResearch {
					fmt.Println(`-  Searching for Key "` + rColumn.PropertyName + `"...`)

					if tagfinder.CheckIfKeyExists(rColumn.PropertyName) {

						columnType := guessColumnType(rColumn.Literals)
						fmt.Println(`-  Key found. Tabel column "` + rColumn.PropertyName + `" added, data type "` + columnType + `" was guessed`)

						newColumn := TableColumn{columnType, rColumn.PropertyName, "", nil, false}
						newTable.Columns = append(newTable.Columns, newColumn)

					} else {
						fmt.Println(`-  Key not found. Tabel column "` + rColumn.PropertyName + `" excluded`)
					}
				}
			}
		}

	} else {
		newTable.Columns = rootTable.Columns
	}
}

func buildMappingValueList(rootTable Table, newTable *Table, requiredMappingValues []string, allowResearch bool) {

	if len(rootTable.Mapping) > 0 {

		newTable.Mapping = make(map[string][]string)

		usedRequiredMappingTypes := make([]string, 0)
		for class, keyList := range rootTable.Mapping {
			for _, key := range keyList {

				if functions.StringInSlice(key, requiredMappingValues) {
					newTable.Mapping[class] = append(newTable.Mapping[class], key)

					usedRequiredMappingTypes = append(usedRequiredMappingTypes, key)
				} else {
					fmt.Println(`- Mapping value excluded in mapping class "` + class + `:` + key + `"`)
				}
			}
		}

		for _, rType := range requiredMappingValues {
			if !functions.StringInSlice(rType, usedRequiredMappingTypes) {

				fmt.Println(`- WARNING: Mapping Value "` + rType + `" is required in SLD, but is not defined in mapping!`)

				if allowResearch {
					fmt.Println(`-  Searching for Tag "` + rType + `"...`)
					findKeys := tagfinder.FindTagKey(rType)

					if len(findKeys) == 1 {
						fmt.Print("-  The following keyword was found: ")
						fmt.Println(findKeys[0])
					} else if len(findKeys) > 1 {
						fmt.Print("-  The following keywords were found: ")
						fmt.Println(findKeys)
					} else {
						fmt.Println("-  No matching keywords were found!")
						continue
					}

					for _, newKey := range findKeys {
						fmt.Println(`- Mapping Value "` + rType + `" added with key/class value "` + newKey + `"`)
						newTable.Mapping[newKey] = append(newTable.Mapping[newKey], rType)
					}
				}
			}
		}

	} else if len(rootTable.Mappings) > 0 {

		newTable.Mappings = make(map[string]TableMapping)
		usedRequiredMappingTypes := make([]string, 0)

		for mainClass, mappingList := range rootTable.Mappings {
			for class, keyList := range mappingList.Mapping {

				newMapping := new(TableMapping)
				newMapping.Mapping = make(map[string][]string)

				for _, key := range keyList {

					if functions.StringInSlice(key, requiredMappingValues) {
						newMapping.Mapping[class] = append(newMapping.Mapping[class], key)
						usedRequiredMappingTypes = append(usedRequiredMappingTypes, key)
					} else {
						fmt.Println(`- Mapping value "` + key + `" excluded in mapping class "` + mainClass + `"`)
					}
				}

				newTable.Mappings[mainClass] = *newMapping
			}
		}

		for _, rType := range requiredMappingValues {
			if !functions.StringInSlice(rType, usedRequiredMappingTypes) {

				fmt.Println(`- WARNING: Mapping Value "` + rType + `" is required in SLD, but is not defined in mapping!`)

				if allowResearch {
					fmt.Println(`-  Searching for Tag "` + rType + `"...`)
					findKeys := tagfinder.FindTagKey(rType)

					if len(findKeys) == 1 {
						fmt.Print("-  The following keyword was found: ")
						fmt.Println(findKeys[0])
					} else if len(findKeys) > 1 {
						fmt.Print("-  The following keywords were found: ")
						fmt.Println(findKeys)
					} else {
						fmt.Println("-  No matching keywords were found!")
						continue
					}

					for _, newKey := range findKeys {
						fmt.Println(`- Mapping Value "` + rType + `" added with key/class value "` + newKey + `"`)

						if newTable.Mappings[newKey].Mapping != nil {
							newTable.Mappings[newKey].Mapping[newKey] = append(newTable.Mappings[newKey].Mapping[newKey], rType)
						} else {
							for mainClass := range rootTable.Mappings {
								newTable.Mappings[mainClass].Mapping[newKey] = append(newTable.Mappings[mainClass].Mapping[newKey], rType)
							}
						}
					}
				}
			}
		}
	}
}

func (m *mappingParser) getRelatedGeneralizedTables(tableName string) []string {
	foundGenTable := make([]string, 0)

	for genTableName, genTable := range m.mappingRoot.GeneralizedTables {
		if genTable.Source == tableName {
			foundGenTable = m.getRelatedGeneralizedTables(genTableName)
			foundGenTable = append(foundGenTable, genTableName)
		}
	}

	return foundGenTable
}

func generateSQLFilter(mappingColumns sld.MappingColumnNames, requiredColumnList []sld.RequiredColumn, requiredMappingValues []string, oldSQLFilter string, useAllMappingTypes bool) string {
	filter := ""

	if !useAllMappingTypes {

		found, _ := sld.ColumnInColumnlist(mappingColumns.MappingValueColumnName, requiredColumnList)

		if found && len(requiredMappingValues) > 0 {

			filter = mappingColumns.MappingValueColumnName + " IN ("

			for i, mappingValue := range requiredMappingValues {
				if i > 0 {
					filter = filter + ", '" + mappingValue + "'"
				} else {
					filter = filter + "'" + mappingValue + "'"
				}
			}

			filter = filter + ")"
		}

		if oldSQLFilter != "" {
			oldSQLFilter = discardMappingValueFilter(oldSQLFilter, mappingColumns)

			if oldSQLFilter != "" {
				filter = filter + " AND " + oldSQLFilter
			}
		}
	}

	if filter == "" {
		filter = oldSQLFilter
	}

	return filter
}

func discardMappingValueFilter(sqlFilter string, mappingColumns sld.MappingColumnNames) string {

	newFilter := ""

	if strings.Contains(sqlFilter, mappingColumns.MappingValueColumnName) || strings.Contains(sqlFilter, mappingColumns.MappingKeyColumnName) {
		split := strings.Fields(sqlFilter)

		mode := false
		bracket := false

		for _, c := range split {
			if c == mappingColumns.MappingValueColumnName || c == mappingColumns.MappingKeyColumnName {
				mode = true
			}

			if mode && !bracket {
				if c == "OR" || c == "AND" {
					mode = false
				} else if strings.Contains(c, "(") {
					bracket = true
				}
			} else if !mode && !bracket {
				newFilter += c + " "
			} else {
				if strings.Contains(c, ")") {
					bracket = false
				}
			}
		}
	} else {
		return sqlFilter
	}

	return newFilter
}
