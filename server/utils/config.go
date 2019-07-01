package utils

type DatabaseConfig struct {
	DriverName     string
	DataSourceName string
	MaxConnectNum  int
}

type OssConfig struct {
	Region string
	Accesskeyid string
    Accesskeysecret string
    Bucket string
}


