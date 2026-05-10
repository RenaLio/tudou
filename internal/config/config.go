package config

type Config struct {
	Env   string `mapstructure:"env"` // prod,dev,test,local
	Debug struct {
		Default bool `mapstructure:"default"`
		DB      bool `mapstructure:"db"`
	} `mapstructure:"debug"`
	Http struct {
		Host      string `mapstructure:"host"`
		Port      int    `mapstructure:"port"`
		RateLimit struct {
			Enabled       bool    `mapstructure:"enabled"`
			GlobalEnabled bool    `mapstructure:"global_enabled"`
			GlobalRPS     float64 `mapstructure:"global_rps"`
			GlobalBurst   int     `mapstructure:"global_burst"`
			IPEnabled     bool    `mapstructure:"ip_enabled"`
			IPRPS         float64 `mapstructure:"ip_rps"`
			IPBurst       int     `mapstructure:"ip_burst"`
			IPTTLMinutes  int     `mapstructure:"ip_ttl_minutes"`
		} `mapstructure:"rate_limit"`
		Gzip struct {
			Enabled bool `mapstructure:"enabled"`
			Level   int  `mapstructure:"level"`
		} `mapstructure:"gzip"`
	} `mapstructure:"http"`
	Security struct {
		JWT struct {
			Secret string `mapstructure:"secret"`
		} `mapstructure:"jwt"`
		Sid struct {
			Id int64 `mapstructure:"id"`
		} `mapstructure:"sid"`
	} `mapstructure:"security"`
	Data struct {
		DB struct {
			User struct {
				Driver string `mapstructure:"driver"`
				DSN    string `mapstructure:"dsn"`
			} `mapstructure:"user"`
		} `mapstructure:"db"`
	} `mapstructure:"data"`
	Log struct {
		Level           string `mapstructure:"log_level"`
		Mode            string `mapstructure:"mode"` // console or file or both
		FileEncoding    string `mapstructure:"file_encoding"`
		ConsoleEncoding string `mapstructure:"console_encoding"`
		Encoding        string `mapstructure:"encoding"`
		LogPath         string `mapstructure:"log_path"`
		FileName        string `mapstructure:"log_file_name"`
		ErrorFileName   string `mapstructure:"error_file_name"`
		MaxBackups      int    `mapstructure:"max_backups"`
		MaxAge          int    `mapstructure:"max_age"`
		MaxSize         int    `mapstructure:"max_size"`
		Compress        bool   `mapstructure:"compress"`
	} `mapstructure:"log"`
}
