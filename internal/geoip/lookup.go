package geoip

import (
	"net"
	"github.com/oschwald/geoip2-golang"
)

type Database struct {
	reader *geoip2.Reader
}

func Open(path string) (*Database, error) {
	r, err := geoip2.Open(path)
	if err != nil {
		return nil, err
	}
	return &Database{reader: r}, nil
}

func (d *Database) Lookup(ipStr string) string {
	if d == nil || d.reader == nil {
		return "N/A"
	}
	
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return "INVALID_IP"
	}

	record, err := d.reader.Country(ip)
	if err != nil || record.Country.IsoCode == "" {
		return "UNKNOWN"
	}

	return record.Country.IsoCode
}

func (d *Database) Close() {
	d.reader.Close()
}