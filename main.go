package main

import (
	"ConverterX/configuration"
	"ConverterX/mapping"
	"ConverterX/sld"
	functions "ConverterX/std_functions"
	"container/list"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strconv"
)

func main() {

	if len(os.Args) > 1 {
		if os.Args[1] == "init" {
			err := configuration.InitConfigFile()
			if err != nil {
				fmt.Println("Error: " + err.Error())
				return
			}
		}
	}

	configError := configuration.ParseConfig(configuration.ConfigFile)
	config := configuration.GetConfiguration()

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
	}

	if config.MappingOutPath == "" {
		fmt.Println("Error: mapping_out_path variable is missing in " + configuration.ConfigFile)
		return
	} else if !functions.DirExists(config.MappingOutPath) {
		fmt.Println(errors.New(`Error: taget directory "` + config.MappingOutPath + `" not found!`))
		return
	}

	if config.TableList == nil {
		config.TableList = make(map[string][]string)
	}

	mappingParser := mapping.New(config.MappingFilePath, config.AllowResearch, config.KeepColumns)
	mappingTables := mappingParser.GetTableNames()

	fmt.Println(mappingParser.GetGeneralizedTableNames())

	//init tables
	tableFilesMap := make(map[string](*list.List))
	for i := range mappingTables {
		tableFilesMap[mappingTables[i]] = list.New()
	}

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

	fmt.Println("\n***************** Comparing tables *****************")

	comparedTables := make(map[string][]sld.ParsedSLD)

	for tableName, fileList := range tableFilesMap {

		if fileList.Len() <= 0 {
			continue
		}

		fmt.Println(`-------- Comparing table "` + tableName + `"... --------`)

		mappingValueColumn, _ := mappingParser.GetMappingColumnNames(tableName)

		parsedSLDList := make([]sld.ParsedSLD, 0)

		for filePath := fileList.Front(); filePath != nil; filePath = filePath.Next() {
			sldParser := sld.New(fmt.Sprintf("%v", filePath.Value))

			fmt.Println("\n" + `Extracting required columns and mapping types from "` + sldParser.GetFilePath() + `"...`)

			newParsedSLD, err := sldParser.ExtractRequirements(mappingValueColumn)

			if err != nil {
				fmt.Println(err)
				return
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

		fmt.Println("")

		comparedTables[tableName] = parsedSLDList
	}

	fmt.Println("************** Rebuilding mapping file *************")

	newFileData := mappingParser.RebuildMappingStructure(comparedTables)

	newMappingFilePath := config.MappingOutPath + "/" + config.MappingPrefix + path.Base(config.MappingFilePath)
	newMappingFilePath = path.Clean(newMappingFilePath)

	fmt.Println("\n" + `Save mapping file at "` + newMappingFilePath + `"`)
	err := ioutil.WriteFile(newMappingFilePath, newFileData, 0666)

	if err != nil {
		fmt.Println("Error: " + err.Error())
	}
}
