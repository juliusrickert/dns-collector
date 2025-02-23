package subprocessors

import (
	"bufio"
	"os"
	"regexp"
	"strings"

	"github.com/dmachard/go-dnscollector/dnsutils"
	"github.com/dmachard/go-logger"
)

type FilteringProcessor struct {
	config           *dnsutils.Config
	logger           *logger.Logger
	dropDomains      bool
	listQueryIp      map[string]bool
	listFqdns        map[string]bool
	listDomainsRegex map[string]*regexp.Regexp
}

func NewFilteringProcessor(config *dnsutils.Config, logger *logger.Logger) FilteringProcessor {
	d := FilteringProcessor{
		config:           config,
		logger:           logger,
		listFqdns:        make(map[string]bool),
		listDomainsRegex: make(map[string]*regexp.Regexp),
	}

	d.LoadDomainsList()
	d.LoadQueryIpList()

	return d
}

func (p *FilteringProcessor) LoadQueryIpList() {
	if len(p.config.Subprocessors.Filtering.DropQueryIpFile) > 0 {
		file, err := os.Open(p.config.Subprocessors.Filtering.DropQueryIpFile)
		if err != nil {
			p.LogError("unable to open query ip file: ", err)
		} else {
			scanner := bufio.NewScanner(file)
			for scanner.Scan() {
				queryip := strings.ToLower(scanner.Text())
				p.listQueryIp[queryip] = true
			}
			p.LogInfo("loaded with %d query ip to the drop list", len(p.listQueryIp))
		}

	}
}
func (p *FilteringProcessor) LoadDomainsList() {

	if len(p.config.Subprocessors.Filtering.DropFqdnFile) > 0 {
		file, err := os.Open(p.config.Subprocessors.Filtering.DropFqdnFile)
		if err != nil {
			p.LogError("unable to open fqdn file: ", err)
			p.dropDomains = true
		} else {
			scanner := bufio.NewScanner(file)
			for scanner.Scan() {
				fqdn := strings.ToLower(scanner.Text())
				p.listFqdns[fqdn] = true
			}
			p.LogInfo("loaded with %d fqdn to the drop list", len(p.listFqdns))
			p.dropDomains = true
		}

	}

	if len(p.config.Subprocessors.Filtering.DropDomainFile) > 0 {
		file, err := os.Open(p.config.Subprocessors.Filtering.DropDomainFile)
		if err != nil {
			p.LogError("unable to open regex list file: ", err)
			p.dropDomains = true
		} else {
			scanner := bufio.NewScanner(file)
			for scanner.Scan() {
				domain := strings.ToLower(scanner.Text())
				p.listDomainsRegex[domain] = regexp.MustCompile(domain)
			}
			p.LogInfo("loaded with %d domains to the drop list", len(p.listDomainsRegex))
			p.dropDomains = true
		}

	}
}

func (p *FilteringProcessor) LogInfo(msg string, v ...interface{}) {
	p.logger.Info("filtering - "+msg, v...)
}

func (p *FilteringProcessor) LogError(msg string, v ...interface{}) {
	p.logger.Error("filtering - "+msg, v...)
}

func (p *FilteringProcessor) CheckIfDrop(dm *dnsutils.DnsMessage) bool {
	// ignore queries ?
	if !p.config.Subprocessors.Filtering.LogQueries && dm.DNS.Type == dnsutils.DnsQuery {
		return true
	}

	// ignore replies ?
	if !p.config.Subprocessors.Filtering.LogReplies && dm.DNS.Type == dnsutils.DnsReply {
		return true
	}

	// drop according to the rcode ?
	for _, v := range p.config.Subprocessors.Filtering.DropRcodes {
		if v == dm.DNS.Rcode {
			return true
		}
	}

	// drop according to the query ip ?
	if len(p.listQueryIp) > 0 {
		for k := range p.listQueryIp {
			if dm.NetworkInfo.QueryIp == k {
				return true
			}
		}
	}

	// drop domains ?
	if p.dropDomains {
		// fqdn
		for k := range p.listFqdns {
			if dm.DNS.Qname == k {
				return true
			}
		}
		// partiel fqdn with regexp
		for _, p := range p.listDomainsRegex {
			if p.MatchString(dm.DNS.Qname) {
				return true
			}
		}
	}

	return false
}
