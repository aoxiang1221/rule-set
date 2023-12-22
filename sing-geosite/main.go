package main

import (
	"flag"
	"os"
	"path/filepath"
	"strings"

	"github.com/sagernet/sing-box/common/geosite"
	"github.com/sagernet/sing-box/common/srs"
	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/option"
	"github.com/sagernet/sing/common"

	"github.com/sagernet/sing-box/log"
	"github.com/v2fly/v2ray-core/v5/app/router/routercommon"
	"google.golang.org/protobuf/proto"
)

func parse(vGeositeData []byte) (map[string][]geosite.Item, error) {
	vGeositeList := routercommon.GeoSiteList{}
	err := proto.Unmarshal(vGeositeData, &vGeositeList)
	if err != nil {
		return nil, err
	}
	domainMap := make(map[string][]geosite.Item)
	for _, vGeositeEntry := range vGeositeList.Entry {
		code := strings.ToLower(vGeositeEntry.CountryCode)
		domains := make([]geosite.Item, 0, len(vGeositeEntry.Domain)*2)
		attributes := make(map[string][]*routercommon.Domain)
		for _, domain := range vGeositeEntry.Domain {
			if len(domain.Attribute) > 0 {
				for _, attribute := range domain.Attribute {
					attributes[attribute.Key] = append(attributes[attribute.Key], domain)
				}
			}
			switch domain.Type {
			case routercommon.Domain_Plain:
				domains = append(domains, geosite.Item{
					Type:  geosite.RuleTypeDomainKeyword,
					Value: domain.Value,
				})
			case routercommon.Domain_Regex:
				domains = append(domains, geosite.Item{
					Type:  geosite.RuleTypeDomainRegex,
					Value: domain.Value,
				})
			case routercommon.Domain_RootDomain:
				if strings.Contains(domain.Value, ".") {
					domains = append(domains, geosite.Item{
						Type:  geosite.RuleTypeDomain,
						Value: domain.Value,
					})
				}
				domains = append(domains, geosite.Item{
					Type:  geosite.RuleTypeDomainSuffix,
					Value: "." + domain.Value,
				})
			case routercommon.Domain_Full:
				domains = append(domains, geosite.Item{
					Type:  geosite.RuleTypeDomain,
					Value: domain.Value,
				})
			}
		}
		domainMap[code] = common.Uniq(domains)
		for attribute, attributeEntries := range attributes {
			attributeDomains := make([]geosite.Item, 0, len(attributeEntries)*2)
			for _, domain := range attributeEntries {
				switch domain.Type {
				case routercommon.Domain_Plain:
					attributeDomains = append(attributeDomains, geosite.Item{
						Type:  geosite.RuleTypeDomainKeyword,
						Value: domain.Value,
					})
				case routercommon.Domain_Regex:
					attributeDomains = append(attributeDomains, geosite.Item{
						Type:  geosite.RuleTypeDomainRegex,
						Value: domain.Value,
					})
				case routercommon.Domain_RootDomain:
					if strings.Contains(domain.Value, ".") {
						attributeDomains = append(attributeDomains, geosite.Item{
							Type:  geosite.RuleTypeDomain,
							Value: domain.Value,
						})
					}
					attributeDomains = append(attributeDomains, geosite.Item{
						Type:  geosite.RuleTypeDomainSuffix,
						Value: "." + domain.Value,
					})
				case routercommon.Domain_Full:
					attributeDomains = append(attributeDomains, geosite.Item{
						Type:  geosite.RuleTypeDomain,
						Value: domain.Value,
					})
				}
			}
			domainMap[code+"@"+attribute] = common.Uniq(attributeDomains)
		}
	}
	return domainMap, nil
}

func generate(input string, output string, ruleSetOutput string, logflag bool) error {
	outputFile, err := os.Create(output)
	if err != nil {
		return err
	}
	defer outputFile.Close()
	vData, err := os.ReadFile(input)
	if err != nil {
		return err
	}
	domainMap, err := parse(vData)
	if err != nil {
		return err
	}

	if logflag {
		outputPath, _ := filepath.Abs(output)
		os.Stderr.WriteString("write " + outputPath + "\n")
	}

	err = geosite.Write(outputFile, domainMap)
	if err != nil {
		return err
	}
	os.RemoveAll(ruleSetOutput)
	err = os.MkdirAll(ruleSetOutput, 0o755)
	if err != nil {
		return err
	}
	for code, domains := range domainMap {
		var headlessRule option.DefaultHeadlessRule
		defaultRule := geosite.Compile(domains)
		headlessRule.Domain = defaultRule.Domain
		headlessRule.DomainSuffix = defaultRule.DomainSuffix
		headlessRule.DomainKeyword = defaultRule.DomainKeyword
		headlessRule.DomainRegex = defaultRule.DomainRegex
		var plainRuleSet option.PlainRuleSet
		plainRuleSet.Rules = []option.HeadlessRule{
			{
				Type:           C.RuleTypeDefault,
				DefaultOptions: headlessRule,
			},
		}
		srsPath, _ := filepath.Abs(filepath.Join(ruleSetOutput, "geosite-"+code+".srs"))

		if logflag {
			os.Stderr.WriteString("write " + srsPath + "\n")
		}

		outputRuleSet, err := os.Create(srsPath)
		if err != nil {
			return err
		}
		err = srs.Write(outputRuleSet, plainRuleSet)
		if err != nil {
			outputRuleSet.Close()
			return err
		}
		outputRuleSet.Close()
	}
	return nil
}

func release(input string, output string, ruleset string, logflag bool) error {
	err := generate(input, output, ruleset, logflag)
	if err != nil {
		return err
	}

	absInput, err := filepath.Abs(input)
	if err != nil {
		return err
	}
	absOutput, err := filepath.Abs(output)
	if err != nil {
		return err
	}
	absRuleSet, err := filepath.Abs(ruleset)
	if err != nil {
		return err
	}

	os.Stdout.WriteString("Input file: " + absInput + "\n")
	os.Stdout.WriteString("Output file: " + absOutput + "\n")
	os.Stdout.WriteString("Rule-set output: " + absRuleSet + "\n")

	return nil
}

var (
	inputFile  = flag.String("i", "geosite.dat", "input geosite.bat file")
	outputFile = flag.String("o", "geosite.db", "output geosite.db file")
	ruleSet    = flag.String("r", "rule-set", "rule-set path")
	logFlag    = flag.Bool("log", false, "whether to log file paths")
)

func main() {
	flag.Parse()
	err := release(*inputFile, *outputFile, *ruleSet, *logFlag)
	if err != nil {
		log.Fatal(err)
	}
}
