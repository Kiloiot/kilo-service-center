module github.com/Kiloiot/kilo-service-center/KC-MQTT

go 1.27.3

require (
	github.com/Kiloiot/kilo-service-center/KC-Core v0.1.0
	github.com/Kiloiot/kilo-service-center/KC-DB v0.1.0
	github.com/Kiloiot/kilo-service-center/pkg v0.0.0-00010101000000-000000000000
	github.com/eclipse/paho.mqtt.golang v1.5.1
	github.com/google/uuid v1.6.0
	github.com/stretchr/testify v1.12.1
)

require (
	github.com/fsnotify/fsnotify v1.10.1 // indirect
	github.com/go-viper/mapstructure/v2 v2.5.0 // indirect
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/gorilla/websocket v1.5.3 // indirect
	github.com/pelletier/go-toml/v2 v2.3.1 // indirect
	github.com/rogpeppe/go-internal v1.14.1 // indirect
	github.com/sagikazarmark/locafero v0.12.0 // indirect
	github.com/spf13/afero v1.15.0 // indirect
	github.com/spf13/cast v1.10.0 // indirect
	github.com/spf13/pflag v1.0.10 // indirect
	github.com/spf13/viper v1.21.0 // indirect
	github.com/subosito/gotenv v1.6.0 // indirect
	github.com/vmihailenco/msgpack/v5 v5.4.1 // indirect
	github.com/vmihailenco/tagparser/v2 v2.0.0 // indirect
	go.yaml.in/yaml/v3 v3.0.5 // indirect
	golang.org/x/net v0.62.0 // indirect
	golang.org/x/sync v0.24.0 // indirect
	golang.org/x/sys v0.49.0 // indirect
	golang.org/x/text v0.43.0 // indirect
)

replace (
	github.com/Kiloiot/kilo-service-center/KC-Core => ../KC-Core
	github.com/Kiloiot/kilo-service-center/KC-DB => ../KC-DB
	github.com/Kiloiot/kilo-service-center/pkg => ../pkg
)
