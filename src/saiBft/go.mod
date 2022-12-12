module github.com/iamthe1whoknocks/bft

go 1.18

replace github.com/iamthe1whoknocks/saiStorage/internal => /internal

replace github.com/iamthe1whoknocks/saiService => ../saiService

require go.uber.org/zap v1.23.0

require (
	github.com/cpuguy83/go-md2man/v2 v2.0.2 // indirect
	github.com/golang/snappy v0.0.1 // indirect
	github.com/klauspost/compress v1.13.6 // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/urfave/cli/v2 v2.11.1 // indirect
	github.com/xrash/smetrics v0.0.0-20201216005158-039620a65673 // indirect
	github.com/youmark/pkcs8 v0.0.0-20181117223130-1be2e3e5546d // indirect
	golang.org/x/crypto v0.0.0-20220622213112-05595931fe9d // indirect
	golang.org/x/net v0.0.0-20211112202133-69e39bad7dc2 // indirect
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

require (
	github.com/asaskevich/govalidator v0.0.0-20210307081110-f21760c49a8d
	github.com/iamthe1whoknocks/saiService v0.0.0-00010101000000-000000000000
	go.mongodb.org/mongo-driver v1.10.2
	go.uber.org/atomic v1.7.0 // indirect
	go.uber.org/multierr v1.6.0 // indirect
)
