package sld

import (
	functions "Imposm_Optimizer/std_functions"
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io/ioutil"
	"path/filepath"
)

var nonExplicitFilteringOperators = []string{
	"PropertyIsNotEqualTo",
	"PropertyIsLike",
	"PropertyIsBetween",
	"PropertyIsLessThan",
	"PropertyIsLessThanOrEqualTo",
	"PropertyIsGreaterThan",
	"PropertyIsGreaterThanOrEqualTo"}

var nonExplicitFilteringFunctions = []string{
	"between",
	"greaterEqualThan",
	"greaterThan",
	"isLike",
	"lessThan",
	"lessEqualThan",
	"not",
	"notEqual"}

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
func (s *Parser) ExtractRequirements(mappingColums MappingColumnNames) (ParsedSLD, error) {

	//Check if sld file is already pared
	if s.successfullPasing == false {
		err := s.loadSLDFile()

		if err != nil {
			return ParsedSLD{}, err
		}
	}

	columnList := make([]RequiredColumn, 0)
	mappingTypeList := make([]string, 0)
	implicitFilteredValueList := make([]string, 0)

	scaleDenominator := ScaleDenominator{-1, -1}

	err := s.searchSLDRecursiv(s.fileByteArray, &columnList, &mappingTypeList, &implicitFilteredValueList, &scaleDenominator, mappingColums.MappingValueColumnName)

	if err != nil {
		return ParsedSLD{}, err
	}

	return ParsedSLD{s.filePath, TableRequirements{mappingColums, columnList, mappingTypeList, implicitFilteredValueList}, scaleDenominator, s.useAllMappingTypes}, nil
}

func (s *Parser) searchSLDRecursiv(mappingFileData []byte, columnList *[]RequiredColumn, mappingTypeList *[]string, implicitFilteredValueList *[]string, scaleDenominator *ScaleDenominator, mappingValueColumnName string) error {
	//init buffer and decoder for unmarshal recursiv xml
	sldBuffer := bytes.NewBuffer(mappingFileData)
	decoder := xml.NewDecoder(sldBuffer)

	//start node
	var node recursiveNode
	err := decoder.Decode(&node)
	if err != nil {
		return err
	}

	//Rule array for all found rule tags
	//For the following calculation of the min/max scale dominators of the filtered mapping values
	ruleList := make([]Rule, 0)

	//walk through nodes
	walk([]recursiveNode{node}, &node, func(node recursiveNode) bool {

		//search for PropertyName Element
		if node.XMLName.Local == "PropertyName" {

			newColumnName := string(node.Content)
			newColumn := RequiredColumn{newColumnName, nil}

			literalList := make([]string, 0)

			//search all Literals that belongs to the PropertyName
			if node.ParentNode != nil {

				//search the parentnode for "Literal"
				for _, adjacentNode := range node.ParentNode.Nodes {
					if adjacentNode.XMLName.Local == "Literal" {

						//add Literal to literalList, is used to calculate the data type
						newLiteralName := string(adjacentNode.Content)

						if !functions.StringInSlice(newLiteralName, literalList) {
							literalList = append(literalList, newLiteralName)
						}

						//add Literal to mappingTypeList if columnname matches the mapping value columnname
						if newColumnName == mappingValueColumnName {

							if !functions.StringInSlice(newLiteralName, *mappingTypeList) {
								*mappingTypeList = append(*mappingTypeList, newLiteralName)
							}

						}
					}
				}
			}

			//check if PropertyName Element is not already in list
			found, i := ColumnInColumnlist(newColumnName, *columnList)

			if !found {

				newColumn.Literals = literalList
				*columnList = append(*columnList, newColumn)
			} else {
				//if PropertyName Element is already in list, add missing literals
				for _, literal := range literalList {
					if !functions.StringInSlice(literal, (*columnList)[i].Literals) {
						(*columnList)[i].Literals = append((*columnList)[i].Literals, literal)
					}
				}
			}

			//search for VendorOption "name" and "sortby" and add attribut to columnList
		} else if node.XMLName.Local == "VendorOption" {
			for _, attr := range node.Attrs {
				if attr.Name.Local == "name" && attr.Value == "sortBy" {
					newColumnName := string(node.Content)

					//check if PropertyName Element is not already in list
					found, _ := ColumnInColumnlist(newColumnName, *columnList)

					if !found {
						*columnList = append(*columnList, RequiredColumn{newColumnName, nil})
					}

				}
			}

		} else if node.XMLName.Local == "Rule" { //extract all rule tags

			copyByteStream := node.Content

			//add beginning and end tag to the rule content, for correct decoding the rule
			copyByteStream = append([]byte("<Rule>"), copyByteStream...)
			copyByteStream = append(copyByteStream, "</Rule>"...)

			newRule := Rule{}
			err := xml.Unmarshal(copyByteStream, &newRule)

			if err != nil {
				fmt.Println("Parsing Error: " + err.Error())
			} else {
				ruleList = append(ruleList, newRule)
			}
		}

		//Continue the walk through function
		return true
	})

	//search after rule tag to check if the rule filter uses the mapping value collumn
	//calculate the minimum and maximum scale denominator of all mapping values used
	foundRule := (len(ruleList) > 0)
	ruleFiltersMappingType := true

	for _, rule := range ruleList {

		if rule.MaxScale == 0 {
			scaleDenominator.MaxScaleDenominator = -2
		} else if rule.MaxScale > scaleDenominator.MaxScaleDenominator && scaleDenominator.MaxScaleDenominator != -2 {
			scaleDenominator.MaxScaleDenominator = rule.MaxScale
		}

		if rule.MinScale < scaleDenominator.MinScaleDenominator || scaleDenominator.MinScaleDenominator == -1 {
			scaleDenominator.MinScaleDenominator = rule.MinScale
		}

		mappingTypeFound, err := checkIfRuleFiltersMappingTypes(&rule, mappingValueColumnName, implicitFilteredValueList)

		if err != nil {
			fmt.Println("Parsing Error: " + err.Error())
			ruleFiltersMappingType = false
		}

		if !mappingTypeFound {
			ruleFiltersMappingType = false
		}
	}

	//set sld filter status
	if foundRule && ruleFiltersMappingType {
		s.useAllMappingTypes = false
	}

	return nil
}

func checkIfRuleFiltersMappingTypes(rule *Rule, mappingValueColumnName string, implicitFilteredValueList *[]string) (bool, error) {

	if rule.Filter.XMLContent == nil {

		if rule.TextSymbolizer == nil {
			return false, nil
		}

		return true, nil
	}

	//init buffer and decoder for unmarshal recursiv xml
	ruleBuffer := bytes.NewBuffer(rule.Filter.XMLContent) //check only the filter tag/content
	dececoder := xml.NewDecoder(ruleBuffer)

	//start node
	var node recursiveNode
	err := dececoder.Decode(&node)
	if err != nil {
		return false, err
	}

	foundMappingFilter := true

	walk([]recursiveNode{node}, &node, func(node recursiveNode) bool {
		if node.XMLName.Local == "PropertyName" {
			if string(node.Content) == mappingValueColumnName {

				if node.ParentNode != nil {
					if functions.StringInSlice(node.ParentNode.XMLName.Local, nonExplicitFilteringOperators) {

						if node.ParentNode.XMLName.Local == "PropertyIsNotEqualTo" {

							for _, adjacentNode := range node.ParentNode.Nodes {
								if adjacentNode.XMLName.Local == "Literal" {
									newLiteralName := string(adjacentNode.Content)
									*implicitFilteredValueList = append(*implicitFilteredValueList, string(node.Content)+":"+newLiteralName)
								}
							}
						}

						foundMappingFilter = false
						return false
					} else if node.ParentNode.XMLName.Local == "Function" {
						for _, attr := range node.ParentNode.Attrs {
							if attr.Name.Local == "name" {
								if functions.StringInSlice(attr.Value, nonExplicitFilteringFunctions) {
									foundMappingFilter = false
									return false
								}
							}
						}
					}
				}

				return true
			}
		}

		return true
	})

	return foundMappingFilter, nil
}

//Node Structure
//- XMLName: Name of the XML Object
//- Attrs: An Array of XML Attributes -> class="test"
//- Content: The full Content as Byte Array
//- Nodes: All Childnode as an Array of recursiveNode Object
//- ParentNode: an recursiveNode Object representing the Parent Node of the current Node
type recursiveNode struct {
	XMLName    xml.Name
	Attrs      []xml.Attr      `xml:"-"`
	Content    []byte          `xml:",innerxml"`
	Nodes      []recursiveNode `xml:",any"`
	ParentNode *recursiveNode  `xml:"-"`
}

//Walk function for recursivly going trow the recursiveNode Objects
//- nodes: Array of nodes to be iterated through
//- parentNode: The Parent Node of the current Nodes
//- f: a cancel function, which stops the recursion of the current node. false = stop, true = continue
func walk(nodes []recursiveNode, parentNode *recursiveNode, f func(recursiveNode) bool) {
	for _, childNode := range nodes {

		childNode.ParentNode = parentNode

		if f(childNode) {
			walk(childNode.Nodes, &childNode, f)
		}
	}
}

//Override UnmarshalXML function
func (r_Node *recursiveNode) UnmarshalXML(decoder *xml.Decoder, start xml.StartElement) error {

	//get XML Attributs
	r_Node.Attrs = start.Attr

	type node recursiveNode

	return decoder.DecodeElement((*node)(r_Node), &start)
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

//ColumnInColumnlist checks if a list of required columns contains a specific column. The check is only performed using the column name
func ColumnInColumnlist(columnName string, columnList []RequiredColumn) (bool, int) {

	for i, column := range columnList {
		if columnName == column.PropertyName {
			return true, i
		}
	}

	return false, -1
}
