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
	allowResearch       bool
	requiredColumnTypes []string
}

//New (filePath) createts a new parser object, file path to the mapping file is requiered
func New(filePath string, allowResearch bool, requiredColumnTypes []string) mappingParser {
	m := mappingParser{filePath, false, Mapping{}, "", allowResearch, requiredColumnTypes}
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

func (m *mappingParser) GetMappingColumnNames(tableName string) (string, string) {
	if m.successfullPasing == false {
		m.GetMappingContent()
	}

	var mappingValueColumnName string
	var mappingKeyColumnName string

	for _, column := range m.mappingRoot.Tabels[tableName].Columns {
		if strings.Compare("mapping_value", column.Type) == 0 {
			mappingValueColumnName = column.Name
		} else if strings.Compare("mapping_key", column.Type) == 0 {
			mappingKeyColumnName = column.Name
		}
	}

	return mappingValueColumnName, mappingKeyColumnName
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

		requiredColumnList := combinedRequirements.RequiredColumnList
		requiredMappingValues := combinedRequirements.RequiredMappingValues

		buildColumnList(table, newTable, requiredColumnList, requiredMappingValues, m.allowResearch, m.requiredColumnTypes)

		if len(requiredMappingValues) > 0 && !useAllMappingTypes {
			buildMappingValueList(table, newTable, requiredMappingValues, m.allowResearch)
		} else {
			newTable.Mapping = table.Mapping
			newTable.Mappings = table.Mappings
		}

		newMappingRoot.Tabels[tableName] = *newTable
	}

	return m.buildMappingFile(*newMappingRoot)
}

func appendRequirements(source *sld.TableRequirements, new sld.ParsedSLD) {
	//add all found required table collumns
	for _, value := range new.Requirements.RequiredColumnList {

		found := false
		foundAt := 0
		for i, rColumn := range source.RequiredColumnList {
			if value.PropertyName == rColumn.PropertyName {
				found = true
				foundAt = i
				break
			}
		}

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

		found := false
		for _, rType := range source.RequiredMappingValues {
			if value.Name == rType.Name {
				found = true
				break
			}
		}

		if !found {
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

func (m *mappingParser) RemoveTableFromRoot(tableName string) {
	delete(m.mappingRoot.Tabels, tableName)
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

func buildColumnList(rootTable Table, newTable *Table, requiredColumnList []sld.RequiredColumn, requiredMappingValues []sld.RequiredMappingValue, allowResearch bool, requiredColumnTypes []string) {
	if len(requiredColumnList) > 0 {

		newTable.Columns = make([]TableColumn, 0)

		//extract required table columns and add them to the new table structure
		usedRequiredColumnList := make([]string, 0)

		for _, column := range rootTable.Columns {

			found := false
			for _, rColumn := range requiredColumnList {
				if rColumn.PropertyName == column.Name || functions.StringInSlice(column.Type, requiredColumnTypes) {
					found = true
					break
				}
			}

			if found {

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

func buildMappingValueList(rootTable Table, newTable *Table, requiredMappingValues []sld.RequiredMappingValue, allowResearch bool) {

	if len(rootTable.Mapping) > 0 {

		newTable.Mapping = make(map[string][]string)

		usedRequiredMappingTypes := make([]string, 0)
		for class, keyList := range rootTable.Mapping {
			for _, key := range keyList {

				found := false
				for _, rValue := range requiredMappingValues {
					if rValue.Name == key {
						found = true
						break
					}
				}

				if found {
					newTable.Mapping[class] = append(newTable.Mapping[class], key)

					usedRequiredMappingTypes = append(usedRequiredMappingTypes, key)
				} else {
					fmt.Println(`- Mapping value excluded in mapping class "` + class + `:` + key + `"`)
				}
			}
		}

		for _, rType := range requiredMappingValues {
			if !functions.StringInSlice(rType.Name, usedRequiredMappingTypes) {

				fmt.Println(`- WARNING: Mapping Value "` + rType.Name + `" is required in SLD, but is not defined in mapping!`)

				if allowResearch {
					fmt.Println(`-  Searching for Tag "` + rType.Name + `"...`)
					findKeys := tagfinder.FindTagKey(rType.Name)

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
						fmt.Println(`- Mapping Value "` + rType.Name + `" added with key/class value "` + newKey + `"`)
						newTable.Mapping[newKey] = append(newTable.Mapping[newKey], rType.Name)
					}
				}
			}
		}

	} else if len(rootTable.Mappings) > 0 {
		for mainClass, mappingList := range rootTable.Mappings {

			newTable.Mappings = make(map[string]TableMapping)

			usedRequiredMappingTypes := make([]string, 0)
			for class, keyList := range mappingList.Mapping {

				newMapping := new(TableMapping)
				newMapping.Mapping = make(map[string][]string)

				for _, key := range keyList {

					found := false
					for _, rValue := range requiredMappingValues {
						if rValue.Name == key {
							found = true
							break
						}
					}

					if found {
						newMapping.Mapping[class] = append(newMapping.Mapping[class], key)

						usedRequiredMappingTypes = append(usedRequiredMappingTypes, key)
					} else {
						fmt.Println(`- Mapping value "` + key + `" excluded in mapping class "` + mainClass + `"`)
					}
				}

				newTable.Mappings[mainClass] = *newMapping
			}

			for _, rType := range requiredMappingValues {
				if !functions.StringInSlice(rType.Name, usedRequiredMappingTypes) {

					fmt.Println(`- WARNING: Mapping Value "` + rType.Name + `" is required in SLD, but is not defined in mapping!`)

					if allowResearch {
						fmt.Println(`-  Searching for Tag "` + rType.Name + `"...`)
						findKeys := tagfinder.FindTagKey(rType.Name)

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
							fmt.Println(`- Mapping Value "` + rType.Name + `" added with key/class value "` + newKey + `"`)
							newTable.Mappings[mainClass].Mapping[newKey] = append(newTable.Mappings[mainClass].Mapping[newKey], rType.Name)
						}
					}
				}
			}
		}
	}
}
