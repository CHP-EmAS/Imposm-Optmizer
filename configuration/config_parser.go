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
	MappingFilePath      string              `json:"mapping_path"`
	MappingOutPath       string              `json:"mapping_out_path"`
	MappingPrefix        string              `json:"mapping_prefix"`
	KeepColumns          []string            `json:"keep_columns,flow,omitempty"`
	ForceFiltering       bool                `json:"force_filtering"`
	AllowResearch        bool                `json:"allow_research"`
	TableList            map[string][]string `json:"tables,flow,omitempty"`
	GeneralizedTableList map[string][]string `json:"generalized_tables,flow,omitempty"`
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

	foundOldConfig := false
	oldConfig := config{}

	if functions.FileExists(ConfigFile) {
		//check if config file already exists
		configError := ParseConfig(ConfigFile)
		oldConfig = GetConfiguration()

		if configError == nil {
			foundOldConfig = true
		}
	}

	//Path to mapping file input
	if foundOldConfig {
		fmt.Print("Path to mapping file [" + oldConfig.MappingFilePath + "]: ")
	} else {
		fmt.Print("Path to mapping file: ")
	}

	var pathToMapping string
	for {

		fmt.Scanln(&pathToMapping)

		if foundOldConfig && pathToMapping == "" {
			pathToMapping = oldConfig.MappingFilePath
		}

		if !functions.FileExists(pathToMapping) {
			fmt.Print("File could not be found, please enter a correct path to the file: ")
		} else {
			break
		}
	}

	//new mapping file output directory input
	if foundOldConfig {
		fmt.Print("Target folder of the new mapping file [" + oldConfig.MappingOutPath + "]: ")
	} else {
		fmt.Print("Target folder of the new mapping file: ")
	}

	var pathOutMapping string
	for {
		fmt.Scanln(&pathOutMapping)

		if foundOldConfig && pathOutMapping == "" {
			pathOutMapping = oldConfig.MappingOutPath
		}

		if !functions.DirExists(pathOutMapping) {
			fmt.Print("Directory could not be found, please enter a correct path: ")
		} else {
			break
		}
	}

	//new mapping file prefix imput
	if foundOldConfig {
		fmt.Print("What prefix should the new mapping file have? If not specified and destination folder is the same, the file will be overwritten! [" + oldConfig.MappingPrefix + "]: ")
	} else {
		fmt.Print("What prefix should the new mapping file have? If not specified and destination folder is the same, the file will be overwritten!: ")
	}
	var prefix string
	fmt.Scanln(&prefix)
	if foundOldConfig && prefix == "" {
		prefix = oldConfig.MappingPrefix
	}

	//allow research via api input yes or no
	fmt.Print("If certain information about column types or keywords is missing, an API(http://tagfinder.herokuapp.com/apidoc) is used to research it. Should this be allowed? (Y/N): ")
	var ans string
	fmt.Scanln(&ans)

	if foundOldConfig && ans == "" {
		if oldConfig.AllowResearch {
			ans = "y"
		}
	}

	allowResearch := false
	if strings.Compare("y", strings.ToLower(ans)) == 0 || strings.Compare("yes", strings.ToLower(ans)) == 0 {
		allowResearch = true
	}

	//standart required columns -- no input, must be changed in json file
	forceFiltering := false

	if foundOldConfig {
		forceFiltering = oldConfig.ForceFiltering
	}

	//standart required columns -- no input, must be changed in json file
	requiredColumnTypes := []string{"geometry", "validated_geometry", "id", "member_id"}

	if foundOldConfig {
		requiredColumnTypes = oldConfig.KeepColumns
	}

	//input sld's for normal tables
	mappingParser := mapping.New(pathToMapping, forceFiltering, allowResearch, requiredColumnTypes)
	mappingTables := mappingParser.GetTableNames()

	tableMap := make(map[string][]string)

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
				tableMap[tableName] = append(tableMap[tableName], fileName)
			}
		}

		if len(tableMap[tableName]) <= 0 {
			fmt.Print(`No files were specified for the table "` + tableName + `", should this table be ignored in the following remapping? (Y/N): `)
			var ans string
			fmt.Scanln(&ans)

			if strings.Compare("y", strings.ToLower(ans)) == 0 || strings.Compare("yes", strings.ToLower(ans)) == 0 {
				tableMap[tableName] = append(tableMap[tableName], "ignore")
			}
		}
	}

	//input sld's for generalized tables
	mappingGeneralizedTables := mappingParser.GetGeneralizedTableNames()

	generalizedTableMap := make(map[string][]string)

	for _, tableName := range mappingGeneralizedTables {

		fmt.Println(`Path to the sld file/s that uses the generalized "` + tableName + `" table. Type "END" to exit`)

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
				generalizedTableMap[tableName] = append(generalizedTableMap[tableName], fileName)
			}
		}

		if len(generalizedTableMap[tableName]) <= 0 {
			fmt.Print(`No files were specified for the generalized table "` + tableName + `", should this table be ignored in the following remapping? (Y/N): `)
			var ans string
			fmt.Scanln(&ans)

			if strings.Compare("y", strings.ToLower(ans)) == 0 || strings.Compare("yes", strings.ToLower(ans)) == 0 {
				generalizedTableMap[tableName] = append(generalizedTableMap[tableName], "ignore")
			}
		}
	}

	newConf := config{pathToMapping, pathOutMapping, prefix, requiredColumnTypes, forceFiltering, allowResearch, tableMap, generalizedTableMap}

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
