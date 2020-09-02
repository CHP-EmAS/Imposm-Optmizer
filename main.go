package main

import (
	"Imposm_Optimizer/configuration"
	"Imposm_Optimizer/mapping"
	"Imposm_Optimizer/sld"
	functions "Imposm_Optimizer/std_functions"
	"container/list"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strconv"
)

func main() {

	//argument init will init a new config file
	if len(os.Args) > 1 {
		if os.Args[1] == "init" {
			err := configuration.InitConfigFile()
			if err != nil {
				fmt.Println("Error: " + err.Error())
				return
			}
		}
	}

	//load config, if not exists init config
	configError := configuration.ParseConfig(configuration.ConfigFile)
	config := configuration.GetConfiguration()

	fmt.Println("Loading configurations (" + configuration.ConfigFile + ")...")

	//check if configurations are valid
	if configError != nil {
		fmt.Println("Error: " + configError.Error())
		return
	}

	if config.MappingFilePath == "" {
		fmt.Println("Error: mapping_path variable is missing in " + configuration.ConfigFile)
		return
	} else if !functions.FileExists(config.MappingFilePath) {
		fmt.Println(errors.New(`Error: mapping file "` + config.MappingFilePath + `" not found!`))
		return
	} else {
		fmt.Println("- mapping file path         :", config.MappingFilePath)
	}

	if config.MappingOutPath == "" {
		fmt.Println("Error: mapping_out_path variable is missing in " + configuration.ConfigFile)
		return
	} else if !functions.DirExists(config.MappingOutPath) {
		fmt.Println(errors.New(`Error: taget directory "` + config.MappingOutPath + `" not found!`))
		return
	} else {
		fmt.Println("- output directory path     :", config.MappingOutPath)
	}

	if config.TableList == nil {
		fmt.Println(errors.New(`WARNING: there are no file tables in the configuration file`))
		config.TableList = make(map[string][]string)
	}

	fmt.Println("- output file prefix        :", config.MappingPrefix)
	fmt.Println("- filtering is forced       :", config.ForceFiltering)
	fmt.Println("- API is used for searching :", config.AllowResearch)
	fmt.Println("- columns which are kept    :", config.KeepColumns)
	fmt.Println("- tolerance scaling         :", config.ToleranceScaling, "\b%")
	fmt.Println("")

	//init mapping parser
	mappingParser := mapping.New(config.MappingFilePath, config.AllowResearch, config.ForceFiltering, config.ToleranceScaling, config.KeepColumns)

	//get all tables
	mappingTables := mappingParser.GetTableNames()
	mappingGenTables := mappingParser.GetGeneralizedTableNames()

	//init tables
	tableFilesMap := make(map[string](*list.List))
	for i := range mappingTables {
		tableFilesMap[mappingTables[i]] = list.New()
	}

	//init gen tables
	genTableFilesMap := make(map[string](*list.List))
	for i := range mappingGenTables {
		genTableFilesMap[mappingGenTables[i]] = list.New()
	}

	//load all and check all SLD's
	fmt.Println("\n**************** Listing SLD Files *****************")

	for tableName, fileList := range tableFilesMap {

		if config.TableList[tableName] != nil {

			if functions.StringInSlice("ignore", config.TableList[tableName]) {
				delete(tableFilesMap, tableName)
				mappingParser.RemoveTableFromRoot(tableName)
				continue
			}

			fmt.Println(`SLD file/s files for table "` + tableName + `" (read from ` + configuration.ConfigFile + `):`)

			for _, value := range config.TableList[tableName] {
				if functions.FileExists(value) {
					fileList.PushBack(value)
					fmt.Println("- " + value + " found")
				} else {
					fmt.Println("- " + value + " not found!")
				}
			}
		} else {
			fmt.Println(`No SLD files found for table "` + tableName + `". Table will not be changed.`)
		}
	}

	for genTableName, fileList := range genTableFilesMap {

		if config.GeneralizedTableList[genTableName] != nil {

			if functions.StringInSlice("ignore", config.GeneralizedTableList[genTableName]) {
				delete(genTableFilesMap, genTableName)
				mappingParser.RemoveGeneralizedTableFromRoot(genTableName)
				continue
			}

			fmt.Println(`SLD file/s files for generalized table "` + genTableName + `" (read from ` + configuration.ConfigFile + `):`)

			for _, value := range config.GeneralizedTableList[genTableName] {
				if functions.FileExists(value) {
					fileList.PushBack(value)
					fmt.Println("- " + value + " found")
				} else {
					fmt.Println("- " + value + " not found!")
				}
			}
		} else {
			fmt.Println(`No SLD files found for generalized table "` + genTableName + `". Table will not be changed.`)
		}
	}

	//Parse all SLD's and get all needed informations
	fmt.Println("\n***************** Comparing tables *****************")

	comparedTables := make(map[string][]sld.ParsedSLD)

	for tableName, fileList := range tableFilesMap {

		if fileList.Len() <= 0 {
			continue
		}

		fmt.Println(`-------- Comparing table "` + tableName + `"... --------`)

		mappingColumns := mappingParser.GetMappingColumnName(tableName)
		parsedSLDList, err := parseSLDFileList(fileList, mappingColumns)

		if err != nil {
			fmt.Println("Error: " + err.Error())
			return
		}

		fmt.Print("\n")

		comparedTables[tableName] = parsedSLDList
	}

	for genTableName, fileList := range genTableFilesMap {

		if fileList.Len() <= 0 {
			continue
		}

		fmt.Println(`-------- Comparing generalized table "` + genTableName + `"... --------`)

		mappingColumns := mappingParser.GetMappingColumnName(genTableName)
		parsedSLDList, err := parseSLDFileList(fileList, mappingColumns)

		if err != nil {
			fmt.Println("Error: " + err.Error())
			return
		}

		fmt.Print("\n")

		comparedTables[genTableName] = parsedSLDList
	}

	fmt.Println("************** Rebuilding mapping file *************")

	newFileData := mappingParser.RebuildMappingStructure(comparedTables)

	newMappingFilePath := config.MappingOutPath + "/" + config.MappingPrefix + path.Base(config.MappingFilePath)
	newMappingFilePath = path.Clean(newMappingFilePath)

	fmt.Println(`Save mapping file at "` + newMappingFilePath + `"`)
	err := ioutil.WriteFile(newMappingFilePath, newFileData, 0666)

	if err != nil {
		fmt.Println("Error: " + err.Error())
		return
	}

	return
}

func parseSLDFileList(fileList *list.List, mappingColumns sld.MappingColumnNames) ([]sld.ParsedSLD, error) {

	parsedSLDList := make([]sld.ParsedSLD, 0)

	for filePath := fileList.Front(); filePath != nil; filePath = filePath.Next() {
		sldParser := sld.New(fmt.Sprintf("%v", filePath.Value))

		fmt.Println("\n" + `Extracting required columns and mapping types from "` + sldParser.GetFilePath() + `"...`)

		newParsedSLD, err := sldParser.ExtractRequirements(mappingColumns)

		if err != nil {
			return nil, err
		}

		parsedSLDList = append(parsedSLDList, newParsedSLD)

		fmt.Print("- required columns: ")
		if len(newParsedSLD.Requirements.RequiredColumnList) <= 15 && len(newParsedSLD.Requirements.RequiredColumnList) > 0 {
			fmt.Print("[")
			for _, value := range newParsedSLD.Requirements.RequiredColumnList {
				fmt.Print(value.PropertyName + " ")
			}
			fmt.Println("\b]")

		} else {
			fmt.Print(len(newParsedSLD.Requirements.RequiredColumnList))
			fmt.Println(" requirements found")
		}

		fmt.Print("- required mappings types: ")
		if len(newParsedSLD.Requirements.RequiredMappingValues) <= 15 && len(newParsedSLD.Requirements.RequiredMappingValues) > 0 {
			fmt.Println(newParsedSLD.Requirements.RequiredMappingValues)
		} else {
			fmt.Print(len(newParsedSLD.Requirements.RequiredMappingValues))
			fmt.Println(" requirements found")
		}

		maxScale := "∞"

		if newParsedSLD.Scale.MaxScaleDenominator != -2 {
			maxScale = strconv.Itoa(newParsedSLD.Scale.MaxScaleDenominator)
		}

		fmt.Println("- required minimum/maximum scaling: " + strconv.Itoa(newParsedSLD.Scale.MinScaleDenominator) + "/" + maxScale)
	}

	return parsedSLDList, nil
}
