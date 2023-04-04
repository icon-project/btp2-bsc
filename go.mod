module github.com/icon-project/btp2/bsc

replace (
	github.com/ethereum/go-ethereum => github.com/bnb-chain/bsc v1.1.18
	github.com/icon-project/btp2 => /Users/cornery/workspace/github/btp2.git
	github.com/tendermint/tendermint => github.com/bnb-chain/tendermint v0.31.12
)

go 1.17

require (
	github.com/ethereum/go-ethereum v1.10.5
	github.com/hashicorp/golang-lru v0.5.5-0.20210104140557-80c98217689d
	//github.com/icon-project/btp v0.0.0-20230302061923-57e466fc6308
	// github.com/icon-project/btp v0.0.0-20230307085542-82e463e69797
	github.com/spf13/cobra v1.6.1
	golang.org/x/crypto v0.0.0-20220525230936-793ad666bf5e
)

require (
	github.com/icon-project/btp v0.0.0-20230307085542-82e463e69797
	github.com/icon-project/btp2 v0.0.0-20230315065508-126354803af1
)

require (
	github.com/AndreasBriese/bbloom v0.0.0-20190306092124-e2d15f34fcf9 // indirect
	github.com/StackExchange/wmi v0.0.0-20180116203802-5d049714c4a6 // indirect
	github.com/VictoriaMetrics/fastcache v1.6.0 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/bshuster-repo/logrus-logstash-hook v0.4.1 // indirect
	github.com/btcsuite/btcd v0.20.1-beta // indirect
	github.com/cespare/xxhash/v2 v2.1.2 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/deckarep/golang-set v1.8.0 // indirect
	github.com/dgraph-io/badger v1.5.4 // indirect
	github.com/dgryski/go-farm v0.0.0-20190416075124-e1214b5e05dc // indirect
	github.com/etcd-io/bbolt v1.3.3 // indirect
	github.com/evalphobia/logrus_fluent v0.5.4 // indirect
	github.com/fluent/fluent-logger-golang v1.4.0 // indirect
	github.com/fsnotify/fsnotify v1.6.0 // indirect
	github.com/go-kit/kit v0.9.0 // indirect
	github.com/go-logfmt/logfmt v0.5.1 // indirect
	github.com/go-ole/go-ole v1.2.1 // indirect
	github.com/go-stack/stack v1.8.0 // indirect
	github.com/gofrs/uuid v3.3.0+incompatible // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/golang/snappy v0.0.4 // indirect
	github.com/google/uuid v1.3.0 // indirect
	github.com/gorilla/websocket v1.4.2 // indirect
	github.com/haltingstate/secp256k1-go v0.0.0-20151224084235-572209b26df6 // indirect
	github.com/hashicorp/hcl v1.0.0 // indirect
	github.com/holiman/bloomfilter/v2 v2.0.3 // indirect
	github.com/holiman/uint256 v1.2.0 // indirect
	github.com/inconshreveable/mousetrap v1.0.1 // indirect
	github.com/jmhodges/levigo v1.0.0 // indirect
	github.com/jroimartin/gocui v0.4.0 // indirect
	github.com/konsorten/go-windows-terminal-sequences v1.0.3 // indirect
	github.com/labstack/echo/v4 v4.9.0 // indirect
	github.com/labstack/gommon v0.3.1 // indirect
	github.com/magiconair/properties v1.8.7 // indirect
	github.com/mattn/go-colorable v0.1.12 // indirect
	github.com/mattn/go-isatty v0.0.14 // indirect
	github.com/mattn/go-runewidth v0.0.9 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.1 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/nsf/termbox-go v0.0.0-20190325093121-288510b9734e // indirect
	github.com/olekukonko/tablewriter v0.0.5 // indirect
	github.com/panjf2000/ants/v2 v2.4.5 // indirect
	github.com/pelletier/go-toml v1.9.5 // indirect
	github.com/pelletier/go-toml/v2 v2.0.6 // indirect
	github.com/philhofer/fwd v1.0.0 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/prometheus/client_golang v1.13.0 // indirect
	github.com/prometheus/client_model v0.2.0 // indirect
	github.com/prometheus/common v0.37.0 // indirect
	github.com/prometheus/procfs v0.8.0 // indirect
	github.com/prometheus/tsdb v0.7.1 // indirect
	github.com/rcrowley/go-metrics v0.0.0-20190826022208-cac0b30c2563 // indirect
	github.com/rjeczalik/notify v0.9.1 // indirect
	github.com/rs/cors v1.7.0 // indirect
	github.com/shirou/gopsutil v3.21.4-0.20210419000835-c7a38de76ee5+incompatible // indirect
	github.com/sirupsen/logrus v1.6.0 // indirect
	github.com/spf13/afero v1.9.3 // indirect
	github.com/spf13/cast v1.5.0 // indirect
	github.com/spf13/jwalterweatherman v1.1.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/spf13/viper v1.14.0 // indirect
	github.com/subosito/gotenv v1.4.1 // indirect
	github.com/syndtr/goleveldb v1.0.1-0.20210819022825-2ae1ddf74ef7 // indirect
	github.com/tendermint/go-amino v0.14.1 // indirect
	github.com/tendermint/iavl v0.12.0 // indirect
	github.com/tendermint/tendermint v0.31.12 // indirect
	github.com/tinylib/msgp v1.1.2 // indirect
	github.com/tklauser/go-sysconf v0.3.5 // indirect
	github.com/tklauser/numcpus v0.2.2 // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	github.com/valyala/fasttemplate v1.2.1 // indirect
	github.com/vmihailenco/msgpack/v4 v4.3.11 // indirect
	github.com/vmihailenco/tagparser v0.1.1 // indirect
	golang.org/x/net v0.7.0 // indirect
	golang.org/x/sys v0.5.0 // indirect
	golang.org/x/text v0.7.0 // indirect
	google.golang.org/appengine v1.6.7 // indirect
	google.golang.org/genproto v0.0.0-20221024183307-1bc688fe9f3e // indirect
	google.golang.org/grpc v1.50.1 // indirect
	google.golang.org/protobuf v1.28.1 // indirect
	gopkg.in/ini.v1 v1.67.0 // indirect
	gopkg.in/natefinch/lumberjack.v2 v2.0.0 // indirect
	gopkg.in/natefinch/npipe.v2 v2.0.0-20160621034901-c1b8fa8bdcce // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
