package models

type ServerInfoReply struct {
	Timezone            string `json:"timezone"`
	TreninghetenVersion string `json:"treningheten_version"`
}
