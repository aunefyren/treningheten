package models

type ConfigStruct struct {
	Timezone                string `json:"timezone"`
	PrivateKey              string `json:"private_key"`
	DBUsername              string `json:"db_username"`
	DBPassword              string `json:"db_password"`
	DBName                  string `json:"db_name"`
	DBIP                    string `json:"db_ip"`
	DBPort                  int    `json:"db_port"`
	TreninghetenPort        int    `json:"treningheten_port"`
	TreninghetenName        string `json:"treningheten_name"`
	TreninghetenExternalURL string `json:"treningheten_external_url"`
	TreninghetenVersion     string `json:"treningheten_version"`
	TreninghetenEnvironment string `json:"treningheten_environment"`
	TreninghetenTestEmail   string `json:"treningheten_test_email"`
	SMTPEnabled             bool   `json:"smtp_enabled"`
	SMTPHost                string `json:"smtp_host"`
	SMTPPort                int    `json:"smtp_port"`
	SMTPUsername            string `json:"smtp_username"`
	SMTPPassword            string `json:"smtp_password"`
	SMTPFrom                string `json:"smtp_from"`
	VAPIDPublicKey          string `json:"vapid_publickey"`
	VAPIDSecretKey          string `json:"vapid_secretkey"`
	VAPIDContact            string `json:"vapid_contact"`
}

type VAPIDSettings struct {
	VAPIDPublicKey string `json:"vapid_publickey"`
	VAPIDSecretKey string `json:"vapid_secretkey"`
	VAPIDContact   string `json:"vapid_contact"`
}
