module argus.domain

go 1.15

require (
	github.com/ChrisTrenkamp/goxpath v0.0.0-20210404020558-97928f7e12b6
	github.com/SAP/go-hdb v0.105.5
	github.com/denisenkom/go-mssqldb v0.12.0
	github.com/go-sql-driver/mysql v1.6.0
	github.com/golang/mock v1.6.0 // indirect
	github.com/jaw0/acgo v0.0.0-20200108170510-4e35a23083f1
	github.com/lib/pq v1.10.4
	github.com/miekg/dns v1.1.47
	github.com/mitchellh/mapstructure v1.4.3
	github.com/oliveagle/jsonpath v0.0.0-20180606110733-2e52cf6e6852
	// NB - the maintainer of this changed. the new "community" version is less good
	//github.com/soniah/gosnmp v0.0.0-20180902041332-35d70ef64360	// last version well tested
	github.com/soniah/gosnmp v0.0.0-20201008234443-3ac04c460252	// last version from maintainer=soniah
	golang.org/x/crypto v0.0.0-20220315160706-3147a52a75dd
)

