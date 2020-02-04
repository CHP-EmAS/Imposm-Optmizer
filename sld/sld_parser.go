package sld

import (
	functions "ConverterX/std_functions"
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io/ioutil"
	"path/filepath"
)

//Parser class
type Parser struct {
	filePath           string
	successfullPasing  bool
	fileByteArray      []byte
	useAllMappingTypes bool
}

//New sldParser instance
func New(filePath string) Parser {
	s := Parser{filePath, false, []byte{}, true}
	return s
}

func (s *Parser) loadSLDFile() error {

	fileExt := filepath.Ext(s.filePath)

	if fileExt != ".sld" {
		return errors.New(`"` + s.filePath + `" must be a .sld file`)
	}

	xmlFile, err := ioutil.ReadFile(s.filePath)

	if err != nil {
		return err
	}

	s.fileByteArray = xmlFile
	s.successfullPasing = true

	return nil
}

/*ExtractRequirements ( mappingValueColumnName ) will return all required values in form of a ParsedSLD structure.
"mappingValueColumnName" specifies the column name, which has the type mapping_value*/
func (s *Parser) ExtractRequirements(mappingValueColumnName string) (ParsedSLD, error) {

	//Check if sld file is already pared
	if s.successfullPasing == false {
		err := s.loadSLDFile()

		if err != nil {
			return ParsedSLD{}, err
		}
	}

	//Parsing required colums from all UserStyles>FeatureTypeStyles>Rules in Filter and Symbolizer Tags
	columnList := make([]RequiredColumn, 0)
	mappingTypeList := make([]RequiredMappingValue, 0)

	err := s.searchSLDColumnNamesRecursiv(s.fileByteArray, &columnList, &mappingTypeList, mappingValueColumnName)

	if err != nil {
		return ParsedSLD{}, err
	}

	return ParsedSLD{s.filePath, TableRequirements{columnList, mappingTypeList}, s.useAllMappingTypes}, nil
}

func (s *Parser) searchSLDColumnNamesRecursiv(mappingFileData []byte, columnList *[]RequiredColumn, mappingTypeList *[]RequiredMappingValue, mappingValueColumnName string) error {
	//init buffer and decoder for unmarshal recursiv xml
	buf := bytes.NewBuffer(mappingFileData)
	dec := xml.NewDecoder(buf)

	//start node
	var n recursiveNode
	err := dec.Decode(&n)
	if err != nil {
		return err
	}

	//Rule array for all found rule tags
	//For the following calculation of the min/max scale dominators of the filtered mapping values
	ruleList := make([]Rule, 0)

	//walk through nodes
	walk([]recursiveNode{n}, &n, func(n recursiveNode) bool {

		//search for PropertyName Element
		if n.XMLName.Local == "PropertyName" {

			newColumnName := string(n.Content)
			newColumn := RequiredColumn{newColumnName, nil}

			literalList := make([]string, 0)

			//search all Literals that belongs to the PropertyName
			if n.ParentNode != nil {
				//search the perentnode for "Literal"
				for _, adjacentNode := range n.ParentNode.Nodes {
					if adjacentNode.XMLName.Local == "Literal" {

						//add Literal to literalList, is used to calculate the data type
						newLiteralName := string(adjacentNode.Content)

						if !functions.StringInSlice(newLiteralName, literalList) {
							literalList = append(literalList, newLiteralName)
						}

						//add Literal to mappingTypeList if columnname matches the mapping value columnname
						if newColumnName == mappingValueColumnName {

							found := false
							for _, mappingType := range *mappingTypeList {
								if mappingType.Name == newLiteralName {
									found = true
									break
								}
							}

							if !found {
								*mappingTypeList = append(*mappingTypeList, RequiredMappingValue{newLiteralName, TypeScaleDenominator{-1, -1}})
							}
						}
					}
				}
			}

			//check if PropertyName Element is not already in list
			found := false
			for _, rColumn := range *columnList {
				if newColumnName == rColumn.PropertyName {
					found = true

					//if PropertyName Element is already in list, add missing literals
					for _, literal := range literalList {
						if !functions.StringInSlice(literal, rColumn.Literals) {
							rColumn.Literals = append(rColumn.Literals, literal)
						}
					}

					break
				}
			}

			if !found {
				newColumn.Literals = literalList
				*columnList = append(*columnList, newColumn)
			}

			//search for VendorOption "name" and "sortby" and add attribut to columnList
		} else if n.XMLName.Local == "VendorOption" {
			for _, attr := range n.Attrs {
				if attr.Name.Local == "name" && attr.Value == "sortBy" {
					newColumnName := string(n.Content)

					//check if PropertyName Element is not already in list
					found := false
					for _, rColumn := range *columnList {
						if newColumnName == rColumn.PropertyName {
							found = true
							break
						}
					}

					if !found {
						*columnList = append(*columnList, RequiredColumn{newColumnName, nil})
					}

				}
			}
			//search after rule tag to check if the rule filter uses the mapping value collumn
		} else if n.XMLName.Local == "Rule" {
			newRule := Rule{}
			err := xml.Unmarshal(n.Content, &newRule)

			if err != nil {
				fmt.Println("Parsing Error: " + err.Error())
			} else {
				ruleList = append(ruleList, newRule)
				fmt.Println(newRule.Name + ": " + string(newRule.Filter.XMLContent))
			}
		}

		//Continue the walk through function
		return true
	})

	foundRule := (len(ruleList) > 0)
	ruleFiltersMappingType := true

	for _, rule := range ruleList {
		mappingTypeFound, err := checkIfRuleFiltersMappingTypes(&rule, mappingValueColumnName)

		if err != nil {
			fmt.Println("Parsing Error: " + err.Error())
			ruleFiltersMappingType = false
		}

		if !mappingTypeFound {
			ruleFiltersMappingType = false
		}
	}

	if foundRule && ruleFiltersMappingType {
		s.useAllMappingTypes = false
	} else {
		fmt.Println("Not all filter tags filter a mapping type, therefore all existing mapping types are used")
	}

	return nil
}

func checkIfRuleFiltersMappingTypes(rule *Rule, mappingValueColumnName string) (bool, error) {

	if rule.Filter.XMLContent == nil {
		return false, fmt.Errorf("Rule XML content is empty")
	}

	//init buffer and decoder for unmarshal recursiv xml
	buf := bytes.NewBuffer(rule.Filter.XMLContent)
	dec := xml.NewDecoder(buf)

	//start node
	var n recursiveNode
	err := dec.Decode(&n)
	if err != nil {
		return false, err
	}

	foundMappingFilter := false

	walk([]recursiveNode{n}, &n, func(n recursiveNode) bool {
		if n.XMLName.Local == "PropertyName" {
			if string(n.Content) == mappingValueColumnName {

				//search all Literals that belongs to the PropertyName
				if n.ParentNode != nil {

					//search the parentnode for tag "Literal"
					for _, adjacentNode := range n.ParentNode.Nodes {
						if adjacentNode.XMLName.Local == "Literal" {

							//fmt.Println(rule.Name + ": " + string(adjacentNode.Content))

						}
					}
				}

				foundMappingFilter = true
			}
		}

		return true
	})

	return foundMappingFilter, nil
}

//Recursive search
type recursiveNode struct {
	XMLName    xml.Name
	Attrs      []xml.Attr      `xml:"-"`
	Content    []byte          `xml:",innerxml"`
	Nodes      []recursiveNode `xml:",any"`
	ParentNode *recursiveNode  `xml:"-"`
}

func walk(nodes []recursiveNode, parentNode *recursiveNode, f func(recursiveNode) bool) {
	for _, n := range nodes {

		n.ParentNode = parentNode

		if f(n) {
			walk(n.Nodes, &n, f)
		}
	}
}

//override UnmarshalXML function
func (n *recursiveNode) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {

	//get XML Attributs
	n.Attrs = start.Attr

	type node recursiveNode

	return d.DecodeElement((*node)(n), &start)
}

//getter/setter

//IsParsed shows if the sld instance was successfully parsed
func (s *Parser) IsParsed() bool {
	return s.successfullPasing
}

//GetFilePath returns the path to the SLD file
func (s *Parser) GetFilePath() string {
	return s.filePath
}

/*UseAllMappingTypes indicates whether it is obvious which mapping values should be used.
true = mapping values can be filtered,
false = it is not obvious which mapping values can be used. There is no filtering*/
func (s *Parser) UseAllMappingTypes() bool {
	return s.useAllMappingTypes
}
