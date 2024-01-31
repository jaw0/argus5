module argus.domain

go 1.18

require (
	github.com/ChrisTrenkamp/goxpath v0.0.0-20210404020558-97928f7e12b6
	github.com/SAP/go-hdb v0.105.5
	github.com/denisenkom/go-mssqldb v0.12.0
	github.com/go-sql-driver/mysql v1.6.0
	github.com/jaw0/acconfig v0.0.0-20240129152758-958dc821d238
	github.com/jaw0/acdiag v0.0.0-20240130233029-909cb48df08b
	github.com/jaw0/go-daemon v0.0.0-20240130233342-2b87370ebf19
	github.com/lib/pq v1.10.4
	github.com/miekg/dns v1.1.47
	github.com/mitchellh/mapstructure v1.4.3
	github.com/oliveagle/jsonpath v0.0.0-20180606110733-2e52cf6e6852
	// NB - the maintainer of this changed. the new "community" version is less good
	//github.com/soniah/gosnmp v0.0.0-20180902041332-35d70ef64360	// last version well tested
	github.com/soniah/gosnmp v0.0.0-20201008234443-3ac04c460252 // last version from maintainer=soniah
	golang.org/x/crypto v0.0.0-20220315160706-3147a52a75dd
)

require (
	github.com/golang-sql/civil v0.0.0-20190719163853-cb61b32ac6fe // indirect
	github.com/golang-sql/sqlexp v0.0.0-20170517235910-f1bb20e5a188 // indirect
	golang.org/x/mod v0.4.2 // indirect
	golang.org/x/net v0.0.0-20211112202133-69e39bad7dc2 // indirect
	golang.org/x/sys v0.0.0-20210630005230-0f9fa26af87c // indirect
	golang.org/x/text v0.3.7 // indirect
	golang.org/x/tools v0.1.6-0.20210726203631-07bc1bf47fb2 // indirect
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1 // indirect
)
