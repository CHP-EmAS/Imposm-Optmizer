package configuration

import (
	"ConverterX/mapping"
	functions "ConverterX/std_functions"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"
)

//ConfigFile Name of the configuration file
const ConfigFile = "config.json"

var parsedConfig config

type config struct {
	MappingFilePath string              `json:"mapping_path"`
	MappingOutPath  string              `json:"mapping_out_path"`
	MappingPrefix   string              `json:"mapping_prefix"`
	KeepColumns     []string            `json:"keep_columns,flow,omitempty"`
	AllowResearch   bool                `json:"allow_research"`
	TableList       map[string][]string `json:"tables,flow,omitempty"`
}

func saveConfigFile(conf config) error {
	newConfByte, err := json.MarshalIndent(&conf, "", "    ")

	if err != nil {
		return err
	}

	fmt.Println("Saving config...")

	return ioutil.WriteFile(ConfigFile, newConfByte, 0666)
}

//InitConfigFile Intialisation of the configuration file, console inputs required
func InitConfigFile() error {
	fmt.Println("***************** Initialization ******************")
	fmt.Println()

	fmt.Print("Path to mapping file: ")
	var pathToMapping string
	for {

		fmt.Scanln(&pathToMapping)

		if !functions.FileExists(pathToMapping) {
			fmt.Print("File could not be found, please enter a correct path to the file: ")
		} else {
			break
		}
	}

	fmt.Print("Target folder of the new mapping file: ")
	var pathOutMapping string
	for {
		fmt.Scanln(&pathOutMapping)

		if !functions.DirExists(pathOutMapping) {
			fmt.Print("Directory could not be found, please enter a correct path: ")
		} else {
			break
		}
	}

	fmt.Print("What prefix should the new mapping file have? If not specified and destination folder is the same, the file will be overwritten!: ")
	var prefix string
	fmt.Scanln(&prefix)

	fmt.Print("If certain information about column types or keywords is missing, an API(http://tagfinder.herokuapp.com/apidoc) is used to research it. Should this be allowed? (Y/N): ")
	var ans string
	fmt.Scanln(&ans)

	allowResearch := false
	if strings.Compare("y", strings.ToLower(ans)) == 0 || strings.Compare("yes", strings.ToLower(ans)) == 0 {
		allowResearch = true
	}

	requiredColumnTypes := []string{"geometry", "validated_geometry", "id", "member_id"}

	mappingParser := mapping.New(pathToMapping, allowResearch, requiredColumnTypes)
	mappingTables := mappingParser.GetTableNames()

	tableList := make(map[string][]string)

	for _, tableName := range mappingTables {

		fmt.Println(`Path to the sld file/s that uses the "` + tableName + `" table. Type "END" to exit`)

		for {
			fmt.Print("-> ")

			var fileName string
			fmt.Scanln(&fileName)

			if strings.Compare("end", strings.ToLower(fileName)) == 0 ||
				strings.Compare("exit", strings.ToLower(fileName)) == 0 {
				break
			} else if !functions.FileExists(fileName) {
				fmt.Println(`File "` + fileName + `" not found!`)
			} else {
				tableList[tableName] = append(tableList[tableName], fileName)
			}
		}

		if len(tableList[tableName]) <= 0 {
			fmt.Print(`No files were specified for the table "` + tableName + `", should this table be ignored in the following remapping? (Y/N): `)
			var ans string
			fmt.Scanln(&ans)

			if strings.Compare("y", strings.ToLower(ans)) == 0 || strings.Compare("yes", strings.ToLower(ans)) == 0 {
				tableList[tableName] = append(tableList[tableName], "ignore")
			}
		}
	}

	newConf := config{pathToMapping, pathOutMapping, prefix, requiredColumnTypes, allowResearch, tableList}

	err := saveConfigFile(newConf)

	if err != nil {
		return err
	}

	return nil
}

//ParseConfig parses the configuration file and saves it into a struct
func ParseConfig(filePath string) error {

	if !functions.FileExists(filePath) {
		err := InitConfigFile()
		if err != nil {
			return err
		}
	}

	yamlFile, err := ioutil.ReadFile(filePath)

	if err != nil {
		return err
	}

	root := config{}

	err = json.Unmarshal(yamlFile, &root)
	if err != nil {
		return err
	}

	parsedConfig = root
	return nil
}

//GetConfiguration returns the parsed configuration struct
func GetConfiguration() config {
	return parsedConfig
}
